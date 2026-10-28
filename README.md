# Guardian

Guardian detects **breaking changes between two versions of an OpenAPI spec**,
so a pull request that would break API consumers can be caught before merge.

> Stage 1 (this repo): a CLI that diffs two OpenAPI 3.0 files, classifies each
> change as breaking or safe, and exits non-zero if anything is breaking.

## Quick start

```bash
go mod tidy                                   # first time only: fetch dependencies
go test ./...                                 # run the test suite
go run ./cmd/guardian diff testdata/demo/base.yaml testdata/demo/pr.yaml
```

```text
❌ BREAKING  POST /orders
   Request field 'currency' is now required

❌ BREAKING  GET /users
   Response field '[].email' was removed

❌ BREAKING  GET /users/{id}
   Response field 'email' was removed

✅ SAFE      GET /users
   New optional query parameter 'limit'

Result: 3 breaking, 1 safe  (exit code 1)
```

Options: `--format json` for machine-readable output (flags go before the file names).
Exit codes: `0` no breaking changes, `1` breaking changes found, `2` error.

## How it works

```text
base.yaml ─┐
           ├─► loader ─► diff ─► rules ─► report ─► terminal / JSON / exit code
pr.yaml  ──┘   (parse)   (what    (is it    (render)
                         changed?) breaking?)
```

| Package | Responsibility |
|---|---|
| `internal/loader` | Parse and validate a spec (kin-openapi), resolving `$ref`s |
| `internal/diff`   | Compare two specs and emit `[]Change` (no judgement) |
| `internal/rules`  | Map each change `Kind` to Breaking/Safe, with a reason |
| `internal/report` | Render changes as text or JSON |
| `internal/check`  | Glue: load → diff → classify (used by CLI and tests) |

## The rules

| Change | Verdict | Why |
|---|---|---|
| Endpoint removed | breaking | clients get 404/405 |
| Success response removed | breaking | clients expect that status code |
| Response field removed | breaking | clients read it |
| Response field no longer required | breaking | clients assume it is present |
| New required request field / parameter | breaking | existing clients don't send it |
| Optional request field / parameter made required | breaking | clients that omitted it are rejected |
| Request body newly required | breaking | clients that send none are rejected |
| Field or parameter type changed | breaking | clients use the old type |
| New endpoint | safe | nobody calls it yet |
| New optional request field / parameter | safe | existing requests stay valid |
| New response field | safe | clients ignore unknown fields |

Unknown change kinds fail closed (treated as breaking).
Renaming a path parameter (`{id}` → `{userId}`) is **not** breaking: it never appears on the wire.

## Known limitations (planned for later stages)

- OpenAPI 3.0 only; 3.1 support depends on the parsing library.
- Only JSON bodies and `2xx` responses are compared.
- `oneOf` / `anyOf` schemas are compared by type name only, not by variants.
- Enum narrowing, `format`/`minLength`-style constraints, and removed request
  fields are not yet analysed.
- It trusts the spec: it does not check that the *code* matches the spec (drift detection).

## Roadmap

1. **Stage 1 (done):** spec diff CLI + rules + tests
2. **Stage 2:** GitHub Action that posts the report as a PR comment
3. **Stage 3:** generate base/PR specs from a demo Go API (Huma) in CI
4. **Stage 4:** drift detection (code vs. spec) and runtime contract tests
