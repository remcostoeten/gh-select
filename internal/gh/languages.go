package gh

import "sort"

// FetchLanguages returns every language GitHub detects in nameWithOwner,
// largest share of code first.
func (c *Client) FetchLanguages(nameWithOwner string) ([]string, error) {
	var bytes map[string]int
	if err := c.rest.Get("repos/"+nameWithOwner+"/languages", &bytes); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(bytes))
	for name := range bytes {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if bytes[names[i]] != bytes[names[j]] {
			return bytes[names[i]] > bytes[names[j]]
		}
		return names[i] < names[j]
	})
	return names, nil
}
