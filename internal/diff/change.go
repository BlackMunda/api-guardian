// Package diff compares two OpenAPI documents and produces a list of Changes.
//
// It only *describes* what changed. Deciding whether a change is breaking is
// the job of the rules package, which keeps "what changed" and "does it
// matter" as separate concerns.
package diff

// Severity says whether a change can break existing API consumers.
type Severity int

const (
	Safe Severity = iota
	Breaking
)

func (s Severity) String() string {
	if s == Breaking {
		return "breaking"
	}
	return "safe"
}

// MarshalText makes Severity appear as "breaking"/"safe" in JSON output.
func (s Severity) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

// Kind is the machine-readable type of a change. The rules package maps each
// Kind to a Severity.
type Kind string

const (
	EndpointRemoved           Kind = "endpoint_removed"
	EndpointAdded             Kind = "endpoint_added"
	ResponseRemoved           Kind = "response_removed"
	ResponseFieldRemoved      Kind = "response_field_removed"
	ResponseFieldAdded        Kind = "response_field_added"
	ResponseFieldMadeOptional Kind = "response_field_made_optional"
	RequestFieldAddedRequired Kind = "request_field_added_required"
	RequestFieldAddedOptional Kind = "request_field_added_optional"
	RequestFieldMadeRequired  Kind = "request_field_made_required"
	RequestBodyAddedRequired  Kind = "request_body_added_required"
	ParamAddedRequired        Kind = "param_added_required"
	ParamAddedOptional        Kind = "param_added_optional"
	ParamMadeRequired         Kind = "param_made_required"
	FieldTypeChanged          Kind = "field_type_changed"
)

// AllKinds lists every Kind. A test uses it to make sure each Kind has a rule.
var AllKinds = []Kind{
	EndpointRemoved, EndpointAdded, ResponseRemoved,
	ResponseFieldRemoved, ResponseFieldAdded, ResponseFieldMadeOptional,
	RequestFieldAddedRequired, RequestFieldAddedOptional, RequestFieldMadeRequired,
	RequestBodyAddedRequired, ParamAddedRequired, ParamAddedOptional,
	ParamMadeRequired, FieldTypeChanged,
}

// Change is the one type the whole pipeline revolves around.
type Change struct {
	Path     string   `json:"path"`     // "/users/{id}"
	Method   string   `json:"method"`   // "GET"
	Location string   `json:"location"` // "response.200.email"
	Kind     Kind     `json:"kind"`     // "response_field_removed"
	Severity Severity `json:"severity"` // filled in by rules.Classify
	Message  string   `json:"message"`  // human-readable explanation
}
