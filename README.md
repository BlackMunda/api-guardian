# Guardian

Guardian detects **breaking changes between two versions of an OpenAPI spec**,
so a pull request that would break API consumers can be caught before merge.

> **Stage 1:** a CLI that diffs two OpenAPI 3.0 files, classifies each change as
> breaking or safe, and exits non-zero if anything is breaking.
> **Stage 2:** a GitHub Action that runs it on every pull request and posts the
> result as a comment (see [GitHub Action](#github-action-stage-2)).

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

Options: `--format text|json|markdown` (flags go before the file names).
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
| `internal/report` | Render changes as text, JSON or a Markdown PR comment |
| `internal/github` | Minimal GitHub client: post/update a PR comment, detect the PR from the environment |
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


## GitHub Action (Stage 2)

On every pull request that touches the spec, the Action:

1. reads the spec **as it was on the base branch** (`scripts/base-spec.sh`),
2. compares it with the spec in the PR (`guardian comment`),
3. posts the report as a PR comment, or **edits the same comment** on later
   pushes (found via a hidden `<!-- guardian-report -->` marker),
4. fails the check if anything is breaking (exit code 1).

The report is also written to the workflow run's job summary.

### Use it

```yaml
# .github/workflows/guardian.yml  (a full example is in this repo)
on:
  pull_request:
    paths: ["api/openapi.yaml"]
permissions:
  contents: read
  pull-requests: write
jobs:
  api-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }      # needed to read the base branch's spec
      - uses: ./                      # or: <owner>/guardian@v1 from another repo
        with:
          spec-path: api/openapi.yaml
```

To make breaking changes **block merging**, add the `api-check` job as a
required status check under *Settings → Branches → Branch protection*.

### Try it locally (no GitHub needed)

```bash
guardian comment --dry-run base.yaml pr.yaml     # prints the comment, posts nothing
scripts/test-base-spec.sh                        # tests the base-spec extraction
scripts/e2e-local.sh                             # full rehearsal against a fake GitHub API
```

### Demo for the instructor

1. `git checkout -b remove-email` and delete the `email` line from `api/openapi.yaml`.
2. Push and open a pull request.
3. Watch the Guardian check fail and the bot comment appear.
4. Restore the line and push again: the **same** comment updates and the check goes green.

### Things to know

- **Commit `go.sum`** (run `go mod tidy` once). The Action builds Guardian from
  source and needs it.
- **Pull requests from forks** get a read-only token, so no comment can be
  posted. Guardian then prints a warning and still fails or passes by the diff
  result; the report is in the job summary.
- If the spec file does not exist on the base branch (the PR adds it), the
  check is skipped.
- Only one spec file per Action step. Use several steps for several specs.

## Known limitations (planned for later stages)

- OpenAPI 3.0 only; 3.1 support depends on the parsing library.
- Only JSON bodies and `2xx` responses are compared.
- `oneOf` / `anyOf` schemas are compared by type name only, not by variants.
- Enum narrowing, `format`/`minLength`-style constraints, and removed request
  fields are not yet analysed.
- It trusts the spec: it does not check that the *code* matches the spec (drift detection).

## Roadmap

1. **Stage 1 (done):** spec diff CLI + rules + tests
2. **Stage 2 (done):** GitHub Action that posts the report as a PR comment
3. **Stage 3:** generate base/PR specs from a demo Go API (Huma) in CI
4. **Stage 4:** drift detection (code vs. spec) and runtime contract tests
