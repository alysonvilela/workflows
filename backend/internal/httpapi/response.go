package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	defaultPerPage = 20
	maxPerPage     = 100
)

// paginationMeta is the project's pagination contract: every list endpoint
// wraps its rows in `data` and reports paging under `meta.pagination`.
type paginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		page = v
	}
	perPage = defaultPerPage
	if v, err := strconv.Atoi(r.URL.Query().Get("per_page")); err == nil && v > 0 && v <= maxPerPage {
		perPage = v
	}
	return page, perPage
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: errorBody{Message: message}})
}
