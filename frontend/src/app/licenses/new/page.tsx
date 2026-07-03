import type { Metadata } from "next";
import { LicenseForm } from "@/components/licenses/license-form";

export const metadata: Metadata = {
  title: "New license | LicenseIQ",
};

export default function NewLicensePage() {
  return <LicenseForm />;
}
