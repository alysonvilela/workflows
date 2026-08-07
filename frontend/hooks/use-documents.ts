"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { queryKeys } from "@/lib/query-keys";

// Polls while ingestion could still be in flight for any Document, so the
// parsing -> ready/failed transition shows up without a reload; stops once
// everything has settled.
export function useDocuments(companyId: string) {
  return useQuery({
    queryKey: queryKeys.documents.list(companyId),
    queryFn: async () => {
      const result = await api.listDocuments(companyId);
      if (!result.ok) throw new Error(result.error.message);
      return result.value;
    },
    refetchInterval: (query) => {
      const stillParsing = query.state.data?.documents.some((d) => d.status === "parsing");
      return stillParsing ? 1500 : false;
    },
  });
}

// Fetches a single Document's full record, including raw_text — omitted
// from the list response to keep polling it cheap.
export function useDocument(companyId: string, id: string, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.documents.detail(companyId, id),
    queryFn: async () => {
      const result = await api.getDocument(companyId, id);
      if (!result.ok) throw new Error(result.error.message);
      return result.value;
    },
    enabled,
  });
}
