"use client";

import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import { api, type ApiError } from "@/lib/api/client";
import type { LicenseBody, LicenseResponse } from "@/lib/api/contracts";
import { dateValueToInput, inputValueToDateTime } from "@/lib/form-utils";
import { Button } from "@/components/ui/button";
import { FormCard } from "@/components/crud/form-card";
import { Field } from "@/components/crud/form-controls";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Skeleton } from "@/components/ui/skeleton";

type EntityOption = { id: string; name: string };

const licenseTypes = ["Subscription", "Perpetual", "PerUser", "PerDevice", "PerCore", "Concurrent", "Enterprise"] as const;

const licenseSchema = z.object({
  productId: z.string().min(1, "Product is required"),
  vendorId: z.string().min(1, "Vendor is required"),
  type: z.enum(licenseTypes),
  licenseKey: z.string().optional().default(""),
  subscriptionId: z.string().optional().default(""),
  contractNumber: z.string().optional().default(""),
  purchaseOrder: z.string().optional().default(""),
  invoice: z.string().optional().default(""),
  purchaseDate: z.string().optional().default(""),
  renewalDate: z.string().optional().default(""),
  expirationDate: z.string().optional().default(""),
  maintenanceExpiration: z.string().optional().default(""),
  seatCount: z.coerce.number().int().nonnegative(),
  assignedSeats: z.coerce.number().int().nonnegative(),
  cost: z.string().optional().default(""),
  currency: z.string().optional().default(""),
  department: z.string().optional().default(""),
  notes: z.string().optional().default(""),
});

type LicenseFormValues = z.input<typeof licenseSchema>;

function toPayload(values: LicenseFormValues): LicenseBody {
  const trim = (value?: string | null) => (value && value.trim() ? value.trim() : undefined);
  return {
    productId: values.productId,
    vendorId: values.vendorId,
    type: values.type,
    seatCount: Number(values.seatCount),
    assignedSeats: Number(values.assignedSeats),
    licenseKey: trim(values.licenseKey),
    subscriptionId: trim(values.subscriptionId),
    contractNumber: trim(values.contractNumber),
    purchaseOrder: trim(values.purchaseOrder),
    invoice: trim(values.invoice),
    purchaseDate: inputValueToDateTime(values.purchaseDate ?? ""),
    renewalDate: inputValueToDateTime(values.renewalDate ?? ""),
    expirationDate: inputValueToDateTime(values.expirationDate ?? ""),
    maintenanceExpiration: inputValueToDateTime(values.maintenanceExpiration ?? ""),
    cost: trim(values.cost),
    currency: trim(values.currency)?.toUpperCase(),
    department: trim(values.department),
    notes: trim(values.notes),
  };
}

function fromLicense(license?: LicenseResponse | null): LicenseFormValues {
  return {
    productId: license?.productId ?? "",
    vendorId: license?.vendorId ?? "",
    type: (license?.type ?? "Subscription") as LicenseFormValues["type"],
    licenseKey: license?.licenseKey ?? "",
    subscriptionId: license?.subscriptionId ?? "",
    contractNumber: license?.contractNumber ?? "",
    purchaseOrder: license?.purchaseOrder ?? "",
    invoice: license?.invoice ?? "",
    purchaseDate: dateValueToInput(license?.purchaseDate ?? null),
    renewalDate: dateValueToInput(license?.renewalDate ?? null),
    expirationDate: dateValueToInput(license?.expirationDate ?? null),
    maintenanceExpiration: dateValueToInput(license?.maintenanceExpiration ?? null),
    seatCount: license?.seatCount ?? 0,
    assignedSeats: license?.assignedSeats ?? 0,
    cost: license?.cost ?? "",
    currency: license?.currency ?? "USD",
    department: license?.department ?? "",
    notes: license?.notes ?? "",
  };
}

export function LicenseForm({ licenseId }: { licenseId?: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const isEdit = Boolean(licenseId);

  const licenseQuery = useQuery({
    queryKey: ["license", licenseId],
    queryFn: () => api.getLicense(licenseId ?? ""),
    enabled: isEdit,
  });
  const vendorQuery = useQuery({
    queryKey: ["vendors", "options"],
    queryFn: () => api.listVendors({ limit: 500 }),
  });
  const productQuery = useQuery({
    queryKey: ["products", "options"],
    queryFn: () => api.listProducts({ limit: 500 }),
  });

  const defaults = useMemo(() => fromLicense(licenseQuery.data ?? null), [licenseQuery.data]);
  const form = useForm<LicenseFormValues>({ resolver: zodResolver(licenseSchema), defaultValues: defaults });

  useEffect(() => {
    form.reset(defaults);
  }, [defaults, form]);

  const mutation = useMutation({
    mutationFn: async (values: LicenseFormValues) => {
      const payload = toPayload(values);
      return isEdit && licenseId ? api.updateLicense(licenseId, payload) : api.createLicense(payload);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["licenses"] });
      if (licenseId) {
        await queryClient.invalidateQueries({ queryKey: ["license", licenseId] });
      }
      toast.success(isEdit ? "License updated" : "License created");
      router.push("/licenses");
    },
    onError: (error: Error) => {
      const apiError = error as ApiError;
      toast.error(apiError.message);
      form.setError("root", { message: apiError.message });
    },
  });

  const vendorOptions = (vendorQuery.data?.data ?? []) as EntityOption[];
  const productOptions = (productQuery.data?.data ?? []) as EntityOption[];

  if (isEdit && licenseQuery.isError) {
    return <FormCard title="License unavailable" description={licenseQuery.error instanceof Error ? licenseQuery.error.message : "Failed to load the license."}><div className="space-y-3"><p className="text-sm text-destructive">{licenseQuery.error instanceof Error ? licenseQuery.error.message : "Failed to load the license."}</p><Button type="button" variant="outline" onClick={() => router.push("/licenses")}>Back to licenses</Button></div></FormCard>;
  }

  if (isEdit && licenseQuery.isLoading) {
    return <LicenseFormSkeleton />;
  }

  return (
    <FormCard title={isEdit ? "Edit license" : "New license"} description="Create and maintain license records with backend validation.">
      <form className="space-y-6" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Vendor" error={form.formState.errors.vendorId}>
            <select className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm" {...form.register("vendorId")}>
              <option value="">Select a vendor</option>
              {vendorOptions.map((vendor) => <option key={vendor.id} value={vendor.id}>{vendor.name}</option>)}
            </select>
          </Field>
          <Field label="Product" error={form.formState.errors.productId}>
            <select className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm" {...form.register("productId")}>
              <option value="">Select a product</option>
              {productOptions.map((product) => <option key={product.id} value={product.id}>{product.name}</option>)}
            </select>
          </Field>
          <Field label="Type" error={form.formState.errors.type}>
            <select className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm" {...form.register("type")}>
              {licenseTypes.map((type) => <option key={type} value={type}>{type}</option>)}
            </select>
          </Field>
          <Field label="License key" error={form.formState.errors.licenseKey}><Input {...form.register("licenseKey")} placeholder="AAAA-BBBB-CCCC" /></Field>
          <Field label="Subscription ID" error={form.formState.errors.subscriptionId}><Input {...form.register("subscriptionId")} /></Field>
          <Field label="Contract number" error={form.formState.errors.contractNumber}><Input {...form.register("contractNumber")} /></Field>
          <Field label="Purchase order" error={form.formState.errors.purchaseOrder}><Input {...form.register("purchaseOrder")} /></Field>
          <Field label="Invoice" error={form.formState.errors.invoice}><Input {...form.register("invoice")} /></Field>
          <Field label="Purchase date" error={form.formState.errors.purchaseDate}><Input type="date" {...form.register("purchaseDate")} /></Field>
          <Field label="Renewal date" error={form.formState.errors.renewalDate}><Input type="date" {...form.register("renewalDate")} /></Field>
          <Field label="Expiration date" error={form.formState.errors.expirationDate}><Input type="date" {...form.register("expirationDate")} /></Field>
          <Field label="Maintenance expiration" error={form.formState.errors.maintenanceExpiration}><Input type="date" {...form.register("maintenanceExpiration")} /></Field>
          <Field label="Seat count" error={form.formState.errors.seatCount}><Input type="number" min={0} {...form.register("seatCount", { valueAsNumber: true })} /></Field>
          <Field label="Assigned seats" error={form.formState.errors.assignedSeats}><Input type="number" min={0} {...form.register("assignedSeats", { valueAsNumber: true })} /></Field>
          <Field label="Cost" error={form.formState.errors.cost}><Input inputMode="decimal" placeholder="1234.56" {...form.register("cost")} /></Field>
          <Field label="Currency" error={form.formState.errors.currency}><Input maxLength={3} {...form.register("currency")} placeholder="USD" /></Field>
          <Field label="Department" error={form.formState.errors.department}><Input {...form.register("department")} /></Field>
        </div>
        <Field label="Notes" error={form.formState.errors.notes}><Textarea rows={4} {...form.register("notes")} /></Field>
        {form.formState.errors.root?.message ? <p className="text-sm text-destructive">{form.formState.errors.root.message}</p> : null}
        <div className="flex gap-2">
          <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "Saving…" : "Save license"}</Button>
          <Button type="button" variant="outline" onClick={() => router.push("/licenses")}>Cancel</Button>
        </div>
      </form>
    </FormCard>
  );
}

function LicenseFormSkeleton() {
  return (
    <FormCard title="Loading license" description="Fetching record details…">
      <div className="grid gap-4 md:grid-cols-2">
        {Array.from({ length: 10 }).map((_, index) => <Skeleton key={index} className="h-16 w-full" />)}
      </div>
    </FormCard>
  );
}
