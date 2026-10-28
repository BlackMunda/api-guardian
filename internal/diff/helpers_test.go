package diff

import (
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestSortedKeys(t *testing.T) {
	got := sortedKeys(map[string]int{"b": 1, "c": 2, "a": 3})
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("got %v", got)
	}
}

func TestJSONMedia(t *testing.T) {
	plain := &openapi3.MediaType{}
	js := &openapi3.MediaType{}
	hal := &openapi3.MediaType{}

	if got := jsonMedia(openapi3.Content{"text/plain": plain}); got != nil {
		t.Error("non-JSON content should return nil")
	}
	if got := jsonMedia(openapi3.Content{"application/hal+json": hal}); got != hal {
		t.Error("should fall back to a +json type")
	}
	if got := jsonMedia(openapi3.Content{"application/hal+json": hal, "application/json": js}); got != js {
		t.Error("should prefer application/json")
	}
}
