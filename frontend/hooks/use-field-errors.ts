"use client";

import { useState } from "react";
import { ApiRequestError } from "@/lib/api-client";

// Bridges a mutation's ApiRequestError.fields onto the Form's `errors` prop,
// and clears a field the moment the consultant edits it rather than waiting
// for the next submit.
export function useFieldErrors() {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  return {
    fieldErrors,
    clear: (field: string) => {
      setFieldErrors((prev) => {
        if (!(field in prev)) return prev;
        const next = { ...prev };
        delete next[field];
        return next;
      });
    },
    reset: () => setFieldErrors({}),
    setFromError: (error: unknown) => {
      setFieldErrors(error instanceof ApiRequestError ? (error.fields ?? {}) : {});
    },
  };
}
