package gh

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Repo is a single GitHub repository with the fields gh-select displays and
// acts on. JSON tags double as the cache serialization format.
type Repo struct {
	NameWithOwner  string    `json:"nameWithOwner"`
	Description    string    `json:"description"`
	IsPrivate      bool      `json:"isPrivate"`
	StargazerCount int       `json:"stargazerCount"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Language       string    `json:"language"`
	OwnerLogin     string    `json:"ownerLogin"`
	IsOwner        bool      `json:"isOwner"`
	IsFork         bool      `json:"isFork"`
	Parent         string    `json:"parent,omitempty"` // "owner/repo" a fork was made from, when known
	PushedAt       time.Time `json:"pushedAt"`
	IsArchived     bool      `json:"isArchived"`
	License        string    `json:"license,omitempty"` // SPDX id, "NOASSERTION" for an unrecognized license
	DiskKB         int       `json:"diskKB"`
}

// LastActivity is when code was last pushed, falling back to the last update
// of any kind for payloads that lack a push time.
func (r Repo) LastActivity() time.Time {
	if r.PushedAt.IsZero() {
		return r.UpdatedAt
	}
	return r.PushedAt
}

// URL returns the canonical https URL for the repository.
func (r Repo) URL() string {
	return "https://github.com/" + r.NameWithOwner
}

// Name returns the repository name without its owner prefix.
func (r Repo) Name() string {
	if i := strings.LastIndex(r.NameWithOwner, "/"); i >= 0 {
		return r.NameWithOwner[i+1:]
	}
	return r.NameWithOwner
}

// reposPerPage is the REST maximum page size, and reposWave how many pages
// are requested at once. GraphQL can only page by cursor, one round-trip after
// another; numbered REST pages can be fetched side by side.
const (
	reposPerPage = 100
	reposWave    = 4
	reposPath    = "user/repos?affiliation=owner,collaborator,organization_member&sort=updated"
)

// forksQuery reads the viewer's login and the parent and language of every
// fork the viewer can reach, which the REST listing leaves out. Both affiliation
// arguments are needed: ownerAffiliations defaults to [OWNER, COLLABORATOR]
// and would drop organization forks regardless of what affiliations says.
const forksQuery = `
query($cursor: String) {
  viewer {
    login
    repositories(first: 100, after: $cursor, isFork: true, affiliations: [OWNER, ORGANIZATION_MEMBER, COLLABORATOR], ownerAffiliations: [OWNER, ORGANIZATION_MEMBER, COLLABORATOR]) {
      nodes {
        nameWithOwner
        parent { nameWithOwner }
        primaryLanguage { name }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type forkFacts struct{ parent, language string }

type forksResponse struct {
	Viewer struct {
		Login        string `json:"login"`
		Repositories struct {
			Nodes []struct {
				NameWithOwner string `json:"nameWithOwner"`
				Parent        *struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"parent"`
				PrimaryLanguage struct {
					Name string `json:"name"`
				} `json:"primaryLanguage"`
			} `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"repositories"`
	} `json:"viewer"`
}

// FetchRepos returns every repository the authenticated user can reach: their
// own, their organizations', and those they collaborate on, most recently
// updated first. The REST pages and the fork parents load concurrently, so a
// few hundred repos cost about one round-trip instead of one per 100 repos.
func (c *Client) FetchRepos() ([]Repo, error) {
	var (
		wg      sync.WaitGroup
		viewer  string
		forks   map[string]forkFacts
		forkErr error
	)
	wg.Go(func() { viewer, forks, forkErr = c.fetchForks() })
	repos, err := c.fetchRepoPages()
	wg.Wait()
	if err != nil {
		return nil, err
	}
	if forkErr != nil {
		return nil, forkErr
	}
	for i := range repos {
		repos[i].IsOwner = repos[i].OwnerLogin == viewer
		if f, ok := forks[repos[i].NameWithOwner]; ok {
			repos[i].Parent = f.parent
			if repos[i].Language == "" {
				repos[i].Language = f.language
			}
		}
	}
	return dedupeRepos(repos), nil
}

// fetchRepoPages requests reposWave pages at a time until one comes back
// short, keeping the pages in order.
func (c *Client) fetchRepoPages() ([]Repo, error) {
	var repos []Repo
	for first := 1; ; first += reposWave {
		pages := make([][]restRepo, reposWave)
		errs := make([]error, reposWave)
		var wg sync.WaitGroup
		for i := range pages {
			wg.Go(func() {
				path := fmt.Sprintf("%s&per_page=%d&page=%d", reposPath, reposPerPage, first+i)
				errs[i] = c.rest.Get(path, &pages[i])
			})
		}
		wg.Wait()
		for i, page := range pages {
			if errs[i] != nil {
				return nil, errs[i]
			}
			for _, r := range page {
				repos = append(repos, r.toRepo())
			}
			if len(page) < reposPerPage {
				return repos, nil
			}
		}
	}
}

func (c *Client) fetchForks() (string, map[string]forkFacts, error) {
	var viewer string
	forks := map[string]forkFacts{}
	var cursor *string
	for {
		var resp forksResponse
		if err := c.gql.Do(forksQuery, map[string]interface{}{"cursor": cursor}, &resp); err != nil {
			return "", nil, err
		}
		viewer = resp.Viewer.Login
		page := resp.Viewer.Repositories
		for _, n := range page.Nodes {
			forks[n.NameWithOwner] = forkFacts{parent: parentName(n.Parent), language: n.PrimaryLanguage.Name}
		}
		if !page.PageInfo.HasNextPage {
			return viewer, forks, nil
		}
		end := page.PageInfo.EndCursor
		cursor = &end
	}
}

func parentName(p *struct {
	NameWithOwner string `json:"nameWithOwner"`
}) string {
	if p == nil {
		return ""
	}
	return p.NameWithOwner
}

type licenseInfo struct {
	SpdxID string `json:"spdxId"`
}

func (l *licenseInfo) id() string {
	if l == nil {
		return ""
	}
	return l.SpdxID
}

// dedupeRepos drops repeated NameWithOwner entries, keeping the first occurrence
// so the server's UPDATED_AT DESC ordering survives. Broadening the affiliation
// filters lets the same repository arrive under more than one affiliation.
func dedupeRepos(repos []Repo) []Repo {
	seen := make(map[string]struct{}, len(repos))
	out := make([]Repo, 0, len(repos))
	for _, r := range repos {
		if _, dup := seen[r.NameWithOwner]; dup {
			continue
		}
		seen[r.NameWithOwner] = struct{}{}
		out = append(out, r)
	}
	return out
}
