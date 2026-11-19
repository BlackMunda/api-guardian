# Guardian demo API

A tiny users/orders API used to show Guardian catching breaking changes **in Go
code**. It is built with [Huma](https://huma.rocks), which generates the OpenAPI
document from the request/response structs, so the spec cannot drift from the code.

```bash
go mod tidy                  # first time only
go test ./...
go run ./cmd/api openapi     # print the OpenAPI 3.0 spec
go run ./cmd/api serve       # http://localhost:8888  (docs at /docs)
```

| Method | Path | |
|---|---|---|
| GET | `/users` | list (`?limit=`) |
| GET | `/users/{id}` | one user |
| POST | `/users` | create |
| DELETE | `/users/{id}` | delete |
| GET | `/orders/{id}` | one order |
| POST | `/orders` | place an order |

## Where the contract lives

The **exported structs** are the contract: `User`, `NewUser`, `Order`, `NewOrder`
and the `*Input` structs. Changing a field there changes the generated spec.
Handlers and storage are in-memory and not part of the contract.

| File | |
|---|---|
| `internal/api/api.go` | builds the API; resource files register themselves |
| `internal/api/users.go` | users endpoints and structs |
| `internal/api/orders.go` | orders endpoints and structs |
| `cmd/api/main.go` | `openapi` and `serve` commands |
| `scenarios/*.patch` | code changes that Guardian should flag (see the main README) |

## Adding a resource

Create `internal/api/<thing>.go` with an `init()` that calls `register(registerThing)`.
You do not need to edit `api.go`.
