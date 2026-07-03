import type { Metadata } from "next";
import { VendorForm } from "@/components/vendors/vendor-form";

export const metadata: Metadata = {
  title: "Edit vendor | LicenseIQ",
};

export default async function VendorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <VendorForm vendorId={id} />;
}
