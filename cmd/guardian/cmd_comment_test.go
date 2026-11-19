package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const (
	demoBase = "../../testdata/demo/base.yaml"
	demoPR   = "../../testdata/demo/pr.yaml"
	safeBase = "../../testdata/safe_additions/base.yaml"
	safePR   = "../../testdata/safe_additions/pr.yaml"
)

func noEnv(string) string { return "" }

// fakeAPI is a minimal in-memory GitHub: one PR (#7) with a list of comments.
type fakeAPI struct {
	mu       sync.Mutex
	comments []map[string]any
	calls    []string
	status   int // if non-zero, answer every request with this status
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method)
	if f.status != 0 {
		http.Error(w, `{"message":"denied"}`, f.status)
		return
	}
	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(f.comments)
	case "POST":
		var in map[string]any
		json.NewDecoder(r.Body).Decode(&in)
		in["id"] = len(f.comments) + 1
		f.comments = append(f.comments, in)
		w.WriteHeader(201)
		w.Write([]byte("{}"))
	case "PATCH":
		var in map[string]any
		json.NewDecoder(r.Body).Decode(&in)
		f.comments[0]["body"] = in["body"]
		w.Write([]byte("{}"))
	}
}

func commentArgs(srv *httptest.Server, base, pr string) []string {
	return []string{"comment", "--repo", "octo/demo", "--pr", "7", "--token", "t", "--api-url", srv.URL, base, pr}
}

func TestCommentPostsThenUpdates(t *testing.T) {
	api := &fakeAPI{}
	srv := httptest.NewServer(api)
	defer srv.Close()

	var out, errb bytes.Buffer
	if code := runWith(commentArgs(srv, demoBase, demoPR), &out, &errb, noEnv); code != 1 {
		t.Fatalf("exit = %d, want 1 (breaking)\nstderr: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "comment created on octo/demo#7") {
		t.Errorf("stdout: %s", out.String())
	}
	if len(api.comments) != 1 || !strings.Contains(api.comments[0]["body"].(string), "<!-- guardian-report -->") {
		t.Fatalf("expected one Guardian comment, got %+v", api.comments)
	}

	// Run again (as on the next push): must edit, not add a second comment.
	out.Reset()
	runWith(commentArgs(srv, safeBase, safePR), &out, &errb, noEnv)
	if len(api.comments) != 1 {
		t.Errorf("second run should update in place, now %d comments", len(api.comments))
	}
	if !strings.Contains(api.comments[0]["body"].(string), "no breaking changes") {
		t.Error("comment body should reflect the latest result")
	}
	if !strings.Contains(out.String(), "comment updated") {
		t.Errorf("stdout: %s", out.String())
	}
}

func TestCommentExitCodeMatchesDiffResult(t *testing.T) {
	srv := httptest.NewServer(&fakeAPI{})
	defer srv.Close()
	var out, errb bytes.Buffer
	if code := runWith(commentArgs(srv, safeBase, safePR), &out, &errb, noEnv); code != 0 {
		t.Errorf("no breaking changes should exit 0, got %d", code)
	}
}

func TestCommentDryRunPostsNothing(t *testing.T) {
	api := &fakeAPI{}
	srv := httptest.NewServer(api)
	defer srv.Close()

	var out, errb bytes.Buffer
	code := runWith([]string{"comment", "--dry-run", demoBase, demoPR}, &out, &errb, noEnv)
	if code != 1 {
		t.Errorf("dry run still reports the result: exit %d, want 1", code)
	}
	if len(api.calls) != 0 {
		t.Errorf("dry run must not call the API: %v", api.calls)
	}
	if !strings.Contains(out.String(), "comment preview") || !strings.Contains(out.String(), "## 🛡️ Guardian API check") {
		t.Errorf("dry run should print the comment:\n%s", out.String())
	}
}

func TestCommentPermissionErrorWarnsButKeepsResult(t *testing.T) {
	srv := httptest.NewServer(&fakeAPI{status: 403})
	defer srv.Close()

	var out, errb bytes.Buffer
	code := runWith(commentArgs(srv, demoBase, demoPR), &out, &errb, noEnv)
	if code != 1 {
		t.Errorf("a failed comment must not change the result: exit %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "pull-requests: write") {
		t.Errorf("expected a permissions hint, got: %s", errb.String())
	}
}

func TestCommentUsesEnvironmentLikeGitHubActions(t *testing.T) {
	api := &fakeAPI{}
	srv := httptest.NewServer(api)
	defer srv.Close()

	dir := t.TempDir()
	event := filepath.Join(dir, "event.json")
	summary := filepath.Join(dir, "summary.md")
	os.WriteFile(event, []byte(`{"pull_request": {"number": 7}}`), 0o644)
	env := map[string]string{
		"GITHUB_REPOSITORY": "octo/demo", "GITHUB_EVENT_PATH": event,
		"GITHUB_TOKEN": "t", "GITHUB_API_URL": srv.URL, "GITHUB_STEP_SUMMARY": summary,
	}

	var out, errb bytes.Buffer
	code := runWith([]string{"comment", demoBase, demoPR}, &out, &errb, func(k string) string { return env[k] })
	if code != 1 {
		t.Fatalf("exit = %d\nstderr: %s", code, errb.String())
	}
	if len(api.comments) != 1 {
		t.Errorf("expected the comment to be posted using env-derived settings")
	}
	if data, _ := os.ReadFile(summary); !strings.Contains(string(data), "Guardian API check") {
		t.Error("report should also be written to the job summary")
	}
}

func TestCommentConfigurationErrors(t *testing.T) {
	var out, errb bytes.Buffer
	// No token anywhere.
	if code := runWith([]string{"comment", "--repo", "o/r", "--pr", "1", demoBase, demoPR}, &out, &errb, noEnv); code != 2 {
		t.Errorf("missing token: exit %d, want 2", code)
	}
	// No PR number anywhere.
	errb.Reset()
	if code := runWith([]string{"comment", "--token", "t", demoBase, demoPR}, &out, &errb, noEnv); code != 2 {
		t.Errorf("missing PR: exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "--repo and --pr") {
		t.Errorf("error should explain the fix: %s", errb.String())
	}
	// Wrong arguments.
	if code := runWith([]string{"comment", "only-one.yaml"}, &out, &errb, noEnv); code != 2 {
		t.Errorf("bad usage: exit %d, want 2", code)
	}
}

// Two named checks on the same PR each get their own comment.
func TestCommentNamesGiveSeparateComments(t *testing.T) {
	api := &fakeAPI{}
	srv := httptest.NewServer(api)
	defer srv.Close()

	post := func(name string) {
		var out, errb bytes.Buffer
		args := append([]string{"comment", "--name", name}, commentArgs(srv, demoBase, demoPR)[1:]...)
		runWith(args, &out, &errb, noEnv)
	}
	post("users-api")
	post("orders-api")

	if len(api.comments) != 2 {
		t.Fatalf("two differently named checks should create two comments, got %d", len(api.comments))
	}
	b0, b1 := api.comments[0]["body"].(string), api.comments[1]["body"].(string)
	if !strings.Contains(b0, "guardian-report:users-api") || !strings.Contains(b1, "guardian-report:orders-api") {
		t.Errorf("comments should carry their own markers:\n%s\n---\n%s", b0, b1)
	}
}
