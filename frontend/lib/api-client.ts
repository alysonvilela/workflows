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

export const api = {
  health,
};
