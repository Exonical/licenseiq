import type { Metadata } from "next";
import { VendorList } from "@/components/vendors/vendor-list";

export const metadata: Metadata = {
  title: "Vendors | LicenseIQ",
};

export default function VendorsPage() {
  return <VendorList />;
}
