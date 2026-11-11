package github

import (
	"fmt"
	"os"
)

// WriteStepSummary appends markdown to the job summary page that GitHub
// Actions shows on the workflow run. It is a no-op outside Actions, where
// GITHUB_STEP_SUMMARY is not set. This gives a second place to read the
// report even when the PR comment cannot be posted (e.g. PRs from forks).
func WriteStepSummary(getenv func(string) string, markdown string) error {
	path := getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("opening step summary: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(markdown + "\n"); err != nil {
		return fmt.Errorf("writing step summary: %w", err)
	}
	return nil
}
