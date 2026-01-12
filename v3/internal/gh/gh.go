package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type Repository struct {
	NameWithOwner string `json:"nameWithOwner"`
	Description   string `json:"description"`
	IsPrivate     bool   `json:"isPrivate"`
}

func FetchRepos(limit int, allowCache bool) ([]Repository, error) {
	// allowCache param is now effectively unused effectively as logic moved to caller
	// but keeping signature for now or revert? 
	// Let's just ignore it for now as caller handles it.
	
	cmd := exec.Command("gh", "repo", "list", "--json", "nameWithOwner,description,isPrivate", "--limit", fmt.Sprint(limit))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos: %w", err)
	}

	var repos []Repository
	if err := json.Unmarshal(output, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse repos: %w", err)
	}

	return repos, nil
}
