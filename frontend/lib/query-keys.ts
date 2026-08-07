export const queryKeys = {
  health: ["health"] as const,
  companies: {
    all: ["companies"] as const,
    list: (page: number, perPage: number) => ["companies", "list", page, perPage] as const,
    detail: (id: string) => ["companies", "detail", id] as const,
  },
};
