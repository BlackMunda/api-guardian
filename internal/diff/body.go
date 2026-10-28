package diff

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ---- request body ---------------------------------------------------------

func bodySchema(op *openapi3.Operation) (*openapi3.SchemaRef, bool) {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil, false
	}
	rb := op.RequestBody.Value
	if mt := jsonMedia(rb.Content); mt != nil {
		return mt.Schema, rb.Required
	}
	return nil, rb.Required
}

func compareRequestBody(c *collector, b, p opEntry) {
	bs, _ := bodySchema(b.Op)
	ps, pReq := bodySchema(p.Op)
	if bs == nil && ps != nil {
		if pReq {
			c.add(RequestBodyAddedRequired, "request.body", "Request body is now required")
		}
		return
	}
	if bs == nil || ps == nil {
		return
	}
	compareFields(c, "request.body", request, flatten(bs), flatten(ps))
}

// ---- responses ------------------------------------------------------------

func responseMap(op *openapi3.Operation) map[string]*openapi3.Response {
	out := map[string]*openapi3.Response{}
	if op.Responses == nil {
		return out
	}
	for code, ref := range op.Responses.Map() {
		if ref != nil && ref.Value != nil {
			out[code] = ref.Value
		}
	}
	return out
}

func responseSchema(r *openapi3.Response) *openapi3.SchemaRef {
	if r == nil {
		return nil
	}
	if mt := jsonMedia(r.Content); mt != nil {
		return mt.Schema
	}
	return nil
}

func compareResponses(c *collector, b, p opEntry) {
	bm, pm := responseMap(b.Op), responseMap(p.Op)
	for _, code := range sortedKeys(bm) {
		if !strings.HasPrefix(code, "2") { // only success responses for now
			continue
		}
		pr, ok := pm[code]
		if !ok {
			c.add(ResponseRemoved, "response."+code, "Success response %s was removed", code)
			continue
		}
		bs := responseSchema(bm[code])
		if bs == nil {
			continue
		}
		compareFields(c, "response."+code, response, flatten(bs), flatten(responseSchema(pr)))
	}
}
