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

// CopyBlobs copies all the blobs associated with a search job to the given writer.
func (s *Service) CopyBlobs(ctx context.Context, w io.Writer, id int64) error {
	_, err := s.GetSearchJob(ctx, id)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("%d-", id)
	iter, err := s.uploadStore.List(ctx, prefix)
	if err != nil {
		return err
	}

	copyBlob := func(key string, skipHeader bool) error {
		rc, err := s.uploadStore.Get(ctx, key)
		if err != nil {
			_ = rc.Close()
			return err
		}
		defer rc.Close()

		scanner := bufio.NewScanner(rc)

		// skip header line
		if skipHeader && scanner.Scan() {
		}

		for scanner.Scan() {
			// add new line to each row
			_, err = w.Write(scanner.Bytes())
			if err != nil {
				return err
			}
			_, err = w.Write([]byte("\n"))
			if err != nil {
				return err
			}
		}

		return scanner.Err()
	}

	skipHeader := false
	for iter.Next() {
		key := iter.Current()
		if err := copyBlob(key, skipHeader); err != nil {
			return err
		}
		skipHeader = true
	}

	return iter.Err()
}
