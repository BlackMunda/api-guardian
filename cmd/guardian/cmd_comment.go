package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"guardian/internal/check"
	"guardian/internal/github"
	"guardian/internal/report"
	"guardian/internal/rules"
)

// runComment: diff two specs, then post (or update) the report as a comment on
// the pull request. The exit code reflects the diff result, exactly like
// `guardian diff`, so the CI job fails when something is breaking.
//
// Inside GitHub Actions the repository, PR number, token and API URL are all
// picked up from the environment; flags override them (handy for local use).
func runComment(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	fs := flag.NewFlagSet("comment", flag.ContinueOnError)
	fs.SetOutput(stderr)
	repo := fs.String("repo", "", "repository as owner/name (default: $GITHUB_REPOSITORY)")
	pr := fs.Int("pr", 0, "pull request number (default: read from $GITHUB_EVENT_PATH)")
	token := fs.String("token", "", "API token (default: $GITHUB_TOKEN)")
	apiURL := fs.String("api-url", "", "GitHub API URL (default: $GITHUB_API_URL or https://api.github.com)")
	name := fs.String("name", "", "name for this check; gives it its own PR comment when several checks run on one PR")
	dryRun := fs.Bool("dry-run", false, "print the comment instead of posting it")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 2 {
		usage(stderr)
		return 2
	}

	changes, err := check.Run(fs.Arg(0), fs.Arg(1))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	// Always show the plain report in the job log.
	report.Text(stdout, changes)
	md := report.MarkdownFor(changes, *name)

	exit := 0
	if breaking, _ := rules.Count(changes); breaking > 0 {
		exit = 1
	}

	// A second copy on the workflow run's summary page (no-op outside Actions).
	if err := github.WriteStepSummary(getenv, md); err != nil {
		fmt.Fprintln(stderr, "warning:", err)
	}

	if *dryRun {
		fmt.Fprintf(stdout, "\n--- comment preview (dry run, nothing posted) ---\n%s", md)
		return exit
	}

	target, err := resolveTarget(*repo, *pr, getenv)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	tok := firstNonEmpty(*token, getenv("GITHUB_TOKEN"), getenv("GH_TOKEN"))
	if tok == "" {
		fmt.Fprintln(stderr, "error: no token: pass --token or set GITHUB_TOKEN")
		return 2
	}

	client := github.NewClient(tok)
	client.BaseURL = firstNonEmpty(*apiURL, getenv("GITHUB_API_URL"), client.BaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	action, err := client.UpsertComment(ctx, target, report.MarkerFor(*name), md)
	if err != nil {
		// A failed comment must not hide the real result, so we only warn and
		// keep the exit code from the diff.
		fmt.Fprintln(stderr, "warning: could not post PR comment:", err)
		if github.IsPermissionError(err) {
			fmt.Fprintln(stderr, "hint: the token needs `pull-requests: write`. PRs from forks get a read-only token, so no comment is possible there; see the job summary instead.")
		}
		return exit
	}
	fmt.Fprintf(stdout, "\nGuardian comment %s on %s\n", action, target)
	return exit
}

// resolveTarget combines the environment with explicit flags (flags win).
func resolveTarget(repoFlag string, prFlag int, getenv func(string) string) (github.Target, error) {
	t, envErr := github.TargetFromEnv(getenv, os.ReadFile)
	if repoFlag != "" {
		owner, name, err := github.ParseRepo(repoFlag)
		if err != nil {
			return t, err
		}
		t.Owner, t.Repo = owner, name
	}
	if prFlag != 0 {
		t.PR = prFlag
	}
	if t.Owner == "" || t.Repo == "" || t.PR == 0 {
		if envErr == nil {
			envErr = errors.New("repository or pull request number is missing")
		}
		return t, fmt.Errorf("cannot tell which pull request to comment on (%v); pass --repo and --pr", envErr)
	}
	return t, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
