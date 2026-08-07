import { createFileRoute } from "@tanstack/react-router";
import type React from "react";
import { RouteSkeleton } from "@/components/layout/route-skeleton";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";

export const Route = createFileRoute("/companies/")({
  component: CompaniesRoute,
  pendingComponent: RouteSkeleton,
});

function CompaniesRoute(): React.ReactElement {
  return (
    <Empty>
      <EmptyHeader>
        <EmptyTitle>No companies yet</EmptyTitle>
        <EmptyDescription>
          Companies you create will show up here, with their documents and Discovery Runs.
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
