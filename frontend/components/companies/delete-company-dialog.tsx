"use client";

import { Trash2Icon } from "lucide-react";
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
import { useDeleteCompany } from "@/hooks/use-company-mutations";
import type { Company } from "@/lib/api-client";

export function DeleteCompanyDialog({
  company,
  onDeleted,
}: {
  company: Company;
  onDeleted?: () => void;
}): React.ReactElement {
  const [open, setOpen] = useState(false);
  const deleteCompany = useDeleteCompany();

  return (
    <AlertDialog onOpenChange={setOpen} open={open}>
      <AlertDialogTrigger
        render={<Button aria-label={`Delete ${company.name}`} size="icon-sm" variant="ghost" />}
      >
        <Trash2Icon />
      </AlertDialogTrigger>
      <AlertDialogPopup>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {company.name}?</AlertDialogTitle>
          <AlertDialogDescription>
            This removes the Company and every Document, Discovery Run and Node under it. This
            cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogClose render={<Button variant="outline" />}>Cancel</AlertDialogClose>
          <Button
            loading={deleteCompany.isPending}
            onClick={() => {
              deleteCompany.mutate(company.id, {
                onError: (error) => {
                  toastManager.add({
                    description: error.message,
                    title: "Could not delete company",
                    type: "error",
                  });
                },
                onSuccess: () => {
                  setOpen(false);
                  toastManager.add({ title: `${company.name} deleted`, type: "success" });
                  onDeleted?.();
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
  );
}
