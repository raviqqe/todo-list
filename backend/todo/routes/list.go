package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/exp/slog"

	"github.com/dackroyd/todo-list/backend/todo"
)

// Response to be encoded and transmitted to the client.
type Response struct {
	Body interface{}
}

// ErrorResponse to be encoded and transmitted to the client on failure.
type ErrorResponse struct {
	Status int
	Error  string
	Cause  error
}

// ItemsBody included when retrieving TODO items.
type ItemsBody struct {
	Items []todo.Item `json:"items"`
}

// ListRepository where TODO lists and items are stored.
type ListRepository interface {
	Items(ctx context.Context, listID string) ([]todo.Item, error)
}

// ListsAPI manages TODO lists.
type ListsAPI struct {
	repo ListRepository
}

// NewListAPI for managing TODO lists.
func NewListAPI(repo ListRepository) *ListsAPI {
	return &ListsAPI{repo: repo}
}

// Items of a TODO list.
func (l *ListsAPI) Items(w http.ResponseWriter, r *http.Request) {
	h := func(w http.ResponseWriter, r *http.Request) (*Response, *ErrorResponse) {
		params := httprouter.ParamsFromContext(r.Context())

		listID := strings.TrimSpace(params.ByName("list_id"))
		if listID == "" {
			return nil, &ErrorResponse{Status: http.StatusBadRequest, Error: `"list_id" path param must not be blank`}
		}

		items, err := l.repo.Items(r.Context(), listID)
		if err != nil {
			return nil, &ErrorResponse{Status: http.StatusInternalServerError, Error: "Internal Server Error", Cause: err}
		}

		if items == nil {
			// Ensure we get an empty array in the response, not `null`
			items = []todo.Item{}
		}

		return &Response{Body: &ItemsBody{Items: items}}, nil
	}

	handleRequest(h)(w, r)
}

func handleRequest(h func(w http.ResponseWriter, r *http.Request) (*Response, *ErrorResponse)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		resp, err := h(w, r)
		if err != nil {
			type errPayload struct {
				Error string `json:"error"`
			}

			w.WriteHeader(err.Status)
			enc.Encode(&errPayload{Error: err.Error})

			if c := err.Cause; c != nil {
				addLogAttrs(r.Context(), slog.String("error_cause", c.Error()))
			}

			return
		}

		enc.Encode(resp.Body)
	}
}