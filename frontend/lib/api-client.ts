import { fail, ok, type Result } from "@/lib/result";

const baseUrl = import.meta.env.VITE_API_URL;
if (!baseUrl) {
  throw new Error("VITE_API_URL is not set — copy .env.example to .env and restart the dev server");
}

export interface HealthStatus {
  status: "ok" | "degraded";
  database: string;
  storage: string;
}

function isHealthStatus(body: unknown): body is HealthStatus {
  return (
    typeof body === "object" &&
    body !== null &&
    "status" in body &&
    "database" in body &&
    "storage" in body
  );
}

// /health reports its own status field instead of only using the HTTP status
// code — a degraded API still answers with a 503 and a full, meaningful body,
// so this reads the body regardless of response.ok.
async function health(): Promise<Result<HealthStatus, string>> {
  let response: Response;
  try {
    response = await fetch(`${baseUrl}/health`);
  } catch (error) {
    return fail(error instanceof Error ? error.message : "network error");
  }

  const body: unknown = await response.json().catch(() => undefined);
  return isHealthStatus(body) ? ok(body) : fail(response.statusText);
}

/* ── shared request helper ──────────────────────────────────────────── */

// What every non-2xx response from this API looks like: a message, plus
// optional per-field messages for inline form validation.
export interface ApiError {
  message: string;
  fields?: Record<string, string>;
}

// Thrown by mutation hooks so a component can recover `fields` via
// `error instanceof ApiRequestError` and render them next to the inputs,
// instead of collapsing every failure into one generic message.
export class ApiRequestError extends Error {
  fields?: Record<string, string>;

  constructor(apiError: ApiError) {
    super(apiError.message);
    this.name = "ApiRequestError";
    this.fields = apiError.fields;
  }
}

function isErrorEnvelope(body: unknown): body is { error: ApiError } {
  return (
    typeof body === "object" &&
    body !== null &&
    "error" in body &&
    typeof (body as { error: unknown }).error === "object"
  );
}

async function request<T>(path: string, init?: RequestInit): Promise<Result<T, ApiError>> {
  const headers = new Headers(init?.headers);
  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(`${baseUrl}${path}`, { ...init, headers });
  } catch (error) {
    return fail({ message: error instanceof Error ? error.message : "network error" });
  }

  if (response.status === 204) {
    return ok(undefined as T);
  }

  const body: unknown = await response.json().catch(() => undefined);

  if (!response.ok) {
    return fail(
      isErrorEnvelope(body) ? body.error : { message: response.statusText || "request failed" },
    );
  }

  return ok(body as T);
}

/* ── companies ───────────────────────────────────────────────────────── */

export interface Company {
  id: string;
  name: string;
  slug: string;
  documentCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface Pagination {
  page: number;
  perPage: number;
  total: number;
  totalPages: number;
}

interface CompanyDTO {
  id: string;
  name: string;
  slug: string;
  document_count: number;
  created_at: string;
  updated_at: string;
}

function fromCompanyDTO(dto: CompanyDTO): Company {
  return {
    id: dto.id,
    name: dto.name,
    slug: dto.slug,
    documentCount: dto.document_count,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  };
}

async function listCompanies(
  page = 1,
  perPage = 20,
): Promise<Result<{ companies: Company[]; pagination: Pagination }, ApiError>> {
  const result = await request<{
    data: CompanyDTO[];
    meta: { pagination: { page: number; per_page: number; total: number; total_pages: number } };
  }>(`/companies?page=${page}&per_page=${perPage}`);
  if (!result.ok) return result;

  return ok({
    companies: result.value.data.map(fromCompanyDTO),
    pagination: {
      page: result.value.meta.pagination.page,
      perPage: result.value.meta.pagination.per_page,
      total: result.value.meta.pagination.total,
      totalPages: result.value.meta.pagination.total_pages,
    },
  });
}

async function getCompany(id: string): Promise<Result<Company, ApiError>> {
  const result = await request<{ data: CompanyDTO }>(`/companies/${id}`);
  return result.ok ? ok(fromCompanyDTO(result.value.data)) : result;
}

async function createCompany(name: string): Promise<Result<Company, ApiError>> {
  const result = await request<{ data: CompanyDTO }>("/companies", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
  return result.ok ? ok(fromCompanyDTO(result.value.data)) : result;
}

async function renameCompany(id: string, name: string): Promise<Result<Company, ApiError>> {
  const result = await request<{ data: CompanyDTO }>(`/companies/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ name }),
  });
  return result.ok ? ok(fromCompanyDTO(result.value.data)) : result;
}

async function deleteCompany(id: string): Promise<Result<void, ApiError>> {
  return request<void>(`/companies/${id}`, { method: "DELETE" });
}

/* ── documents ───────────────────────────────────────────────────────── */

export type DocumentStatus = "parsing" | "ready" | "failed";
export type DocumentKind = "markdown" | "csv";

export interface Document {
  id: string;
  companyId: string;
  title: string;
  kind: DocumentKind;
  status: DocumentStatus;
  error: string | null;
  checksum: string;
  rawText: string | null;
  createdAt: string;
  updatedAt: string;
}

interface DocumentDTO {
  id: string;
  company_id: string;
  title: string;
  kind: DocumentKind;
  status: DocumentStatus;
  error?: string | null;
  checksum: string;
  raw_text?: string | null;
  created_at: string;
  updated_at: string;
}

function fromDocumentDTO(dto: DocumentDTO): Document {
  return {
    id: dto.id,
    companyId: dto.company_id,
    title: dto.title,
    kind: dto.kind,
    status: dto.status,
    error: dto.error ?? null,
    checksum: dto.checksum,
    rawText: dto.raw_text ?? null,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  };
}

async function listDocuments(
  companyId: string,
  page = 1,
  perPage = 50,
): Promise<Result<{ documents: Document[]; pagination: Pagination }, ApiError>> {
  const result = await request<{
    data: DocumentDTO[];
    meta: { pagination: { page: number; per_page: number; total: number; total_pages: number } };
  }>(`/companies/${companyId}/documents?page=${page}&per_page=${perPage}`);
  if (!result.ok) return result;

  return ok({
    documents: result.value.data.map(fromDocumentDTO),
    pagination: {
      page: result.value.meta.pagination.page,
      perPage: result.value.meta.pagination.per_page,
      total: result.value.meta.pagination.total,
      totalPages: result.value.meta.pagination.total_pages,
    },
  });
}

async function getDocument(companyId: string, id: string): Promise<Result<Document, ApiError>> {
  const result = await request<{ data: DocumentDTO }>(`/companies/${companyId}/documents/${id}`);
  return result.ok ? ok(fromDocumentDTO(result.value.data)) : result;
}

async function deleteDocument(companyId: string, id: string): Promise<Result<void, ApiError>> {
  return request<void>(`/companies/${companyId}/documents/${id}`, { method: "DELETE" });
}

async function resummariseDocument(
  companyId: string,
  id: string,
): Promise<Result<Document, ApiError>> {
  const result = await request<{ data: DocumentDTO }>(
    `/companies/${companyId}/documents/${id}/resummarise`,
    { method: "POST" },
  );
  return result.ok ? ok(fromDocumentDTO(result.value.data)) : result;
}

/* ── company context ─────────────────────────────────────────────────── */

async function getCompanyContext(companyId: string): Promise<Result<string, ApiError>> {
  const result = await request<{ data: { text: string } }>(`/companies/${companyId}/context`);
  return result.ok ? ok(result.value.data.text) : result;
}

export interface UploadFileResult {
  filename: string;
  document: Document | null;
  error: string | null;
}

// A per-file failure (unsupported type, too large) never surfaces as an
// ApiRequestError — it comes back as one entry in the array, alongside
// whichever other files in the same batch succeeded.
async function uploadDocuments(
  companyId: string,
  files: File[],
): Promise<Result<UploadFileResult[], ApiError>> {
  const form = new FormData();
  for (const file of files) {
    form.append("files", file);
  }

  let response: Response;
  try {
    response = await fetch(`${baseUrl}/companies/${companyId}/documents`, {
      method: "POST",
      body: form,
    });
  } catch (error) {
    return fail({ message: error instanceof Error ? error.message : "network error" });
  }

  const body: unknown = await response.json().catch(() => undefined);

  if (!response.ok) {
    return fail(
      isErrorEnvelope(body) ? body.error : { message: response.statusText || "upload failed" },
    );
  }

  const envelope = body as {
    data: { filename: string; document?: DocumentDTO; error?: string }[];
  };
  return ok(
    envelope.data.map((r) => ({
      filename: r.filename,
      document: r.document ? fromDocumentDTO(r.document) : null,
      error: r.error ?? null,
    })),
  );
}

export const api = {
  health,
  listCompanies,
  getCompany,
  createCompany,
  renameCompany,
  deleteCompany,
  listDocuments,
  getDocument,
  uploadDocuments,
  deleteDocument,
  resummariseDocument,
  getCompanyContext,
};
