package api

import (
	"context"
	"net/http"
	"sort"
	"sync"

	"github.com/danielgtaylor/huma/v2"
)

func init() { register(registerUsers) }

// ---- what clients see: THIS is the API contract --------------------------

// User is returned by the users endpoints.
type User struct {
	ID    int    `json:"id" doc:"Unique user ID" example:"1"`
	Name  string `json:"name" doc:"Display name" example:"Ada Lovelace"`
	Email string `json:"email" doc:"Contact email" example:"ada@example.com"`
	Age   int    `json:"age,omitempty" doc:"Age in years, if the user chose to share it"`
}

// NewUser is the request body for creating a user.
type NewUser struct {
	Name  string `json:"name" minLength:"1" doc:"Display name"`
	Email string `json:"email,omitempty" doc:"Contact email"`
}

type ListUsersInput struct {
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Maximum number of users to return"`
}
type ListUsersOutput struct{ Body []User }

type GetUserInput struct {
	ID int `path:"id" doc:"User ID"`
}
type GetUserOutput struct{ Body User }

type CreateUserInput struct{ Body NewUser }
type CreateUserOutput struct{ Body User }

type DeleteUserInput struct {
	ID int `path:"id" doc:"User ID"`
}
type DeleteUserOutput struct{}

// ---- implementation (in-memory) ------------------------------------------

type userStore struct {
	mu   sync.Mutex
	byID map[int]User
	next int
}

func newUserStore() *userStore {
	return &userStore{
		byID: map[int]User{1: {ID: 1, Name: "Ada Lovelace", Email: "ada@example.com"}},
		next: 2,
	}
}

func registerUsers(api huma.API) {
	db := newUserStore()

	huma.Register(api, huma.Operation{
		OperationID: "list-users", Method: http.MethodGet, Path: "/users",
		Summary: "List users", Tags: []string{"users"},
	}, func(ctx context.Context, in *ListUsersInput) (*ListUsersOutput, error) {
		db.mu.Lock()
		defer db.mu.Unlock()
		out := make([]User, 0, len(db.byID))
		for _, u := range db.byID {
			out = append(out, u)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
		if len(out) > in.Limit {
			out = out[:in.Limit]
		}
		return &ListUsersOutput{Body: out}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-user", Method: http.MethodGet, Path: "/users/{id}",
		Summary: "Get one user", Tags: []string{"users"},
	}, func(ctx context.Context, in *GetUserInput) (*GetUserOutput, error) {
		db.mu.Lock()
		defer db.mu.Unlock()
		u, ok := db.byID[in.ID]
		if !ok {
			return nil, huma.Error404NotFound("user not found")
		}
		return &GetUserOutput{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "create-user", Method: http.MethodPost, Path: "/users",
		Summary: "Create a user", Tags: []string{"users"}, DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *CreateUserInput) (*CreateUserOutput, error) {
		db.mu.Lock()
		defer db.mu.Unlock()
		u := User{ID: db.next, Name: in.Body.Name, Email: in.Body.Email}
		db.byID[u.ID] = u
		db.next++
		return &CreateUserOutput{Body: u}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-user", Method: http.MethodDelete, Path: "/users/{id}",
		Summary: "Delete a user", Tags: []string{"users"}, DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *DeleteUserInput) (*DeleteUserOutput, error) {
		db.mu.Lock()
		defer db.mu.Unlock()
		if _, ok := db.byID[in.ID]; !ok {
			return nil, huma.Error404NotFound("user not found")
		}
		delete(db.byID, in.ID)
		return &DeleteUserOutput{}, nil
	})
}
