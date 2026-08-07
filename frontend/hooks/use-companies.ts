"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/query-keys";

export function useCompanies(page: number, perPage: number) {
  return useQuery({
    queryKey: queryKeys.companies.list(page, perPage),
    queryFn: async () => {
      const result = await api.listCompanies(page, perPage);
      if (!result.ok) throw new Error(result.error.message);
      return result.value;
    },
  });
}

export function useCompany(id: string) {
  return useQuery({
    queryKey: queryKeys.companies.detail(id),
    queryFn: async () => {
      const result = await api.getCompany(id);
      if (!result.ok) throw new Error(result.error.message);
      return result.value;
    },
  });
}

// The exact text that goes into every prompt (ADR-0001) — what a consultant
// reads is what the LLM reads.
export function useCompanyContext(id: string) {
  return useQuery({
    queryKey: queryKeys.companies.context(id),
    queryFn: async () => {
      const result = await api.getCompanyContext(id);
      if (!result.ok) throw new Error(result.error.message);
      return result.value;
    },
  });
}
