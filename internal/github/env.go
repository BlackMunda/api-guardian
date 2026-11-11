package github

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseRepo splits "owner/name" into its two parts.
func ParseRepo(s string) (owner, name string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository %q: want owner/name", s)
	}
	return parts[0], parts[1], nil
}

// TargetFromEnv works out which PR we are running for, using the variables
// GitHub Actions sets automatically:
//
//	GITHUB_REPOSITORY   "owner/name"
//	GITHUB_EVENT_PATH   path to a JSON file describing the triggering event
//
// For a pull_request event, that JSON contains {"pull_request": {"number": N}}.
// It returns whatever it managed to find along with an error, so callers can
// let command-line flags fill the gaps.
func TargetFromEnv(getenv func(string) string, readFile func(string) ([]byte, error)) (Target, error) {
	var t Target

	if repo := getenv("GITHUB_REPOSITORY"); repo != "" {
		owner, name, err := ParseRepo(repo)
		if err != nil {
			return t, err
		}
		t.Owner, t.Repo = owner, name
	}

	path := getenv("GITHUB_EVENT_PATH")
	if path == "" {
		return t, fmt.Errorf("GITHUB_EVENT_PATH is not set (is this running inside GitHub Actions?)")
	}
	data, err := readFile(path)
	if err != nil {
		return t, fmt.Errorf("reading event file: %w", err)
	}
	var event struct {
		Number      int `json:"number"`
		PullRequest *struct {
			Number int `json:"number"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		return t, fmt.Errorf("parsing event file: %w", err)
	}
	switch {
	case event.PullRequest != nil && event.PullRequest.Number != 0:
		t.PR = event.PullRequest.Number
	case event.Number != 0:
		t.PR = event.Number
	default:
		return t, fmt.Errorf("event file has no pull request number (is the workflow triggered by pull_request?)")
	}
	return t, nil
}
