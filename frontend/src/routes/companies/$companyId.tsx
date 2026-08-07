import { createFileRoute } from "@tanstack/react-router";
import type React from "react";
import { RouteSkeleton } from "@/components/layout/route-skeleton";

export const Route = createFileRoute("/companies/$companyId")({
  component: CompanyRoute,
  pendingComponent: RouteSkeleton,
});

function CompanyRoute(): React.ReactElement {
  const { companyId } = Route.useParams();

  return (
    <div className="p-6">
      <h1 className="font-heading text-lg font-semibold">Company {companyId}</h1>
      <p className="mt-1 text-muted-foreground text-sm">
        Company Context, Discovery Runs and the reasoning graph land here.
      </p>
    </div>
  );
}
