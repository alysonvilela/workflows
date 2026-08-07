export const queryKeys = {
  health: ["health"] as const,
  companies: {
    all: ["companies"] as const,
    list: (page: number, perPage: number) => ["companies", "list", page, perPage] as const,
    detail: (id: string) => ["companies", "detail", id] as const,
    context: (id: string) => ["companies", "detail", id, "context"] as const,
  },
  documents: {
    list: (companyId: string) => ["companies", companyId, "documents"] as const,
    detail: (companyId: string, id: string) => ["companies", companyId, "documents", id] as const,
  },
};
