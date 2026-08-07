"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ApiRequestError, type Document, type UploadFileResult } from "@/lib/api-client";
import { queryKeys } from "@/lib/query-keys";

export function useUploadDocuments(companyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (files: File[]): Promise<UploadFileResult[]> => {
      const result = await api.uploadDocuments(companyId, files);
      if (!result.ok) throw new ApiRequestError(result.error);
      return result.value;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.documents.list(companyId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.all });
      // Documents just uploaded still need to reach 'ready' before they
      // land in Company Context — the polling in useDocuments settles
      // things, but a stale context read isn't worth guarding against here.
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.context(companyId) });
    },
  });
}

// Removing a Document removes its Company Context entry in the same store
// transaction (ticket 06) — both queries are invalidated together so the
// interface never shows a context entry for a Document that is gone.
export function useDeleteDocument(companyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (documentId: string): Promise<void> => {
      const result = await api.deleteDocument(companyId, documentId);
      if (!result.ok) throw new ApiRequestError(result.error);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.documents.list(companyId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.all });
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.context(companyId) });
    },
  });
}

export function useResummariseDocument(companyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (documentId: string): Promise<Document> => {
      const result = await api.resummariseDocument(companyId, documentId);
      if (!result.ok) throw new ApiRequestError(result.error);
      return result.value;
    },
    onSuccess: (document) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.documents.list(companyId) });
      void queryClient.invalidateQueries({
        queryKey: queryKeys.documents.detail(companyId, document.id),
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.context(companyId) });
    },
  });
}
