// Package api is a small demo HTTP API built with Huma.
//
// Huma generates the OpenAPI document FROM THE GO CODE (the request/response
// structs below), so the spec can never drift from the implementation. That is
// what lets Guardian catch a breaking change in a pull request even though
// nobody edited a YAML file.
package api

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// registrars lets each resource file (users.go, orders.go, ...) add its own
// operations from an init() function, without anyone editing this file.
var registrars []func(huma.API)

func register(f func(huma.API)) { registrars = append(registrars, f) }

// New builds the API on top of the given mux.
func New(mux *http.ServeMux) huma.API {
	cfg := huma.DefaultConfig("Guardian Demo API", "1.0.0")
	// Huma normally adds a "$schema" property to every response. Switching it
	// off keeps the contract limited to our own fields.
	cfg.CreateHooks = nil

	api := humago.New(mux, cfg)
	for _, r := range registrars {
		r(api)
	}
	return api
}

// SpecYAML returns the OpenAPI 3.0 document generated from the code.
// (Huma natively produces 3.1; Guardian reads 3.0, so we downgrade.)
func SpecYAML() ([]byte, error) {
	return New(http.NewServeMux()).OpenAPI().DowngradeYAML()
}
