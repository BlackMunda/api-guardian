// Package github is a tiny GitHub REST client: just enough to post a PR
// comment, or update the one we posted earlier. It uses only net/http, so it
// adds no dependencies and is easy to test against a fake server.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Target identifies the pull request to comment on.
// (GitHub treats a PR as an "issue" for the purpose of comments.)
type Target struct {
	Owner string
	Repo  string
	PR    int
}

func (t Target) String() string { return fmt.Sprintf("%s/%s#%d", t.Owner, t.Repo, t.PR) }

// Client talks to the GitHub REST API.
type Client struct {
	BaseURL string       // e.g. https://api.github.com (no trailing slash)
	Token   string       // GITHUB_TOKEN or a personal access token
	HTTP    *http.Client // optional; a 30s-timeout client is used if nil
}

// NewClient returns a client for github.com. Override BaseURL for GitHub
// Enterprise or for tests.
func NewClient(token string) *Client {
	return &Client{BaseURL: "https://api.github.com", Token: token}
}

// APIError is returned when GitHub answers with a non-2xx status.
type APIError struct {
	Method string
	URL    string
	Status int
	Body   string // first part of the response body
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github: %s %s: HTTP %d: %s", e.Method, e.URL, e.Status, strings.TrimSpace(e.Body))
}

// IsPermissionError reports whether err means "the token may not do this".
// The common cause is a pull request from a fork, whose GITHUB_TOKEN is
// read-only.
func IsPermissionError(err error) bool {
	ae, ok := err.(*APIError)
	return ok && (ae.Status == http.StatusForbidden || ae.Status == http.StatusUnauthorized || ae.Status == http.StatusNotFound)
}

type comment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

const (
	perPage  = 100
	maxPages = 50 // safety cap: 5,000 comments
)

// UpsertComment makes sure the PR has exactly one comment containing marker.
// If one exists it is edited in place; otherwise a new one is created.
// It returns "created" or "updated".
func (c *Client) UpsertComment(ctx context.Context, t Target, marker, body string) (string, error) {
	existing, err := c.findComment(ctx, t, marker)
	if err != nil {
		return "", err
	}
	payload := map[string]string{"body": body}

	if existing != nil {
		url := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d", c.BaseURL, t.Owner, t.Repo, existing.ID)
		if err := c.do(ctx, http.MethodPatch, url, payload, nil); err != nil {
			return "", err
		}
		return "updated", nil
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.BaseURL, t.Owner, t.Repo, t.PR)
	if err := c.do(ctx, http.MethodPost, url, payload, nil); err != nil {
		return "", err
	}
	return "created", nil
}

// findComment pages through the PR's comments looking for the marker.
func (c *Client) findComment(ctx context.Context, t Target, marker string) (*comment, error) {
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=%d&page=%d",
			c.BaseURL, t.Owner, t.Repo, t.PR, perPage, page)
		var list []comment
		if err := c.do(ctx, http.MethodGet, url, nil, &list); err != nil {
			return nil, err
		}
		for i := range list {
			if strings.Contains(list[i].Body, marker) {
				return &list[i], nil
			}
		}
		if len(list) < perPage {
			return nil, nil
		}
	}
	return nil, nil
}

func (c *Client) do(ctx context.Context, method, url string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "guardian")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &APIError{Method: method, URL: url, Status: resp.StatusCode, Body: string(b)}
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
