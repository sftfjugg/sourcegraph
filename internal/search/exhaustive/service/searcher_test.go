package service_test

import (
	"testing"

	"github.com/sourcegraph/log/logtest"
	"github.com/sourcegraph/sourcegraph/internal/database/dbmocks"
	searchbackend "github.com/sourcegraph/sourcegraph/internal/search/backend"
	"github.com/sourcegraph/sourcegraph/internal/search/client"
	"github.com/sourcegraph/sourcegraph/internal/search/exhaustive/service"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/zoekt"
)

func TestFromSearchClient(t *testing.T) {
	mock := mockSearchClient(t, []string{"1", "2"})

	testNewSearcher(t, service.FromSearchClient(mock), newSearcherTestCase{
		Query: "type:file index:no content",
	})
}

// mockSearchClient returns a client which will return matches. This exercises
// more of the search code path to give a bit more confidence we are correctly
// calling Plan and Execute vs a dumb SearchClient mock.
//
// Note: for now we only support nicely mocking zoekt. This isn't good enough
// to gain confidence in how this all works, so will follow up with making it
// possible to mock searcher.
func mockSearchClient(t testing.TB, repoNames []string) client.SearchClient {
	repos := dbmocks.NewMockRepoStore()
	repos.ListMinimalReposFunc.SetDefaultReturn([]types.MinimalRepo{}, nil)
	repos.CountFunc.SetDefaultReturn(0, nil)

	db := dbmocks.NewMockDB()
	db.ReposFunc.SetDefaultReturn(repos)

	var matches []zoekt.FileMatch
	for i, name := range repoNames {
		matches = append(matches, zoekt.FileMatch{
			RepositoryID: uint32(i),
			Repository:   name,
		})
	}
	mockZoekt := &searchbackend.FakeStreamer{
		Repos: []*zoekt.RepoListEntry{},
		Results: []*zoekt.SearchResult{{
			Files: matches,
		}},
	}

	return client.MockedZoekt(logtest.Scoped(t), db, mockZoekt)
}
