"use client";

import { useQuery } from "@tanstack/react-query";
import { api, type HealthStatus } from "@/lib/api-client";
import { queryKeys } from "@/lib/query-keys";

const POLL_INTERVAL_MS = 15_000;

export type ApiHealth =
  | { state: "loading" }
  | { state: "online"; health: HealthStatus }
  | { state: "offline"; error: string };

export function useApiHealth(): ApiHealth {
  const query = useQuery({
    queryKey: queryKeys.health,
    queryFn: async () => {
      const result = await api.health();
      if (!result.ok) throw new Error(result.error);
      return result.value;
    },
    refetchInterval: POLL_INTERVAL_MS,
  });

  if (query.isPending) return { state: "loading" };
  if (query.isError) return { state: "offline", error: query.error.message };
  return { state: "online", health: query.data };
}
