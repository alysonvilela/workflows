import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeftIcon } from "lucide-react";
import type React from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { DeleteCompanyDialog } from "@/components/companies/delete-company-dialog";
import { RouteSkeleton } from "@/components/layout/route-skeleton";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { useCompany } from "@/hooks/use-companies";
import { useRenameCompany } from "@/hooks/use-company-mutations";
import { ApiRequestError, type Company } from "@/lib/api-client";

export const Route = createFileRoute("/companies/$companyId")({
  component: CompanyRoute,
  pendingComponent: RouteSkeleton,
});

function CompanyRoute(): React.ReactElement {
  const { companyId } = Route.useParams();
  const navigate = useNavigate();
  const { data: company, isError, isPending } = useCompany(companyId);

  if (isPending) return <RouteSkeleton />;

  if (isError || !company) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>Company not found</EmptyTitle>
          <EmptyDescription>It may have been deleted.</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-4 p-6">
      <Link
        className="flex w-fit items-center gap-1 text-muted-foreground text-sm hover:text-foreground hover:underline"
        to="/companies"
      >
        <ArrowLeftIcon className="size-3.5" />
        Companies
      </Link>

      <Card>
        <CardHeader>
          <CardTitle>{company.name}</CardTitle>
          <CardDescription>
            Company Context, Documents and Discovery Runs land here.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <RenameCompanyForm company={company} />
          <div className="flex items-center gap-2 text-muted-foreground text-sm">
            <span>
              {company.documentCount} document{company.documentCount === 1 ? "" : "s"}
            </span>
            <Separator className="h-4" orientation="vertical" />
            <span>
              slug: <span className="font-mono">{company.slug}</span>
            </span>
          </div>
        </CardContent>
        <CardFooter className="justify-end border-t">
          <DeleteCompanyDialog company={company} onDeleted={() => navigate({ to: "/companies" })} />
        </CardFooter>
      </Card>
    </div>
  );
}

interface RenameCompanyValues {
  name: string;
}

function RenameCompanyForm({ company }: { company: Company }): React.ReactElement {
  const renameCompany = useRenameCompany();
  const {
    control,
    clearErrors,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<RenameCompanyValues>({ values: { name: company.name } });

  const nameValue = useWatch({ control, name: "name" });
  const trimmed = nameValue.trim();
  const unchanged = trimmed === company.name;

  const onSubmit = handleSubmit((values) => {
    const trimmedName = values.name.trim();
    if (trimmedName === company.name) return;
    renameCompany.mutate(
      { id: company.id, name: trimmedName },
      {
        onError: (error) => {
          if (!(error instanceof ApiRequestError)) return;
          for (const [field, message] of Object.entries(error.fields ?? {})) {
            setError(field as keyof RenameCompanyValues, { message, type: "server" });
          }
        },
      },
    );
  });

  return (
    <form className="flex items-end gap-3" onSubmit={onSubmit}>
      <Field className="flex-1" invalid={Boolean(errors.name)} name="name">
        <FieldLabel>Name</FieldLabel>
        <Controller
          control={control}
          name="name"
          render={({ field }) => (
            <Input
              name={field.name}
              onValueChange={(value) => {
                field.onChange(value);
                clearErrors("name");
              }}
              value={field.value}
            />
          )}
        />
        {errors.name && <FieldError match>{errors.name.message}</FieldError>}
      </Field>
      <Button
        disabled={unchanged || trimmed === ""}
        loading={renameCompany.isPending}
        type="submit"
        variant="outline"
      >
        Save
      </Button>
    </form>
  );
}
