import type { Metadata } from "next";
import { LicenseForm } from "@/components/licenses/license-form";

export const metadata: Metadata = {
  title: "Edit license | LicenseIQ",
};

export default async function LicensePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <LicenseForm licenseId={id} />;
}
