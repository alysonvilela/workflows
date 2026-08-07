package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/discovery/backend/internal/domain"
	"github.com/discovery/backend/internal/ingest"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"
)

// DocumentStore is the persistence seam the Document handlers depend on,
// beyond the ingestion pipeline itself.
type DocumentStore interface {
	ListDocuments(ctx context.Context, companyID string, page, perPage int) ([]domain.Document, int, error)
	GetDocument(ctx context.Context, companyID, id string) (domain.Document, error)
}

func registerDocumentRoutes(r chi.Router, companies CompanyStore, documents DocumentStore, ingestor *ingest.Ingestor) {
	r.Route("/companies/{companyId}/documents", func(r chi.Router) {
		r.Post("/", uploadDocumentsHandler(companies, ingestor))
		r.Get("/", listDocumentsHandler(companies, documents))
		r.Get("/{documentId}", getDocumentHandler(companies, documents))
	})
}

type documentResponse struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	Error     *string   `json:"error,omitempty"`
	Checksum  string    `json:"checksum"`
	RawText   *string   `json:"raw_text,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newDocumentResponse(d domain.Document) documentResponse {
	return documentResponse{
		ID:        d.ID,
		CompanyID: d.CompanyID,
		Title:     d.Title,
		Kind:      d.Kind,
		Status:    d.Status,
		Error:     d.Error,
		Checksum:  d.Checksum,
		RawText:   d.RawText,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

type documentEnvelope struct {
	Data documentResponse `json:"data"`
}

type listDocumentsResponse struct {
	Data []documentResponse `json:"data"`
	Meta struct {
		Pagination paginationMeta `json:"pagination"`
	} `json:"meta"`
}

// uploadResult reports one file's outcome inside a multi-file upload. A file
// rejected outright (unsupported type, too large) carries Error and no
// Document; a file that reached the pipeline always carries a Document,
// whose own Status/Error record a parsing failure.
type uploadResult struct {
	Filename string             `json:"filename"`
	Document *documentResponse  `json:"document,omitempty"`
	Error    *string            `json:"error,omitempty"`
}

type uploadDocumentsResponse struct {
	Data []uploadResult `json:"data"`
}

// companyOr404 loads the Company a Document route is nested under, writing
// a 404 and returning ok=false if it does not exist.
func companyOr404(w http.ResponseWriter, r *http.Request, companies CompanyStore) (domain.Company, bool) {
	company, err := companies.GetCompany(r.Context(), chi.URLParam(r, "companyId"))
	if err != nil {
		writeStoreError(w, err)
		return domain.Company{}, false
	}
	return company, true
}

func uploadDocumentsHandler(companies CompanyStore, ingestor *ingest.Ingestor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		company, ok := companyOr404(w, r, companies)
		if !ok {
			return
		}

		type buffered struct {
			filename string
			data     []byte
		}

		mr, err := r.MultipartReader()
		if err != nil {
			writeError(w, http.StatusBadRequest, "expected a multipart/form-data request")
			return
		}

		var files []buffered
		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				writeError(w, http.StatusBadRequest, "malformed multipart body")
				return
			}
			if part.FormName() != "files" || part.FileName() == "" {
				_ = part.Close()
				continue
			}

			data, err := io.ReadAll(io.LimitReader(part, ingestor.MaxUploadBytes+1))
			_ = part.Close()
			if err != nil {
				writeError(w, http.StatusBadRequest, "could not read uploaded file")
				return
			}
			files = append(files, buffered{filename: part.FileName(), data: data})
		}

		if len(files) == 0 {
			writeError(w, http.StatusBadRequest, "no files uploaded — expected one or more \"files\" parts")
			return
		}

		results := make([]uploadResult, len(files))
		var g errgroup.Group
		for i, f := range files {
			g.Go(func() error {
				doc, err := ingestor.Ingest(r.Context(), company.ID, f.filename, f.data)
				if err != nil {
					message := err.Error()
					results[i] = uploadResult{Filename: f.filename, Error: &message}
					return nil
				}
				resp := newDocumentResponse(doc)
				results[i] = uploadResult{Filename: f.filename, Document: &resp}
				return nil
			})
		}
		_ = g.Wait()

		writeJSON(w, http.StatusCreated, uploadDocumentsResponse{Data: results})
	}
}

func listDocumentsHandler(companies CompanyStore, documents DocumentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		company, ok := companyOr404(w, r, companies)
		if !ok {
			return
		}

		page, perPage := parsePagination(r)
		docs, total, err := documents.ListDocuments(r.Context(), company.ID, page, perPage)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		data := make([]documentResponse, len(docs))
		for i, d := range docs {
			data[i] = newDocumentResponse(d)
		}

		totalPages := (total + perPage - 1) / perPage
		if totalPages == 0 {
			totalPages = 1
		}

		resp := listDocumentsResponse{Data: data}
		resp.Meta.Pagination = paginationMeta{Page: page, PerPage: perPage, Total: total, TotalPages: totalPages}
		writeJSON(w, http.StatusOK, resp)
	}
}

func getDocumentHandler(companies CompanyStore, documents DocumentStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		company, ok := companyOr404(w, r, companies)
		if !ok {
			return
		}

		doc, err := documents.GetDocument(r.Context(), company.ID, chi.URLParam(r, "documentId"))
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				writeError(w, http.StatusNotFound, "document not found")
				return
			}
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, documentEnvelope{Data: newDocumentResponse(doc)})
	}
}
