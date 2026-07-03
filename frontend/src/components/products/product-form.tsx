"use client";

import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import { api, type ApiError } from "@/lib/api/client";
import type { ProductBody, ProductResponse } from "@/lib/api/contracts";
import { Button } from "@/components/ui/button";
import { FormCard } from "@/components/crud/form-card";
import { Field } from "@/components/crud/form-controls";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { splitCommaList, joinCommaList } from "@/lib/form-utils";

type EntityOption = { id: string; name: string };

const productSchema = z.object({
  name: z.string().min(1, "Product name is required"),
  vendorId: z.string().min(1, "Vendor is required"),
  category: z.string().optional().default(""),
  version: z.string().optional().default(""),
  website: z.string().optional().default(""),
  description: z.string().optional().default(""),
  tags: z.string().optional().default(""),
});

type ProductFormValues = z.input<typeof productSchema>;

function toPayload(values: ProductFormValues): ProductBody {
  const trim = (value?: string | null) => (value && value.trim() ? value.trim() : undefined);
  return {
    name: values.name.trim(),
    vendorId: values.vendorId,
    category: trim(values.category),
    version: trim(values.version),
    website: trim(values.website),
    description: trim(values.description),
    tags: splitCommaList(values.tags ?? ""),
  };
}

function fromProduct(product?: ProductResponse | null): ProductFormValues {
  return {
    name: product?.name ?? "",
    vendorId: product?.vendorId ?? "",
    category: product?.category ?? "",
    version: product?.version ?? "",
    website: product?.website ?? "",
    description: product?.description ?? "",
    tags: joinCommaList(product?.tags ?? null),
  };
}

export function ProductForm({ productId }: { productId?: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const isEdit = Boolean(productId);

  const productQuery = useQuery({
    queryKey: ["product", productId],
    queryFn: () => api.getProduct(productId ?? ""),
    enabled: isEdit,
  });
  const vendorQuery = useQuery({
    queryKey: ["vendors", "options"],
    queryFn: () => api.listVendors({ limit: 500 }),
  });

  const defaults = useMemo(() => fromProduct(productQuery.data ?? null), [productQuery.data]);
  const form = useForm<ProductFormValues>({ resolver: zodResolver(productSchema), defaultValues: defaults });

  useEffect(() => {
    form.reset(defaults);
  }, [defaults, form]);

  const mutation = useMutation({
    mutationFn: async (values: ProductFormValues) => {
      const payload = toPayload(values);
      return isEdit && productId ? api.updateProduct(productId, payload) : api.createProduct(payload);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["products"] });
      if (productId) {
        await queryClient.invalidateQueries({ queryKey: ["product", productId] });
      }
      toast.success(isEdit ? "Product updated" : "Product created");
      router.push("/products");
    },
    onError: (error: Error) => {
      const apiError = error as ApiError;
      toast.error(apiError.message);
      form.setError("root", { message: apiError.message });
    },
  });

  if (isEdit && productQuery.isError) {
    return <FormCard title="Product unavailable" description={productQuery.error instanceof Error ? productQuery.error.message : "Failed to load the product."}><div className="space-y-3"><p className="text-sm text-destructive">{productQuery.error instanceof Error ? productQuery.error.message : "Failed to load the product."}</p><Button type="button" variant="outline" onClick={() => router.push("/products")}>Back to products</Button></div></FormCard>;
  }

  if (isEdit && productQuery.isLoading) {
    return <ProductFormSkeleton />;
  }

  const vendorOptions = (vendorQuery.data?.data ?? []) as EntityOption[];

  return (
    <FormCard title={isEdit ? "Edit product" : "New product"} description="Maintain products and their metadata.">
      <form className="space-y-6" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Name" error={form.formState.errors.name}><Input {...form.register("name")} /></Field>
          <Field label="Vendor" error={form.formState.errors.vendorId}>
            <select className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm" {...form.register("vendorId")}>
              <option value="">Select a vendor</option>
              {vendorOptions.map((vendor) => <option key={vendor.id} value={vendor.id}>{vendor.name}</option>)}
            </select>
          </Field>
          <Field label="Category" error={form.formState.errors.category}><Input {...form.register("category")} /></Field>
          <Field label="Version" error={form.formState.errors.version}><Input {...form.register("version")} /></Field>
          <Field label="Website" error={form.formState.errors.website}><Input {...form.register("website")} placeholder="https://example.com" /></Field>
        </div>
        <Field label="Description" error={form.formState.errors.description}><Textarea rows={4} {...form.register("description")} /></Field>
        <Field label="Tags" error={form.formState.errors.tags} hint="Comma-separated tags">
          <Input {...form.register("tags")} placeholder="saas, security" />
        </Field>
        {form.formState.errors.root?.message ? <p className="text-sm text-destructive">{form.formState.errors.root.message}</p> : null}
        <div className="flex gap-2">
          <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "Saving…" : "Save product"}</Button>
          <Button type="button" variant="outline" onClick={() => router.push("/products")}>Cancel</Button>
        </div>
      </form>
    </FormCard>
  );
}

function ProductFormSkeleton() {
  return (
    <FormCard title="Loading product" description="Fetching record details…">
      <div className="grid gap-4 md:grid-cols-2">
        {Array.from({ length: 5 }).map((_, index) => <Skeleton key={index} className="h-16 w-full" />)}
      </div>
    </FormCard>
  );
}
