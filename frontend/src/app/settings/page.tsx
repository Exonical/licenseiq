import type { Metadata } from "next";
import { AuthSettingsCard } from "@/components/auth/auth-settings-card";

export const metadata: Metadata = {
  title: "Auth settings | LicenseIQ",
};

export default function SettingsPage() {
  return <AuthSettingsCard />;
}
