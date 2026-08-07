"use client";

import { AlertTriangleIcon, CheckCircle2Icon, FileTextIcon, UploadIcon } from "lucide-react";
import type React from "react";
import { useRef, useState } from "react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Spinner } from "@/components/ui/spinner";
import { toastManager } from "@/components/ui/toast";
import { useUploadDocuments } from "@/hooks/use-document-mutations";
import { useDocument, useDocuments } from "@/hooks/use-documents";
import type { Document, DocumentStatus } from "@/lib/api-client";
import { cn } from "@/lib/utils";

export function DocumentsPanel({ companyId }: { companyId: string }): React.ReactElement {
  const { data, isPending } = useDocuments(companyId);

  return (
    <div className="flex flex-col gap-3">
      <UploadDropzone companyId={companyId} />

      {isPending ? null : !data || data.documents.length === 0 ? (
        <Empty className="py-6">
          <EmptyHeader>
            <EmptyTitle>No documents yet</EmptyTitle>
            <EmptyDescription>
              Drop a Markdown file above to start building the Company Context.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <DocumentList companyId={companyId} documents={data.documents} />
      )}
    </div>
  );
}

function UploadDropzone({ companyId }: { companyId: string }): React.ReactElement {
  const uploadDocuments = useUploadDocuments(companyId);
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragActive, setDragActive] = useState(false);

  function upload(files: FileList | null) {
    if (!files || files.length === 0) return;
    uploadDocuments.mutate([...files], {
      onSuccess: (results) => {
        const failed = results.filter((r) => r.error !== null);
        if (failed.length > 0) {
          toastManager.add({
            description: failed.map((r) => `${r.filename}: ${r.error}`).join("; "),
            title: failed.length === results.length ? "Upload failed" : "Some files failed",
            type: "error",
          });
        }
      },
      onError: (error) => {
        toastManager.add({ description: error.message, title: "Upload failed", type: "error" });
      },
    });
  }

  return (
    <button
      className={cn(
        "flex w-full flex-col items-center gap-1 rounded-2xl border-2 border-dashed bg-muted/30 py-6 transition-colors hover:border-primary/50 hover:bg-muted/50",
        dragActive && "border-primary bg-primary/5",
      )}
      onClick={() => inputRef.current?.click()}
      onDragLeave={() => setDragActive(false)}
      onDragOver={(e) => {
        e.preventDefault();
        setDragActive(true);
      }}
      onDrop={(e) => {
        e.preventDefault();
        setDragActive(false);
        upload(e.dataTransfer.files);
      }}
      type="button"
    >
      {uploadDocuments.isPending ? (
        <Spinner className="size-5 text-muted-foreground" />
      ) : (
        <UploadIcon className="size-5 text-muted-foreground" />
      )}
      <span className="font-medium text-sm">Drop Markdown files, or click to browse</span>
      <span className="text-[11px] text-muted-foreground">one or several at a time</span>
      <input
        accept=".md,.markdown"
        className="sr-only"
        multiple
        onChange={(e) => {
          upload(e.target.files);
          e.target.value = "";
        }}
        ref={inputRef}
        type="file"
      />
    </button>
  );
}

function DocumentList({
  companyId,
  documents,
}: {
  companyId: string;
  documents: Document[];
}): React.ReactElement {
  const [expanded, setExpanded] = useState<string[]>([]);

  return (
    <Accordion className="w-full" onValueChange={setExpanded} value={expanded}>
      {documents.map((doc) => (
        <AccordionItem key={doc.id} value={doc.id}>
          <AccordionTrigger className="gap-2">
            <span className="flex min-w-0 flex-1 items-center gap-2">
              <FileTextIcon className="size-3.5 shrink-0 text-muted-foreground" />
              <span className="truncate text-[13px] text-foreground">{doc.title}</span>
              <StatusBadge status={doc.status} />
            </span>
          </AccordionTrigger>
          <AccordionContent>
            <DocumentDetail companyId={companyId} doc={doc} open={expanded.includes(doc.id)} />
          </AccordionContent>
        </AccordionItem>
      ))}
    </Accordion>
  );
}

function DocumentDetail({
  companyId,
  doc,
  open,
}: {
  companyId: string;
  doc: Document;
  open: boolean;
}): React.ReactElement | null {
  const { data } = useDocument(companyId, doc.id, open && doc.status === "ready");

  if (doc.status === "failed") {
    return doc.error ? (
      <p className="rounded-lg bg-destructive/8 px-2.5 py-1.5 text-destructive-foreground text-xs">
        {doc.error}
      </p>
    ) : null;
  }

  if (doc.status === "parsing") {
    return <p className="text-muted-foreground text-xs">Parsing…</p>;
  }

  return (
    <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-lg bg-muted/40 px-2.5 py-1.5 text-[11px] text-muted-foreground">
      {data?.rawText ?? "Loading…"}
    </pre>
  );
}

function StatusBadge({ status }: { status: DocumentStatus }): React.ReactElement {
  if (status === "ready") {
    return (
      <Badge size="sm" variant="success">
        <CheckCircle2Icon />
        ready
      </Badge>
    );
  }
  if (status === "failed") {
    return (
      <Badge size="sm" variant="error">
        <AlertTriangleIcon />
        failed
      </Badge>
    );
  }
  return (
    <Badge size="sm" variant="warning">
      <Spinner />
      parsing
    </Badge>
  );
}
