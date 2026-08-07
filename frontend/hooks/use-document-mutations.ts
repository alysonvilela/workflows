"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ApiRequestError, type UploadFileResult } from "@/lib/api-client";
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
    },
  });
}
