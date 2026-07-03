"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createColumnHelper, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { AlertCircle, ArrowLeft, ArrowRight, PenSquare, Plus, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { api, type ApiError } from "@/lib/api/client";
import type { ProductResponse } from "@/lib/api/contracts";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PageFrame, ListCard } from "@/components/crud/page-frame";

const columnHelper = createColumnHelper<ProductResponse>();
const pageSize = 50;

type EntityOption = { id: string; name: string };

export function ProductList() {
  const [search, setSearch] = useState("");
  const [pageIndex, setPageIndex] = useState(0);
  const queryClient = useQueryClient();

  const productsQuery = useQuery({
    queryKey: ["products", pageIndex, pageSize],
    queryFn: () => api.listProducts({ limit: pageSize, offset: pageIndex * pageSize }),
    placeholderData: (previous) => previous,
  });
  const vendorsQuery = useQuery({ queryKey: ["vendors", "options"], queryFn: () => api.listVendors({ limit: 500 }) });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteProduct(id),
    onSuccess: async () => {
      toast.success("Product deleted");
      await queryClient.invalidateQueries({ queryKey: ["products"] });
    },
    onError: (error: Error) => toast.error((error as ApiError).message),
  });

  const vendorOptions = useMemo(() => (vendorsQuery.data?.data ?? []) as EntityOption[], [vendorsQuery.data]);
  const vendorById = useMemo(() => new Map(vendorOptions.map((vendor) => [vendor.id, vendor] as const)), [vendorOptions]);
  const productRows = useMemo(() => (productsQuery.data?.data ?? []) as ProductResponse[], [productsQuery.data]);

  const rows = useMemo(() => {
    const normalized = search.trim().toLowerCase();
    return productRows.filter((product) => [product.name, product.category, product.version, product.website, product.description, vendorById.get(product.vendorId)?.name].some((value) => String(value ?? "").toLowerCase().includes(normalized)));
  }, [productRows, search, vendorById]);

  const columns = useMemo(() => [
    columnHelper.accessor("name", { header: "Name" }),
    columnHelper.display({ id: "vendor", header: "Vendor", cell: ({ row }) => vendorById.get(row.original.vendorId)?.name ?? row.original.vendorId }),
    columnHelper.accessor("category", { header: "Category", cell: (info) => info.getValue() || "—" }),
    columnHelper.accessor("version", { header: "Version", cell: (info) => info.getValue() || "—" }),
    columnHelper.display({ id: "tags", header: "Tags", cell: ({ row }) => row.original.tags?.length ? row.original.tags.join(", ") : "—" }),
    columnHelper.display({ id: "actions", header: "Actions", cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <Button asChild variant="ghost" size="sm"><Link href={`/products/${row.original.id}`}><PenSquare className="h-4 w-4" />Edit</Link></Button>
        <Button variant="ghost" size="sm" onClick={() => {
          if (window.confirm(`Delete product ${row.original.name}?`)) {
            deleteMutation.mutate(row.original.id);
          }
        }}><Trash2 className="h-4 w-4" />Delete</Button>
      </div>
    ) }),
  ], [deleteMutation, vendorById]);

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({ data: rows, columns, getCoreRowModel: getCoreRowModel() });
  const loading = productsQuery.isLoading || vendorsQuery.isLoading;
  const error = productsQuery.error instanceof Error ? productsQuery.error.message : null;
  const hasMore = (productsQuery.data?.data ?? []).length === pageSize;

  return (
    <PageFrame title="Products" description="Manage products, versions, and tags." actions={<Button asChild><Link href="/products/new"><Plus className="h-4 w-4" />New product</Link></Button>}>
      <ListCard title="Product records" description="Search within the current page of results.">
        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="relative w-full sm:max-w-sm">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search products" className="pl-9" />
          </div>
          <div className="text-sm text-muted-foreground">{loading ? "Loading…" : `${rows.length} shown`}</div>
        </div>
        {error ? <div className="mb-4 flex items-center gap-2 rounded-md border border-dashed p-3 text-sm text-destructive"><AlertCircle className="h-4 w-4" />{error}</div> : null}
        {loading ? <ProductTableSkeleton /> : <div className="rounded-md border"><Table><TableHeader>{table.getHeaderGroups().map((headerGroup) => <TableRow key={headerGroup.id}>{headerGroup.headers.map((header) => <TableHead key={header.id}>{header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}</TableHead>)}</TableRow>)}</TableHeader><TableBody>{table.getRowModel().rows.length ? table.getRowModel().rows.map((row) => <TableRow key={row.id}>{row.getVisibleCells().map((cell) => <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>)}</TableRow>) : <TableRow><TableCell colSpan={columns.length} className="py-10 text-center text-muted-foreground">No products found.</TableCell></TableRow>}</TableBody></Table></div>}
        <div className="mt-4 flex items-center justify-between">
          <Button variant="outline" disabled={pageIndex === 0} onClick={() => setPageIndex((current) => Math.max(0, current - 1))}><ArrowLeft className="h-4 w-4" />Previous</Button>
          <div className="text-sm text-muted-foreground">Page {pageIndex + 1}{hasMore ? "" : " · end"}</div>
          <Button variant="outline" disabled={!hasMore} onClick={() => setPageIndex((current) => current + 1)}>Next<ArrowRight className="h-4 w-4" /></Button>
        </div>
      </ListCard>
    </PageFrame>
  );
}

function ProductTableSkeleton() {
  return <div className="space-y-2 rounded-md border p-4">{Array.from({ length: 6 }).map((_, index) => <Skeleton key={index} className="h-10 w-full" />)}</div>;
}
