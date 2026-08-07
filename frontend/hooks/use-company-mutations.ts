"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, ApiRequestError, type Company } from "@/lib/api-client";
import { queryKeys } from "@/lib/query-keys";

export function useCreateCompany() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (name: string): Promise<Company> => {
      const result = await api.createCompany(name);
      if (!result.ok) throw new ApiRequestError(result.error);
      return result.value;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.all });
    },
  });
}

export function useRenameCompany() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, name }: { id: string; name: string }): Promise<Company> => {
      const result = await api.renameCompany(id, name);
      if (!result.ok) throw new ApiRequestError(result.error);
      return result.value;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.all });
    },
  });
}

export function useDeleteCompany() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string): Promise<void> => {
      const result = await api.deleteCompany(id);
      if (!result.ok) throw new ApiRequestError(result.error);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.companies.all });
    },
  });
}
