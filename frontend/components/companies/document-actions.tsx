"use client";

import { RotateCcwIcon, Trash2Icon } from "lucide-react";
import type React from "react";
import { useState } from "react";
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { toastManager } from "@/components/ui/toast";
import { useDeleteDocument, useResummariseDocument } from "@/hooks/use-document-mutations";
import type { Document } from "@/lib/api-client";

// Re-summarise reprocesses the stored raw_text — only meaningful once a
// Document has one, i.e. once it is ready. Delete is offered regardless of
// status.
export function DocumentActions({
  companyId,
  doc,
}: {
  companyId: string;
  doc: Document;
}): React.ReactElement {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const deleteDocument = useDeleteDocument(companyId);
  const resummarise = useResummariseDocument(companyId);

  return (
    <div className="flex items-center justify-end gap-1">
      {doc.status === "ready" && (
        <Button
          loading={resummarise.isPending}
          onClick={() => {
            resummarise.mutate(doc.id, {
              onError: (error) => {
                toastManager.add({
                  description: error.message,
                  title: "Re-summarise failed",
                  type: "error",
                });
              },
              onSuccess: () => {
                toastManager.add({ title: "Re-summarised", type: "success" });
              },
            });
          }}
          size="sm"
          variant="ghost"
        >
          <RotateCcwIcon />
          Re-summarise
        </Button>
      )}
      <AlertDialog onOpenChange={setConfirmOpen} open={confirmOpen}>
        <AlertDialogTrigger
          render={<Button aria-label={`Delete ${doc.title}`} size="icon-sm" variant="ghost" />}
        >
          <Trash2Icon />
        </AlertDialogTrigger>
        <AlertDialogPopup>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {doc.title}?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes the Document and its entry in the Company Context. This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose render={<Button variant="outline" />}>Cancel</AlertDialogClose>
            <Button
              loading={deleteDocument.isPending}
              onClick={() => {
                deleteDocument.mutate(doc.id, {
                  onError: (error) => {
                    toastManager.add({
                      description: error.message,
                      title: "Could not delete document",
                      type: "error",
                    });
                  },
                  onSuccess: () => {
                    setConfirmOpen(false);
                    toastManager.add({ title: `${doc.title} deleted`, type: "success" });
                  },
                });
              }}
              variant="destructive"
            >
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogPopup>
      </AlertDialog>
    </div>
  );
}
