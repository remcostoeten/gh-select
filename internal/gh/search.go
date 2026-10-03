package gh

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sahilm/fuzzy"
)

// restRepo is the REST repository payload shared by search, owner listings and
// single-repo lookups. Field names differ from the GraphQL schema.
type restRepo struct {
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	Private         bool      `json:"private"`
	StargazersCount int       `json:"stargazers_count"`
	UpdatedAt       time.Time `json:"updated_at"`
	Language        string    `json:"language"`
	Fork            bool      `json:"fork"`
	Owner           struct {
		Login string `json:"login"`
	} `json:"owner"`
}

func (r restRepo) toRepo() Repo {
	return Repo{
		NameWithOwner:  r.FullName,
		Description:    r.Description,
		IsPrivate:      r.Private,
		StargazerCount: r.StargazersCount,
		UpdatedAt:      r.UpdatedAt,
		Language:       r.Language,
		OwnerLogin:     r.Owner.Login,
		IsFork:         r.Fork,
	}
}

// searchLimit caps results to a single page — one request per query keeps the
// search API's strict rate limit (30 req/min) comfortable when debounced.
const searchLimit = 30

// ownerListLimit is how many of an owner's most recently pushed repos are
// fetched to match a partial name against.
const ownerListLimit = 100

// maxLoginGuesses bounds how many look-alike owners are tried for a name.
const maxLoginGuesses = 2

// maxHyphenSplits bounds the owner/name guesses tried for "owner-repo" input.
const maxHyphenSplits = 3

// Matches a github host with any TLD, the separator and an optional port before
// the path, as in https://www.github.com/, git@github.nl: or
// ssh://git@github.com:22/, but not subdomains such as gist. or api.
var githubHost = regexp.MustCompile(`(?i)(?:^|[\s/@])(?:www\.)?github(?:\.[a-z]{2,})+\s*[:/]+(?:\d+/)?`)

// Matches search qualifiers that pass straight through to GitHub.
var qualifier = regexp.MustCompile(`(?i)^(?:language|is|topic|stars|archived|fork|user|org):\S+$`)

// Matches the qualifiers the UI re-applies to an owner listing; any other one
// means only GitHub search can honor the query.
var listableQualifier = regexp.MustCompile(`(?i)^(?:language|is):`)

// searchPlan describes every lookup one search input fans out into.
type searchPlan struct {
	exact         []string
	owner         string
	filter        string
	query         string
	ownerFallback bool
	byStars       bool
}

// planSearch interprets free-form input: a repo name, an owner, owner/name,
// "owner name", "owner-name", a github URL with any scheme or TLD, an SSH
// remote or a pasted `git clone` command.
func planSearch(input string) searchPlan {
	plan, quals := planWords(input)
	for _, q := range quals {
		if !listableQualifier.MatchString(q) {
			plan.exact, plan.owner, plan.filter, plan.ownerFallback = nil, "", "", false
			break
		}
	}
	return plan
}

func planWords(input string) (searchPlan, []string) {
	var quals, words []string
	for _, f := range strings.Fields(input) {
		if qualifier.MatchString(f) {
			quals = append(quals, f)
		} else {
			words = append(words, f)
		}
	}
	if len(words) >= 2 && strings.EqualFold(words[0], "git") && strings.EqualFold(words[1], "clone") {
		words = words[2:]
	}
	text := strings.Join(words, " ")
	withQuals := func(q string) string {
		return strings.TrimSpace(strings.Join(append([]string{q}, quals...), " "))
	}

	if loc := githubHost.FindStringIndex(text); loc != nil {
		owner, name := splitRef(firstField(text[loc[1]:]))
		if owner == "" {
			return searchPlan{query: withQuals("")}, quals
		}
		return refPlan(owner, name, false, withQuals), quals
	}

	if len(words) > 1 && isGithubWord(words[0]) {
		words = words[1:]
	}
	switch {
	case len(words) == 0:
		return searchPlan{query: withQuals("")}, quals
	case len(words) == 1 && strings.Contains(words[0], "://"):
		return searchPlan{query: withQuals("")}, quals
	case len(words) == 1 && strings.Contains(words[0], "/"):
		owner, name := splitRef(words[0])
		if owner == "" || strings.HasPrefix(words[0], "/") {
			return searchPlan{query: withQuals(strings.Trim(words[0], "/") + " in:name")}, quals
		}
		return refPlan(owner, name, true, withQuals), quals
	case len(words) == 1:
		word := strings.TrimSuffix(words[0], ".git")
		return searchPlan{
			exact:   hyphenGuesses(word),
			owner:   word,
			query:   withQuals(word + " in:name"),
			byStars: true,
		}, quals
	}
	rest := strings.Join(words[1:], "-")
	return searchPlan{
		exact:         []string{words[0] + "/" + rest},
		owner:         words[0],
		filter:        rest,
		query:         withQuals(strings.Join(words, " ")),
		ownerFallback: true,
	}, quals
}

func refPlan(owner, name string, fallback bool, withQuals func(string) string) searchPlan {
	if name == "" {
		return searchPlan{owner: owner, query: withQuals("user:" + owner), ownerFallback: fallback}
	}
	return searchPlan{
		exact:         []string{owner + "/" + name},
		owner:         owner,
		filter:        name,
		query:         withQuals(name + " in:name user:" + owner),
		ownerFallback: fallback,
	}
}

// splitRef reads owner and repo from a path such as heygen-com/repo.git/tree/main.
func splitRef(path string) (owner, name string) {
	path, _, _ = strings.Cut(path, "?")
	path, _, _ = strings.Cut(path, "#")
	path = strings.TrimRight(strings.TrimLeft(path, "([<\"'`"), ")]>.,;\"'`")
	segs := strings.Split(strings.Trim(path, "/"), "/")
	owner = strings.TrimSpace(segs[0])
	if len(segs) > 1 {
		name = strings.TrimSuffix(strings.TrimSpace(segs[1]), ".git")
	}
	return owner, name
}

func firstField(s string) string {
	if fields := strings.Fields(s); len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func isGithubWord(w string) bool {
	host, _, _ := strings.Cut(strings.ToLower(w), ".")
	return host == "github"
}

func hyphenGuesses(word string) []string {
	var out []string
	for i, c := range word {
		if c != '-' || i == 0 || i == len(word)-1 {
			continue
		}
		out = append(out, word[:i]+"/"+word[i+1:])
		if len(out) == maxHyphenSplits {
			break
		}
	}
	return out
}

// SearchRepos finds repositories anywhere on GitHub from free-form input (see
// planSearch). Direct hits come first, then the owner's repos matching the
// name, then GitHub search results. It returns nil for empty input.
func (c *Client) SearchRepos(input string) ([]Repo, error) {
	plan := planSearch(input)
	if plan.query == "" && plan.owner == "" {
		return nil, nil
	}

	var (
		wg        sync.WaitGroup
		exact     = make([]*Repo, len(plan.exact))
		owned     []Repo
		found     []Repo
		searchErr error
	)
	for i, ref := range plan.exact {
		wg.Go(func() {
			var r restRepo
			if c.rest.Get("repos/"+ref, &r) == nil {
				repo := r.toRepo()
				exact[i] = &repo
			}
		})
	}
	if plan.owner != "" {
		wg.Go(func() { owned = c.ownerRepos(plan) })
	}
	if plan.query != "" {
		wg.Go(func() { found, searchErr = c.searchRepos(plan.query) })
	}
	wg.Wait()

	var out []Repo
	for _, r := range exact {
		if r != nil {
			out = append(out, *r)
		}
	}
	rest := append(owned, found...)
	if plan.byStars {
		sort.SliceStable(rest, func(i, j int) bool { return rest[i].StargazerCount > rest[j].StargazerCount })
	}
	out = dedupeRepos(append(out, rest...))
	if len(out) == 0 && searchErr != nil {
		return nil, searchErr
	}
	return out, nil
}

func (c *Client) ownerRepos(plan searchPlan) []Repo {
	repos, err := c.listOwnerRepos(plan.owner)
	matched := filterByName(repos, plan.filter)
	if (err == nil && len(matched) > 0) || !plan.ownerFallback {
		return matched
	}
	for _, login := range c.similarLogins(plan.owner) {
		if repos, err := c.listOwnerRepos(login); err == nil {
			if matched := filterByName(repos, plan.filter); len(matched) > 0 {
				return matched
			}
		}
	}
	return nil
}

func (c *Client) listOwnerRepos(owner string) ([]Repo, error) {
	var resp []restRepo
	path := "users/" + url.PathEscape(owner) + "/repos?sort=pushed&per_page=" + strconv.Itoa(ownerListLimit)
	if err := c.rest.Get(path, &resp); err != nil {
		return nil, err
	}
	repos := make([]Repo, len(resp))
	for i, r := range resp {
		repos[i] = r.toRepo()
	}
	return repos, nil
}

// similarLogins finds up to maxLoginGuesses accounts whose login resembles
// owner, so "heygen" also reaches heygen-com.
func (c *Client) similarLogins(owner string) []string {
	var resp struct {
		Items []struct {
			Login string `json:"login"`
		} `json:"items"`
	}
	params := url.Values{"q": {owner + " in:login"}, "per_page": {strconv.Itoa(maxLoginGuesses + 1)}}
	if c.rest.Get("search/users?"+params.Encode(), &resp) != nil {
		return nil
	}
	var out []string
	for _, it := range resp.Items {
		if !strings.EqualFold(it.Login, owner) && len(out) < maxLoginGuesses {
			out = append(out, it.Login)
		}
	}
	return out
}

func (c *Client) searchRepos(q string) ([]Repo, error) {
	params := url.Values{}
	params.Set("q", q)
	params.Set("sort", "stars")
	params.Set("order", "desc")
	params.Set("per_page", strconv.Itoa(searchLimit))

	var resp struct {
		Items []restRepo `json:"items"`
	}
	if err := c.rest.Get("search/repositories?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	repos := make([]Repo, len(resp.Items))
	for i, it := range resp.Items {
		repos[i] = it.toRepo()
	}
	return repos, nil
}

// filterByName keeps repos whose name fuzzy-matches filter, best match first.
func filterByName(repos []Repo, filter string) []Repo {
	filter = strings.NewReplacer(" ", "", "-", "", "_", "").Replace(filter)
	if filter == "" {
		return repos
	}
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.Name()
	}
	matches := fuzzy.Find(filter, names)
	out := make([]Repo, len(matches))
	for i, m := range matches {
		out[i] = repos[m.Index]
	}
	return out
}

// IsRepoURL reports whether input holds a github URL, SSH remote or clone
// command, which only a GitHub-wide search can resolve.
func IsRepoURL(input string) bool {
	return githubHost.MatchString(input)
}
