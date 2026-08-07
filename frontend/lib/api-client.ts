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

export const api = {
  health,
  listCompanies,
  getCompany,
  createCompany,
  renameCompany,
  deleteCompany,
};
