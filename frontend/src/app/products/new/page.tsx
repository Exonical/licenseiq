import type { Metadata } from "next";
import { ProductForm } from "@/components/products/product-form";

export const metadata: Metadata = {
  title: "New product | LicenseIQ",
};

export default function NewProductPage() {
  return <ProductForm />;
}
