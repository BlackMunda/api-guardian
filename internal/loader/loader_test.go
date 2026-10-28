package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadValidSpecResolvesRefs(t *testing.T) {
	p := write(t, `
openapi: 3.0.3
info: {title: T, version: 1.0.0}
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: "#/components/schemas/User"}
components:
  schemas:
    User:
      type: object
      properties:
        id: {type: integer}
`)
	doc, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	schema := doc.Paths.Find("/users").Get.Responses.Status(200).Value.Content["application/json"].Schema
	if schema.Value == nil || schema.Value.Properties["id"] == nil {
		t.Error("expected $ref to be resolved to the User schema")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load("does-not-exist.yaml"); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestLoadInvalidSpec(t *testing.T) {
	// "info" requires title and version, so validation must reject this.
	p := write(t, "openapi: 3.0.3\ninfo: {}\npaths: {}\n")
	if _, err := Load(p); err == nil {
		t.Error("expected a validation error")
	}
}
