import type { Metadata } from "next";
import { VendorForm } from "@/components/vendors/vendor-form";

export const metadata: Metadata = {
  title: "New vendor | LicenseIQ",
};

export default function NewVendorPage() {
  return <VendorForm />;
}
