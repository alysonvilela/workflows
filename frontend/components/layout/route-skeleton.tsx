import type React from "react";
import { Skeleton } from "@/components/ui/skeleton";

export function RouteSkeleton(): React.ReactElement {
  return (
    <div className="flex flex-col gap-3 p-6">
      <Skeleton className="h-6 w-48" />
      <Skeleton className="h-4 w-full max-w-md" />
      <Skeleton className="h-4 w-full max-w-sm" />
    </div>
  );
}
