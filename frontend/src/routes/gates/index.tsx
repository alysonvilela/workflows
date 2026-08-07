import { createFileRoute } from "@tanstack/react-router";
import type React from "react";
import { RouteSkeleton } from "@/components/layout/route-skeleton";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";

export const Route = createFileRoute("/gates/")({
  component: GatesRoute,
  pendingComponent: RouteSkeleton,
});

function GatesRoute(): React.ReactElement {
  return (
    <Empty>
      <EmptyHeader>
        <EmptyTitle>No open Gates</EmptyTitle>
        <EmptyDescription>
          Every Discovery Run waiting on you, across every Company, will show up here.
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
