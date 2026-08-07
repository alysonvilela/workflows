import { createFileRoute, Link } from "@tanstack/react-router";
import { ChevronLeftIcon, ChevronRightIcon, PlusIcon } from "lucide-react";
import type React from "react";
import { useState } from "react";
import { DeleteCompanyDialog } from "@/components/companies/delete-company-dialog";
import { RouteSkeleton } from "@/components/layout/route-skeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Form } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
} from "@/components/ui/pagination";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCompanies } from "@/hooks/use-companies";
import { useCreateCompany } from "@/hooks/use-company-mutations";
import { useFieldErrors } from "@/hooks/use-field-errors";
import type { Company } from "@/lib/api-client";

const PER_PAGE = 20;

export const Route = createFileRoute("/companies/")({
  component: CompaniesRoute,
  pendingComponent: RouteSkeleton,
});

function CompaniesRoute(): React.ReactElement {
  const [page, setPage] = useState(1);
  const { data, isPending } = useCompanies(page, PER_PAGE);

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-6">
      <CreateCompanyForm />

      {isPending ? (
        <RouteSkeleton />
      ) : !data || data.companies.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>No companies yet</EmptyTitle>
            <EmptyDescription>
              Create one above — everything you learn about a client accumulates there across
              months.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <>
          <CompaniesTable companies={data.companies} />
          {data.pagination.totalPages > 1 && (
            <PageControls
              onPageChange={setPage}
              page={data.pagination.page}
              totalPages={data.pagination.totalPages}
            />
          )}
        </>
      )}
    </div>
  );
}

function CreateCompanyForm(): React.ReactElement {
  const [name, setName] = useState("");
  const { clear, fieldErrors, reset, setFromError } = useFieldErrors();
  const createCompany = useCreateCompany();

  return (
    <Card>
      <CardHeader>
        <CardTitle>New company</CardTitle>
        <CardDescription>
          Everything you learn about one client accumulates here across months.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form
          className="flex items-start gap-3"
          errors={fieldErrors}
          onFormSubmit={() => {
            createCompany.mutate(name, {
              onError: setFromError,
              onSuccess: () => {
                setName("");
                reset();
              },
            });
          }}
        >
          <Field className="flex-1" name="name">
            <FieldLabel className="sr-only">Company name</FieldLabel>
            <Input
              name="name"
              onValueChange={(value) => {
                setName(value);
                clear("name");
              }}
              placeholder="e.g. Aurora Logistics"
              value={name}
            />
            <FieldError match />
          </Field>
          <Button loading={createCompany.isPending} type="submit">
            <PlusIcon />
            Create
          </Button>
        </Form>
      </CardContent>
    </Card>
  );
}

function CompaniesTable({ companies }: { companies: Company[] }): React.ReactElement {
  return (
    <Table variant="card">
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Slug</TableHead>
          <TableHead>Documents</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {companies.map((company) => (
          <TableRow key={company.id}>
            <TableCell>
              <Link
                className="font-medium hover:underline"
                params={{ companyId: company.id }}
                to="/companies/$companyId"
              >
                {company.name}
              </Link>
            </TableCell>
            <TableCell className="font-mono text-muted-foreground text-xs">
              {company.slug}
            </TableCell>
            <TableCell>
              <Badge variant="secondary">{company.documentCount}</Badge>
            </TableCell>
            <TableCell className="text-muted-foreground text-sm">
              {new Date(company.createdAt).toLocaleDateString()}
            </TableCell>
            <TableCell className="text-right">
              <DeleteCompanyDialog company={company} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function PageControls({
  onPageChange,
  page,
  totalPages,
}: {
  onPageChange: (page: number) => void;
  page: number;
  totalPages: number;
}): React.ReactElement {
  return (
    <Pagination className="items-center justify-end gap-3">
      <span className="text-muted-foreground text-sm">
        Page {page} of {totalPages}
      </span>
      <PaginationContent>
        <PaginationItem>
          <PaginationLink
            aria-label="Go to previous page"
            render={
              <Button
                disabled={page <= 1}
                onClick={() => onPageChange(page - 1)}
                size="sm"
                type="button"
                variant="outline"
              />
            }
          >
            <ChevronLeftIcon />
            Previous
          </PaginationLink>
        </PaginationItem>
        <PaginationItem>
          <PaginationLink
            aria-label="Go to next page"
            render={
              <Button
                disabled={page >= totalPages}
                onClick={() => onPageChange(page + 1)}
                size="sm"
                type="button"
                variant="outline"
              />
            }
          >
            Next
            <ChevronRightIcon />
          </PaginationLink>
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  );
}
