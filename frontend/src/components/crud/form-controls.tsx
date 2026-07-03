"use client";

import type { ReactNode } from "react";
import type { FieldError } from "react-hook-form";
import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";

export function Field({ label, error, children, hint, className }: { label: string; error?: FieldError | { message?: string }; children: ReactNode; hint?: string; className?: string; }) {
  return (
    <div className={cn("space-y-2", className)}>
      <Label>{label}</Label>
      {children}
      {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      {error?.message ? <p className="text-sm text-destructive">{error.message}</p> : null}
    </div>
  );
}
