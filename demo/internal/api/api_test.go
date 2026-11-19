package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	New(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func call(t *testing.T, method, url string, body any) (int, []byte) {
	t.Helper()
	var r *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, url, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.Bytes()
}

func TestUsersLifecycle(t *testing.T) {
	srv := newServer(t)

	if code, body := call(t, "GET", srv.URL+"/users/1", nil); code != 200 || !strings.Contains(string(body), "Ada Lovelace") {
		t.Fatalf("seed user: %d %s", code, body)
	}

	code, body := call(t, "POST", srv.URL+"/users", map[string]any{"name": "Grace", "email": "g@example.com"})
	if code != 201 {
		t.Fatalf("create: %d %s", code, body)
	}
	var created User
	json.Unmarshal(body, &created)
	if created.ID != 2 || created.Name != "Grace" {
		t.Errorf("unexpected created user: %+v", created)
	}

	_, body = call(t, "GET", srv.URL+"/users?limit=1", nil)
	var list []User
	json.Unmarshal(body, &list)
	if len(list) != 1 || list[0].ID != 1 {
		t.Errorf("limit=1 should return only the first user: %s", body)
	}

	if code, _ := call(t, "DELETE", srv.URL+"/users/2", nil); code != 204 {
		t.Errorf("delete: %d, want 204", code)
	}
	if code, _ := call(t, "GET", srv.URL+"/users/2", nil); code != 404 {
		t.Errorf("deleted user should be 404, got %d", code)
	}
}

func TestUserValidation(t *testing.T) {
	srv := newServer(t)
	if code, _ := call(t, "POST", srv.URL+"/users", map[string]any{"name": ""}); code != 422 {
		t.Errorf("empty name should be rejected with 422, got %d", code)
	}
	if code, _ := call(t, "GET", srv.URL+"/users?limit=0", nil); code != 422 {
		t.Errorf("limit=0 should be rejected with 422, got %d", code)
	}
}

// ---- the spec -------------------------------------------------------------

func TestSpecIsOpenAPI30WithAllOperations(t *testing.T) {
	spec, err := SpecYAML()
	if err != nil {
		t.Fatal(err)
	}
	s := string(spec)
	if !strings.Contains(s, "openapi: 3.0") {
		t.Error("spec should be downgraded to OpenAPI 3.0, which Guardian reads")
	}
	for _, want := range []string{"/users:", "/users/{id}:",
		"operationId: list-users", "operationId: delete-user", "operationId: create-user"} {
		if !strings.Contains(s, want) {
			t.Errorf("spec is missing %q", want)
		}
	}
	if strings.Contains(s, "$schema") {
		t.Error("responses should not carry a $schema property")
	}
}

// CI diffs specs generated on two different commits. If generation were not
// deterministic, every PR would show phantom changes.
func TestSpecIsDeterministic(t *testing.T) {
	first, _ := SpecYAML()
	for i := 0; i < 5; i++ {
		again, _ := SpecYAML()
		if !bytes.Equal(first, again) {
			t.Fatal("SpecYAML() produced different output on a later call")
		}
	}
}

// The contract fields that the Guardian scenarios depend on.
func TestSpecDescribesTheUserContract(t *testing.T) {
	spec, _ := SpecYAML()
	s := string(spec)
	for _, want := range []string{
		"    User:",
		"        email:", // User.email in the response schema
		"        age:",   // User.age
	} {
		if !strings.Contains(s, want) {
			t.Errorf("spec should contain %q", want)
		}
	}
}
