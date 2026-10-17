package gh

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestDedupeReposKeepsFirstAndOrder(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	in := []Repo{
		{NameWithOwner: "acme/api", UpdatedAt: base, IsOwner: false, OwnerLogin: "acme"},
		{NameWithOwner: "me/dotfiles", UpdatedAt: base.Add(-time.Hour), IsOwner: true, OwnerLogin: "me"},
		{NameWithOwner: "acme/api", UpdatedAt: base, IsOwner: false, OwnerLogin: "acme"},
		{NameWithOwner: "other/lib", UpdatedAt: base.Add(-2 * time.Hour), OwnerLogin: "other"},
	}

	got := dedupeRepos(in)
	want := []string{"acme/api", "me/dotfiles", "other/lib"}
	if len(got) != len(want) {
		t.Fatalf("dedupeRepos returned %d repos, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].NameWithOwner != name {
			t.Errorf("repo %d = %q, want %q", i, got[i].NameWithOwner, name)
		}
	}
	if !got[1].IsOwner || got[0].IsOwner {
		t.Errorf("IsOwner not preserved: %+v", got)
	}
}

func TestDedupeReposEmpty(t *testing.T) {
	if got := dedupeRepos(nil); len(got) != 0 {
		t.Fatalf("dedupeRepos(nil) = %v, want empty", got)
	}
}

type fakeGitHub func(r *http.Request) (int, string)

func (f fakeGitHub) RoundTrip(r *http.Request) (*http.Response, error) {
	status, body := f(r)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func fakeClient(t *testing.T, serve fakeGitHub) *Client {
	t.Helper()
	opts := api.ClientOptions{Host: "github.com", AuthToken: "x", Transport: serve}
	rest, err := api.NewRESTClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	gql, err := api.NewGraphQLClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	return &Client{rest: rest, gql: gql}
}

func TestFetchReposPagesAndMergesForks(t *testing.T) {
	const total = reposPerPage + 2
	var pages []string
	var mu sync.Mutex
	client := fakeClient(t, func(r *http.Request) (int, string) {
		if r.URL.Path == "/graphql" {
			return 200, `{"data":{"viewer":{"login":"me","repositories":{
				"nodes":[{"nameWithOwner":"me/repo-1","parent":{"nameWithOwner":"up/repo-1"},"primaryLanguage":{"name":"Go"}}],
				"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		mu.Lock()
		pages = append(pages, r.URL.Query().Get("page"))
		mu.Unlock()
		var items []string
		for i := (page - 1) * reposPerPage; i < min(page*reposPerPage, total); i++ {
			owner := "me"
			if i == 2 {
				owner = "org"
			}
			items = append(items, fmt.Sprintf(`{"full_name":"%s/repo-%d","owner":{"login":"%s"},"fork":%t}`, owner, i, owner, i == 1))
		}
		return 200, "[" + strings.Join(items, ",") + "]"
	})

	repos, err := client.FetchRepos()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != total {
		t.Fatalf("got %d repos, want %d", len(repos), total)
	}
	for i, r := range repos {
		if want := fmt.Sprintf("repo-%d", i); r.Name() != want {
			t.Fatalf("repo %d = %s, want %s", i, r.Name(), want)
		}
	}
	if fork := repos[1]; fork.Parent != "up/repo-1" || fork.Language != "Go" || !fork.IsOwner {
		t.Errorf("fork not merged: %+v", fork)
	}
	if repos[2].IsOwner {
		t.Errorf("org repo marked as owned: %+v", repos[2])
	}
	if len(pages) != reposWave {
		t.Errorf("requested pages %v, want one wave of %d", pages, reposWave)
	}
}

func TestFetchStarredLooksUpForks(t *testing.T) {
	var queries []string
	var mu sync.Mutex
	client := fakeClient(t, func(r *http.Request) (int, string) {
		switch {
		case r.URL.Path == "/graphql":
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			queries = append(queries, string(body))
			mu.Unlock()
			return 200, `{"data":{"r0":{"nameWithOwner":"them/fork","parent":{"nameWithOwner":"up/fork"},"primaryLanguage":{"name":"Rust"}}}}`
		case r.URL.Path == "/user":
			return 200, `{"login":"me"}`
		case r.URL.Query().Get("page") == "1":
			return 200, `[{"full_name":"me/mine","owner":{"login":"me"}},{"full_name":"them/fork","owner":{"login":"them"},"fork":true}]`
		}
		return 200, `[]`
	})

	repos, err := client.FetchStarred()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 || !repos[0].IsOwner || repos[1].IsOwner {
		t.Fatalf("starred = %+v", repos)
	}
	if fork := repos[1]; fork.Parent != "up/fork" || fork.Language != "Rust" {
		t.Errorf("fork not merged: %+v", fork)
	}
	if len(queries) != 1 || !strings.Contains(queries[0], `repository(owner: \"them\", name: \"fork\")`) {
		t.Errorf("fork lookup queries = %v", queries)
	}
}
