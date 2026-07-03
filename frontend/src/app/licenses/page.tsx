import type { Metadata } from "next";
import { LicenseList } from "@/components/licenses/license-list";

export const metadata: Metadata = {
  title: "Licenses | LicenseIQ",
};

export default function LicensesPage() {
  return <LicenseList />;
}
