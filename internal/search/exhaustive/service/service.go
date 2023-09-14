package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/sourcegraph/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sourcegraph/sourcegraph/internal/actor"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/search/exhaustive/store"
	"github.com/sourcegraph/sourcegraph/internal/search/exhaustive/types"
	"github.com/sourcegraph/sourcegraph/internal/uploadstore"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/sourcegraph/lib/iterator"
)

// New returns a Service.
func New(observationCtx *observation.Context, store *store.Store, uploadStore uploadstore.Store) *Service {
	logger := log.Scoped("searchjobs.Service", "search job service")
	svc := &Service{
		logger:      logger,
		store:       store,
		uploadStore: uploadStore,
		operations:  newOperations(observationCtx),
	}

	return svc
}

type Service struct {
	logger      log.Logger
	store       *store.Store
	uploadStore uploadstore.Store
	operations  *operations
}

func opAttrs(attrs ...attribute.KeyValue) observation.Args {
	return observation.Args{Attrs: attrs}
}

type operations struct {
	createSearchJob *observation.Operation
	getSearchJob    *observation.Operation
	listSearchJobs  *observation.Operation
	cancelSearchJob *observation.Operation
	copyBlobs       *observation.Operation
}

var (
	singletonOperations *operations
	operationsOnce      sync.Once
)

// newOperations generates a singleton of the operations struct.
//
// TODO: We should create one per observationCtx. This is a copy-pasta from
// the batches service, we should validate if we need to do this protection.
func newOperations(observationCtx *observation.Context) *operations {
	operationsOnce.Do(func() {
		m := metrics.NewREDMetrics(
			observationCtx.Registerer,
			"searchjobs_service",
			metrics.WithLabels("op"),
			metrics.WithCountHelp("Total number of method invocations."),
		)

		op := func(name string) *observation.Operation {
			return observationCtx.Operation(observation.Op{
				Name:              fmt.Sprintf("searchjobs.service.%s", name),
				MetricLabelValues: []string{name},
				Metrics:           m,
			})
		}

		singletonOperations = &operations{
			createSearchJob: op("CreateSearchJob"),
			getSearchJob:    op("GetSearchJob"),
			listSearchJobs:  op("ListSearchJobs"),
			cancelSearchJob: op("CancelSearchJob"),
			copyBlobs:       op("CopyBlobs"),
		}
	})
	return singletonOperations
}

func (s *Service) CreateSearchJob(ctx context.Context, query string) (_ *types.ExhaustiveSearchJob, err error) {
	ctx, _, endObservation := s.operations.createSearchJob.With(ctx, &err, opAttrs(
		attribute.String("query", query),
	))
	defer endObservation(1, observation.Args{})

	actor := actor.FromContext(ctx)
	if !actor.IsAuthenticated() {
		return nil, errors.New("search jobs can only be created by an authenticated user")
	}

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = tx.Done(err) }()

	// XXX(keegancsmith) this API for creating seems easy to mess up since the
	// ExhaustiveSearchJob type has lots of fields, but reading the store
	// implementation only two fields are read.
	jobID, err := tx.CreateExhaustiveSearchJob(ctx, types.ExhaustiveSearchJob{
		InitiatorID: actor.UID,
		Query:       query,
	})
	if err != nil {
		return nil, err
	}

	return tx.GetExhaustiveSearchJob(ctx, jobID)
}

func (s *Service) CancelSearchJob(ctx context.Context, id int64) (err error) {
	ctx, _, endObservation := s.operations.cancelSearchJob.With(ctx, &err, opAttrs(
		attribute.Int64("id", id),
	))
	defer endObservation(1, observation.Args{})

	tx, err := s.store.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	_, err = tx.CancelSearchJob(ctx, id)
	return err
}

func (s *Service) GetSearchJob(ctx context.Context, id int64) (_ *types.ExhaustiveSearchJob, err error) {
	ctx, _, endObservation := s.operations.getSearchJob.With(ctx, &err, opAttrs(
		attribute.Int64("id", id),
	))
	defer endObservation(1, observation.Args{})

	return s.store.GetExhaustiveSearchJob(ctx, id)
}

func (s *Service) ListSearchJobs(ctx context.Context) (jobs []*types.ExhaustiveSearchJob, err error) {
	ctx, _, endObservation := s.operations.listSearchJobs.With(ctx, &err, observation.Args{})
	defer func() {
		endObservation(1, opAttrs(
			attribute.Int("len", len(jobs)),
		))
	}()

	return s.store.ListExhaustiveSearchJobs(ctx)
}

func getPrefix(id int64) string {
	return fmt.Sprintf("%d-", id)
}

// CopyBlobs copies all the blobs associated with a search job to the given writer.
func (s *Service) CopyBlobs(ctx context.Context, w io.Writer, id int64) (err error) {
	ctx, _, endObservation := s.operations.copyBlobs.With(ctx, &err, opAttrs(
		attribute.Int64("id", id)))
	defer endObservation(1, observation.Args{})

	// 🚨 SECURITY: only someone with access to the job may copy the blobs
	_, err = s.GetSearchJob(ctx, id)
	if err != nil {
		return err
	}

	iter, err := s.uploadStore.List(ctx, getPrefix(id))
	if err != nil {
		return err
	}

	return copyBlobs(ctx, iter, s.uploadStore, w)
}

// discards output from br up until delim is read. If an error is encountered
// it is returned. Note: often the error is io.EOF
func discardUntil(br *bufio.Reader, delim byte) error {
	// This function just wraps ReadSlice which will read until delim. If we
	// get the error ErrBufferFull we didn't find delim since we need to read
	// more, so we just try again. For every other error (or nil) we can
	// return it.
	for {
		_, err := br.ReadSlice(delim)
		if err != bufio.ErrBufferFull {
			return err
		}
	}
}

func copyBlobs(ctx context.Context, iter *iterator.Iterator[string], uploadStore uploadstore.Store, w io.Writer) error {
	// keep a single bufio.Reader so we can reuse its buffer.
	var br bufio.Reader
	copyBlob := func(key string, skipHeader bool) error {
		rc, err := uploadStore.Get(ctx, key)
		if err != nil {
			_ = rc.Close()
			return err
		}
		defer rc.Close()

		br.Reset(rc)

		// skip header line
		if skipHeader {
			err := discardUntil(&br, '\n')
			if err == io.EOF {
				// reached end of file before finding the newline. Write
				// nothing
				return nil
			} else if err != nil {
				return err
			}
		}

		_, err = br.WriteTo(w)
		return err
	}

	// For the first blob we want the header, for the rest we don't
	if iter.Next() {
		key := iter.Current()
		if err := copyBlob(key, false); err != nil {
			return err
		}
	}

	for iter.Next() {
		key := iter.Current()
		if err := copyBlob(key, true); err != nil {
			return err
		}
	}

	return iter.Err()
}
