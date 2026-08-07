package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/discovery/backend/internal/domain"
	"github.com/go-chi/chi/v5"
)

// CompanyStore is the persistence seam the Company handlers depend on.
type CompanyStore interface {
	CreateCompany(ctx context.Context, name string) (domain.Company, error)
	ListCompanies(ctx context.Context, page, perPage int) ([]domain.Company, int, error)
	GetCompany(ctx context.Context, id string) (domain.Company, error)
	RenameCompany(ctx context.Context, id, name string) (domain.Company, error)
	DeleteCompany(ctx context.Context, id string) error
	GetCompanyContext(ctx context.Context, id string) (string, error)
}

func registerCompanyRoutes(r chi.Router, store CompanyStore) {
	r.Route("/companies", func(r chi.Router) {
		r.Post("/", createCompanyHandler(store))
		r.Get("/", listCompaniesHandler(store))
		r.Route("/{companyId}", func(r chi.Router) {
			r.Get("/", getCompanyHandler(store))
			r.Patch("/", renameCompanyHandler(store))
			r.Delete("/", deleteCompanyHandler(store))
			r.Get("/context", getCompanyContextHandler(store))
		})
	})
}

type companyResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	DocumentCount int       `json:"document_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newCompanyResponse(c domain.Company) companyResponse {
	return companyResponse{
		ID:            c.ID,
		Name:          c.Name,
		Slug:          c.Slug,
		DocumentCount: c.DocumentCount,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

type companyEnvelope struct {
	Data companyResponse `json:"data"`
}

type listCompaniesResponse struct {
	Data []companyResponse `json:"data"`
	Meta struct {
		Pagination paginationMeta `json:"pagination"`
	} `json:"meta"`
}

type companyRequest struct {
	Name string `json:"name"`
}

func createCompanyHandler(store CompanyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req companyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		company, err := store.CreateCompany(r.Context(), req.Name)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, companyEnvelope{Data: newCompanyResponse(company)})
	}
}

func listCompaniesHandler(store CompanyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, perPage := parsePagination(r)

		companies, total, err := store.ListCompanies(r.Context(), page, perPage)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		data := make([]companyResponse, len(companies))
		for i, c := range companies {
			data[i] = newCompanyResponse(c)
		}

		totalPages := (total + perPage - 1) / perPage
		if totalPages == 0 {
			totalPages = 1
		}

		resp := listCompaniesResponse{Data: data}
		resp.Meta.Pagination = paginationMeta{Page: page, PerPage: perPage, Total: total, TotalPages: totalPages}
		writeJSON(w, http.StatusOK, resp)
	}
}

func getCompanyHandler(store CompanyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		company, err := store.GetCompany(r.Context(), chi.URLParam(r, "companyId"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, companyEnvelope{Data: newCompanyResponse(company)})
	}
}

func renameCompanyHandler(store CompanyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req companyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		company, err := store.RenameCompany(r.Context(), chi.URLParam(r, "companyId"), req.Name)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, companyEnvelope{Data: newCompanyResponse(company)})
	}
}

func deleteCompanyHandler(store CompanyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := store.DeleteCompany(r.Context(), chi.URLParam(r, "companyId")); err != nil {
			writeStoreError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type companyContextResponse struct {
	Text string `json:"text"`
}

type companyContextEnvelope struct {
	Data companyContextResponse `json:"data"`
}

// getCompanyContextHandler returns the Company Context exactly as it goes
// into a prompt — one concatenated block of text, not the underlying jsonb
// entries — so the interface shows a consultant precisely what the LLM
// reads (ADR-0001).
func getCompanyContextHandler(store CompanyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		company, ok := companyOr404(w, r, store)
		if !ok {
			return
		}
		text, err := store.GetCompanyContext(r.Context(), company.ID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, companyContextEnvelope{Data: companyContextResponse{Text: text}})
	}
}

// writeStoreError maps the two sentinel failure shapes a store call can
// return — a field-level ValidationError and ErrNotFound — onto their HTTP
// status; anything else is a 500.
func writeStoreError(w http.ResponseWriter, err error) {
	var validationErr *domain.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
			Error: errorBody{Message: "validation failed", Fields: validationErr.Fields},
		})
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "company not found")
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
