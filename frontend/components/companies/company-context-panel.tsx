"use client";

import type React from "react";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Spinner } from "@/components/ui/spinner";
import { useCompanyContext } from "@/hooks/use-companies";

// Shows exactly the text a Discovery Run's prompts read (ADR-0001) — one
// entry per ready Document, concatenated by the backend, not reassembled
// here.
export function CompanyContextPanel({ companyId }: { companyId: string }): React.ReactElement {
  const { data: text, isPending } = useCompanyContext(companyId);

  if (isPending) {
    return (
      <div className="flex items-center justify-center py-6">
        <Spinner className="size-4 text-muted-foreground" />
      </div>
    );
  }

  if (!text) {
    return (
      <Empty className="py-6">
        <EmptyHeader>
          <EmptyTitle>No Company Context yet</EmptyTitle>
          <EmptyDescription>It fills in once a Document reaches ready.</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-lg bg-muted/40 px-3 py-2 text-[12px] text-muted-foreground">
      {text}
    </pre>
  );
}
