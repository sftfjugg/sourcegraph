package service

import (
	"context"
	"fmt"

	"github.com/sourcegraph/sourcegraph/internal/search"
	"github.com/sourcegraph/sourcegraph/internal/search/client"
	"github.com/sourcegraph/sourcegraph/internal/search/exhaustive/types"
	"github.com/sourcegraph/sourcegraph/internal/search/job"
	"github.com/sourcegraph/sourcegraph/internal/search/job/jobutil"
	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func FromSearchClient(client client.SearchClient) NewSearcher {
	return newSearcherFunc(func(ctx context.Context, q string) (SearchQuery, error) {
		// TODO adjust NewSearch API to enforce the user passing in a user id.
		// IE do not rely on ctx actor since that could easily lead to a bug.
		inputs, err := client.Plan(
			ctx,
			"V3",
			nil,
			q,
			search.Precise,
			search.Streaming,
		)
		if err != nil {
			return nil, err
		}

		if len(inputs.Plan) != 1 {
			return nil, fmt.Errorf("expected a simple expression (no and/or/etc). Got multiple jobs to run %v", inputs.Plan)
		}

		b := inputs.Plan[0]
		term, ok := b.Pattern.(query.Pattern)
		if !ok {
			return nil, fmt.Errorf("expected a simple expression (no and/or/etc). Got %v", b.Pattern)
		}

		planJob, err := jobutil.NewFlatJob(inputs, query.Flat{Parameters: b.Parameters, Pattern: &term})
		if err != nil {
			return nil, err
		}

		repoPager, ok := planJob.(*jobutil.RepoPagerJob)
		if !ok {
			return nil, fmt.Errorf("internal error: expected a repo pager job when converting plan into search jobs got %T", planJob)
		}

		// TODO should the Plan be parsed differently? Right now by default we
		// will set a timeout and a limit if unspecified by the input query.
		//
		// Additionally by default we try use the index and do things like a
		// RepoSearchJob, ReposComputeExcluded, etc. The shape of our job
		// should be quite different in this use case.

		// Constraints for v0
		//   - type:file only

		// fmt.Println(printer.SexpVerbose(planJob, job.VerbosityMax, true))

		return searchQuery{
			repoPager: repoPager,
			clients:   client.JobClients(),
		}, nil
	})
}

// TODO maybe reuse for the fake
type newSearcherFunc func(context.Context, string) (SearchQuery, error)

func (f newSearcherFunc) NewSearch(ctx context.Context, q string) (SearchQuery, error) {
	return f(ctx, q)
}

type searchQuery struct {
	repoPager *jobutil.RepoPagerJob
	clients   job.RuntimeClients
}

func (s searchQuery) RepositoryRevSpecs(ctx context.Context) ([]types.RepositoryRevSpec, error) {
	var repoRevSpecs []types.RepositoryRevSpec
	it := s.repoPager.RepositoryRevSpecs(ctx, s.clients)
	for it.Next() {
		page := it.Current()
		if page.BackendsMissing > 0 {
			return nil, errors.New("job needs to be retried, some backends are down")
		}
		for _, repoRev := range page.RepoRevs {
			for _, rev := range repoRev.Revs {
				repoRevSpecs = append(repoRevSpecs, types.RepositoryRevSpec{
					Repository:        repoRev.Repo.ID,
					RevisionSpecifier: rev,
				})
			}
		}
	}
	return repoRevSpecs, it.Err()
}

func (s searchQuery) ResolveRepositoryRevSpec(context.Context, types.RepositoryRevSpec) ([]types.RepositoryRevision, error) {
	return nil, nil
}

func (s searchQuery) Search(ctx context.Context, reporev types.RepositoryRevision, w CSVWriter) error {
	//planJob.Run(ctx, s.JobClients(), stream)
	return nil
}
