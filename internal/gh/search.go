package gh

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// searchResponse is the subset of the REST repository-search payload we need.
// Field names differ from the GraphQL schema (full_name, stargazers_count, …).
type searchResponse struct {
	Items []struct {
		FullName        string    `json:"full_name"`
		Description     string    `json:"description"`
		Private         bool      `json:"private"`
		StargazersCount int       `json:"stargazers_count"`
		UpdatedAt       time.Time `json:"updated_at"`
		Language        string    `json:"language"`
	} `json:"items"`
}

// searchLimit caps results to a single page — one request per query keeps the
// search API's strict rate limit (30 req/min) comfortable when debounced.
const searchLimit = 30

// buildSearchQuery turns user input into a GitHub repository-search query.
//
//	"react"            → react in:name           (match repo names anywhere)
//	"facebook/"        → user:facebook           (all of an owner's repos)
//	"facebook/re"      → re in:name user:facebook (an owner's repos by name)
//
// Returns "" when there is nothing to search for.
func buildSearchQuery(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	owner, name, hasSlash := strings.Cut(input, "/")
	if !hasSlash {
		return input + " in:name"
	}
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" {
		// "/foo" — no owner to scope by; fall back to a name search.
		return name + " in:name"
	}
	if name == "" {
		return "user:" + owner
	}
	return name + " in:name user:" + owner
}

// SearchRepos queries GitHub for repositories matching input (a name, an
// owner/, or owner/name), ranked by stars. It returns nil for empty input.
func (c *Client) SearchRepos(input string) ([]Repo, error) {
	q := buildSearchQuery(input)
	if q == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("q", q)
	params.Set("sort", "stars")
	params.Set("order", "desc")
	params.Set("per_page", strconv.Itoa(searchLimit))
	path := "search/repositories?" + params.Encode()

	var resp searchResponse
	if err := c.rest.Get(path, &resp); err != nil {
		return nil, err
	}

	repos := make([]Repo, 0, len(resp.Items))
	for _, it := range resp.Items {
		repos = append(repos, Repo{
			NameWithOwner:  it.FullName,
			Description:    it.Description,
			IsPrivate:      it.Private,
			StargazerCount: it.StargazersCount,
			UpdatedAt:      it.UpdatedAt,
			Language:       it.Language,
		})
	}
	return repos, nil
}
