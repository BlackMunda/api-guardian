package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestOrders(t *testing.T) {
	srv := newServer(t)
	code, body := call(t, "POST", srv.URL+"/orders", map[string]any{"userId": 1, "product": "keyboard", "quantity": 2})
	if code != 201 {
		t.Fatalf("create order: %d %s", code, body)
	}
	var o Order
	json.Unmarshal(body, &o)
	if code, _ := call(t, "GET", srv.URL+"/orders/"+strconv.Itoa(o.ID), nil); code != 200 {
		t.Errorf("get order: %d", code)
	}
	if code, _ := call(t, "GET", srv.URL+"/orders/9999", nil); code != 404 {
		t.Errorf("unknown order should be 404, got %d", code)
	}
	if code, _ := call(t, "POST", srv.URL+"/orders", map[string]any{"userId": 1, "product": "x", "quantity": 0}); code != 422 {
		t.Errorf("quantity 0 should be rejected with 422, got %d", code)
	}
}

func TestSpecIncludesOrders(t *testing.T) {
	spec, err := SpecYAML()
	if err != nil {
		t.Fatal(err)
	}
	s := string(spec)
	for _, want := range []string{
		"/orders:", "/orders/{id}:", "operationId: create-order", "operationId: get-order",
		"    NewOrder:",
		"        quantity:", // NewOrder.quantity in the request schema
	} {
		if !strings.Contains(s, want) {
			t.Errorf("spec is missing %q", want)
		}
	}
}
