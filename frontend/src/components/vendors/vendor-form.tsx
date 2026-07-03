"use client";

import { useEffect, useMemo } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import { api, type ApiError } from "@/lib/api/client";
import type { VendorBody, VendorResponse } from "@/lib/api/contracts";
import { Button } from "@/components/ui/button";
import { FormCard } from "@/components/crud/form-card";
import { Field } from "@/components/crud/form-controls";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";

const contactSchema = z.object({
  name: z.string().min(1, "Contact name is required"),
  email: z.string().optional().default(""),
  phone: z.string().optional().default(""),
  role: z.string().optional().default(""),
});

const vendorSchema = z.object({
  name: z.string().min(1, "Vendor name is required"),
  supportUrl: z.string().optional().default(""),
  accountManager: z.string().optional().default(""),
  notes: z.string().optional().default(""),
  contacts: z.array(contactSchema).default([]),
});

type VendorFormValues = z.input<typeof vendorSchema>;

function toPayload(values: VendorFormValues): VendorBody {
  const trim = (value?: string | null) => (value && value.trim() ? value.trim() : undefined);
  return {
    name: values.name.trim(),
    supportUrl: trim(values.supportUrl),
    accountManager: trim(values.accountManager),
    notes: trim(values.notes),
    contacts: (values.contacts ?? [])
      .map((contact) => ({ name: contact.name.trim(), email: trim(contact.email), phone: trim(contact.phone), role: trim(contact.role) }))
      .filter((contact) => contact.name),
  };
}

function fromVendor(vendor?: VendorResponse | null): VendorFormValues {
  return {
    name: vendor?.name ?? "",
    supportUrl: vendor?.supportUrl ?? "",
    accountManager: vendor?.accountManager ?? "",
    notes: vendor?.notes ?? "",
    contacts: vendor?.contacts?.length
      ? vendor.contacts.map((contact) => ({ name: contact.name ?? "", email: contact.email ?? "", phone: contact.phone ?? "", role: contact.role ?? "" }))
      : [],
  };
}

export function VendorForm({ vendorId }: { vendorId?: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const isEdit = Boolean(vendorId);

  const vendorQuery = useQuery({
    queryKey: ["vendor", vendorId],
    queryFn: () => api.getVendor(vendorId ?? ""),
    enabled: isEdit,
  });

  const defaults = useMemo(() => fromVendor(vendorQuery.data ?? null), [vendorQuery.data]);
  const form = useForm<VendorFormValues>({ resolver: zodResolver(vendorSchema), defaultValues: defaults });
  const contacts = useFieldArray({ control: form.control, name: "contacts" });

  useEffect(() => {
    form.reset(defaults);
  }, [defaults, form]);

  const mutation = useMutation({
    mutationFn: async (values: VendorFormValues) => {
      const payload = toPayload(values);
      return isEdit && vendorId ? api.updateVendor(vendorId, payload) : api.createVendor(payload);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["vendors"] });
      if (vendorId) {
        await queryClient.invalidateQueries({ queryKey: ["vendor", vendorId] });
      }
      toast.success(isEdit ? "Vendor updated" : "Vendor created");
      router.push("/vendors");
    },
    onError: (error: Error) => {
      const apiError = error as ApiError;
      toast.error(apiError.message);
      form.setError("root", { message: apiError.message });
    },
  });

  if (isEdit && vendorQuery.isError) {
    return <FormCard title="Vendor unavailable" description={vendorQuery.error instanceof Error ? vendorQuery.error.message : "Failed to load the vendor."}><div className="space-y-3"><p className="text-sm text-destructive">{vendorQuery.error instanceof Error ? vendorQuery.error.message : "Failed to load the vendor."}</p><Button type="button" variant="outline" onClick={() => router.push("/vendors")}>Back to vendors</Button></div></FormCard>;
  }

  if (isEdit && vendorQuery.isLoading) {
    return <VendorFormSkeleton />;
  }

  return (
    <FormCard title={isEdit ? "Edit vendor" : "New vendor"} description="Maintain vendors and their contact roster.">
      <form className="space-y-6" onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Name" error={form.formState.errors.name}><Input {...form.register("name")} /></Field>
          <Field label="Support URL" error={form.formState.errors.supportUrl}><Input {...form.register("supportUrl")} placeholder="https://support.example.com" /></Field>
          <Field label="Account manager" error={form.formState.errors.accountManager}><Input {...form.register("accountManager")} /></Field>
        </div>
        <Field label="Notes" error={form.formState.errors.notes}><Textarea rows={4} {...form.register("notes")} /></Field>

        <div className="space-y-4 rounded-lg border p-4">
          <div className="flex items-center justify-between gap-2">
            <div>
              <h3 className="font-medium">Contacts</h3>
              <p className="text-sm text-muted-foreground">Add one or more vendor contacts.</p>
            </div>
            <Button type="button" variant="outline" onClick={() => contacts.append({ name: "", email: "", phone: "", role: "" })}>Add contact</Button>
          </div>
          <div className="space-y-4">
            {contacts.fields.length ? contacts.fields.map((field, index) => (
              <div key={field.id} className={cn("grid gap-4 rounded-md border p-4 md:grid-cols-2", index % 2 === 1 && "bg-muted/30")}>
                <Field label="Contact name" error={form.formState.errors.contacts?.[index]?.name}><Input {...form.register(`contacts.${index}.name` as const)} /></Field>
                <Field label="Email" error={form.formState.errors.contacts?.[index]?.email}><Input {...form.register(`contacts.${index}.email` as const)} /></Field>
                <Field label="Phone" error={form.formState.errors.contacts?.[index]?.phone}><Input {...form.register(`contacts.${index}.phone` as const)} /></Field>
                <Field label="Role" error={form.formState.errors.contacts?.[index]?.role}><Input {...form.register(`contacts.${index}.role` as const)} /></Field>
                <div className="md:col-span-2 flex justify-end">
                  <Button type="button" variant="ghost" onClick={() => contacts.remove(index)}>Remove contact</Button>
                </div>
              </div>
            )) : <p className="text-sm text-muted-foreground">No contacts added yet.</p>}
          </div>
        </div>

        {form.formState.errors.root?.message ? <p className="text-sm text-destructive">{form.formState.errors.root.message}</p> : null}
        <div className="flex gap-2">
          <Button type="submit" disabled={mutation.isPending}>{mutation.isPending ? "Saving…" : "Save vendor"}</Button>
          <Button type="button" variant="outline" onClick={() => router.push("/vendors")}>Cancel</Button>
        </div>
      </form>
    </FormCard>
  );
}

function VendorFormSkeleton() {
  return (
    <FormCard title="Loading vendor" description="Fetching record details…">
      <div className="grid gap-4 md:grid-cols-2">
        {Array.from({ length: 4 }).map((_, index) => <Skeleton key={index} className="h-16 w-full" />)}
      </div>
    </FormCard>
  );
}
