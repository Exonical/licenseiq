"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { AlertCircle, ArrowLeft, ArrowRight, PenSquare, Plus, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { api, type ApiError } from "@/lib/api/client";
import type { LicenseResponse } from "@/lib/api/contracts";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PageFrame, ListCard } from "@/components/crud/page-frame";

const columnHelper = createColumnHelper<LicenseResponse>();
const pageSize = 50;

type EntityOption = { id: string; name: string };

function formatDate(value?: string | null) {
  if (!value) return "—";
  return new Date(value).toLocaleDateString();
}

export function LicenseList() {
  const [search, setSearch] = useState("");
  const [pageIndex, setPageIndex] = useState(0);
  const queryClient = useQueryClient();

  const licensesQuery = useQuery({
    queryKey: ["licenses", pageIndex, pageSize],
    queryFn: () => api.listLicenses({ limit: pageSize, offset: pageIndex * pageSize }),
    placeholderData: (previous) => previous,
  });
  const vendorQuery = useQuery({ queryKey: ["vendors", "options"], queryFn: () => api.listVendors({ limit: 500 }) });
  const productQuery = useQuery({ queryKey: ["products", "options"], queryFn: () => api.listProducts({ limit: 500 }) });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteLicense(id),
    onSuccess: async () => {
      toast.success("License deleted");
      await queryClient.invalidateQueries({ queryKey: ["licenses"] });
    },
    onError: (error: Error) => toast.error((error as ApiError).message),
  });

  const vendorOptions = useMemo(() => (vendorQuery.data?.data ?? []) as EntityOption[], [vendorQuery.data]);
  const productOptions = useMemo(() => (productQuery.data?.data ?? []) as EntityOption[], [productQuery.data]);
  const licenseRows = useMemo(() => (licensesQuery.data?.data ?? []) as LicenseResponse[], [licensesQuery.data]);
  const vendorById = useMemo(() => new Map(vendorOptions.map((item) => [item.id, item] as const)), [vendorOptions]);
  const productById = useMemo(() => new Map(productOptions.map((item) => [item.id, item] as const)), [productOptions]);

  const rows = useMemo(() => {
    const normalized = search.trim().toLowerCase();
    return licenseRows.filter((license) => {
      if (!normalized) return true;
      const vendor = vendorById.get(license.vendorId)?.name ?? license.vendorId;
      const product = productById.get(license.productId)?.name ?? license.productId;
      return [license.licenseKey, vendor, product, license.department, license.type].some((value) => String(value ?? "").toLowerCase().includes(normalized));
    });
  }, [licenseRows, search, productById, vendorById]);

  const columns = useMemo(() => [
    columnHelper.accessor("licenseKey", { header: "License", cell: (info) => info.getValue() || "—" }),
    columnHelper.display({ id: "vendor", header: "Vendor", cell: ({ row }) => vendorById.get(row.original.vendorId)?.name ?? row.original.vendorId }),
    columnHelper.display({ id: "product", header: "Product", cell: ({ row }) => productById.get(row.original.productId)?.name ?? row.original.productId }),
    columnHelper.accessor("type", { header: "Type" }),
    columnHelper.display({ id: "seats", header: "Seats", cell: ({ row }) => `${row.original.assignedSeats}/${row.original.seatCount}` }),
    columnHelper.display({ id: "renewal", header: "Renewal", cell: ({ row }) => formatDate(row.original.renewalDate) }),
    columnHelper.display({ id: "actions", header: "Actions", cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <Button asChild variant="ghost" size="sm"><Link href={`/licenses/${row.original.id}`}><PenSquare className="h-4 w-4" />Edit</Link></Button>
        <Button variant="ghost" size="sm" onClick={() => {
          if (window.confirm(`Delete license ${row.original.licenseKey || row.original.id}?`)) {
            deleteMutation.mutate(row.original.id);
          }
        }}><Trash2 className="h-4 w-4" />Delete</Button>
      </div>
    ) }),
  ], [deleteMutation, productById, vendorById]);

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({ data: rows, columns, getCoreRowModel: getCoreRowModel() });
  const loading = licensesQuery.isLoading;
  const error = licensesQuery.error instanceof Error ? licensesQuery.error.message : null;
  const hasMore = (licensesQuery.data?.data ?? []).length === pageSize;

  return (
    <PageFrame
      title="Licenses"
      description="Browse, create, edit, and remove license records."
      actions={<Button asChild><Link href="/licenses/new"><Plus className="h-4 w-4" />New license</Link></Button>}
    >
      <ListCard title="License records" description="Search within the current page of results.">
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative w-full sm:max-w-sm">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search licenses" className="pl-9" />
          </div>
          <div className="text-sm text-muted-foreground">{loading ? "Loading…" : `${rows.length} shown`}</div>
        </div>
        {error ? <div className="mb-4 flex items-center gap-2 rounded-md border border-dashed p-3 text-sm text-destructive"><AlertCircle className="h-4 w-4" />{error}</div> : null}
        {loading ? <LicenseTableSkeleton /> : <div className="rounded-md border"><Table><TableHeader>{table.getHeaderGroups().map((headerGroup) => <TableRow key={headerGroup.id}>{headerGroup.headers.map((header) => <TableHead key={header.id}>{header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}</TableHead>)}</TableRow>)}</TableHeader><TableBody>{table.getRowModel().rows.length ? table.getRowModel().rows.map((row) => <TableRow key={row.id}>{row.getVisibleCells().map((cell) => <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>)}</TableRow>) : <TableRow><TableCell colSpan={columns.length} className="py-10 text-center text-muted-foreground">No licenses found.</TableCell></TableRow>}</TableBody></Table></div>}
        <div className="mt-4 flex items-center justify-between">
          <Button variant="outline" disabled={pageIndex === 0} onClick={() => setPageIndex((current) => Math.max(0, current - 1))}><ArrowLeft className="h-4 w-4" />Previous</Button>
          <div className="text-sm text-muted-foreground">Page {pageIndex + 1}{hasMore ? "" : " · end"}</div>
          <Button variant="outline" disabled={!hasMore} onClick={() => setPageIndex((current) => current + 1)}>Next<ArrowRight className="h-4 w-4" /></Button>
        </div>
      </ListCard>
    </PageFrame>
  );
}

function LicenseTableSkeleton() {
  return <div className="space-y-2 rounded-md border p-4">{Array.from({ length: 6 }).map((_, index) => <Skeleton key={index} className="h-10 w-full" />)}</div>;
}
