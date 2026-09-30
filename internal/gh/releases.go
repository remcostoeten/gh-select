package gh

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// releasesPerPage bounds the listing to one request (GitHub's maximum page);
// older releases are rarely what someone browsing from a picker is after.
const releasesPerPage = 100

// Asset is a downloadable file attached to a release. URL is the API endpoint,
// which works for private repositories where the browser URL would not.
type Asset struct {
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	DownloadCount int    `json:"download_count"`
	URL           string `json:"url"`
	DownloadURL   string `json:"browser_download_url"`
	Source        bool   `json:"-"` // a generated source archive, not an uploaded file
}

// Release is a published (or, for maintainers, draft) GitHub release.
type Release struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	Assets      []Asset   `json:"assets"`
	TarballURL  string    `json:"tarball_url"`
	ZipballURL  string    `json:"zipball_url"`
}

// Title is the release name, falling back to its tag like GitHub's own UI.
func (r Release) Title() string {
	if strings.TrimSpace(r.Name) != "" {
		return r.Name
	}
	return r.TagName
}

// Date is when the release went out; drafts have only a creation date.
func (r Release) Date() time.Time {
	if r.PublishedAt.IsZero() {
		return r.CreatedAt
	}
	return r.PublishedAt
}

// Downloads lists the uploaded assets followed by the source archives GitHub
// generates for every release, named the way github.com names them.
func (r Release) Downloads(repoName string) []Asset {
	out := append([]Asset{}, r.Assets...)
	base := repoName + "-" + strings.TrimPrefix(r.TagName, "v")
	if r.TarballURL != "" {
		out = append(out, Asset{Name: base + ".tar.gz", URL: r.TarballURL, Source: true})
	}
	if r.ZipballURL != "" {
		out = append(out, Asset{Name: base + ".zip", URL: r.ZipballURL, Source: true})
	}
	return out
}

// FetchReleases returns the repository's most recent releases, newest first.
func (c *Client) FetchReleases(nameWithOwner string) ([]Release, error) {
	var releases []Release
	path := fmt.Sprintf("repos/%s/releases?per_page=%d", nameWithOwner, releasesPerPage)
	if err := c.rest.Get(path, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

// OpenAsset starts downloading an asset and returns its body with the length
// GitHub reported (-1 when unknown, as for generated source archives). For an
// uploaded asset the octet-stream Accept header makes the API redirect to the
// file itself rather than describe it; the archive endpoints reject that header
// and redirect on their own. The token is dropped on cross-host redirects.
func (c *Client) OpenAsset(a Asset) (io.ReadCloser, int64, error) {
	req, err := http.NewRequest(http.MethodGet, a.URL, nil)
	if err != nil {
		return nil, 0, err
	}
	if !a.Source {
		req.Header.Set("Accept", "application/octet-stream")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, 0, fmt.Errorf("%s: %w", a.Name, api.HandleHTTPError(resp))
	}
	return resp.Body, resp.ContentLength, nil
}
