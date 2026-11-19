package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"github.com/danielgtaylor/huma/v2"
)

func init() { register(registerOrders) }

// ---- what clients see: THIS is the API contract --------------------------

// Order is returned by the orders endpoints.
type Order struct {
	ID       int    `json:"id" doc:"Unique order ID" example:"100"`
	UserID   int    `json:"userId" doc:"The user who placed the order" example:"1"`
	Product  string `json:"product" doc:"What was ordered" example:"keyboard"`
	Quantity int    `json:"quantity" doc:"How many" example:"2"`
	Note     string `json:"note,omitempty" doc:"Optional delivery note"`
}

// NewOrder is the request body for placing an order.
type NewOrder struct {
	UserID   int    `json:"userId" minimum:"1" doc:"The user placing the order"`
	Product  string `json:"product" minLength:"1" doc:"What to order"`
	Quantity int    `json:"quantity" minimum:"1" doc:"How many"`
	Note     string `json:"note,omitempty" doc:"Optional delivery note"`
}

type GetOrderInput struct {
	ID int `path:"id" doc:"Order ID"`
}
type GetOrderOutput struct{ Body Order }

type CreateOrderInput struct{ Body NewOrder }
type CreateOrderOutput struct{ Body Order }

// ---- implementation (in-memory) ------------------------------------------

type orderStore struct {
	mu   sync.Mutex
	byID map[int]Order
	next int
}

func registerOrders(api huma.API) {
	db := &orderStore{byID: map[int]Order{}, next: 100}

	huma.Register(api, huma.Operation{
		OperationID: "get-order", Method: http.MethodGet, Path: "/orders/{id}",
		Summary: "Get one order", Tags: []string{"orders"},
	}, func(ctx context.Context, in *GetOrderInput) (*GetOrderOutput, error) {
		db.mu.Lock()
		defer db.mu.Unlock()
		o, ok := db.byID[in.ID]
		if !ok {
			return nil, huma.Error404NotFound("order " + strconv.Itoa(in.ID) + " not found")
		}
		return &GetOrderOutput{Body: o}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-order", Method: http.MethodPost, Path: "/orders",
		Summary: "Place an order", Tags: []string{"orders"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *CreateOrderInput) (*CreateOrderOutput, error) {
		db.mu.Lock()
		defer db.mu.Unlock()
		o := Order{
			ID: db.next, UserID: in.Body.UserID, Product: in.Body.Product,
			Quantity: in.Body.Quantity, Note: in.Body.Note,
		}
		db.byID[o.ID] = o
		db.next++
		return &CreateOrderOutput{Body: o}, nil
	})
}
