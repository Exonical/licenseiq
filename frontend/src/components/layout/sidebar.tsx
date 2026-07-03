"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BarChart3, Boxes, Database, Home, Search, Shield, Sparkles, Tags, Users } from "lucide-react";
import { cn } from "@/lib/utils";

const sections = [
  { href: "/", label: "Dashboard", icon: Home, enabled: true },
  { href: "/licenses", label: "Licenses", icon: Database, enabled: false },
  { href: "/vendors", label: "Vendors", icon: Users, enabled: false },
  { href: "/products", label: "Products", icon: Boxes, enabled: false },
  { href: "/assignments", label: "Assignments", icon: Tags, enabled: false },
  { href: "/reports", label: "Reports", icon: BarChart3, enabled: false },
  { href: "/search", label: "Search", icon: Search, enabled: false },
  { href: "/admin", label: "Admin", icon: Shield, enabled: false },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex h-full w-72 flex-col border-r bg-background/90 backdrop-blur">
      <div className="flex items-center gap-3 border-b px-6 py-5">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary text-primary-foreground">
          <Sparkles className="h-5 w-5" />
        </div>
        <div>
          <p className="text-sm font-semibold leading-none">LicenseIQ</p>
          <p className="text-xs text-muted-foreground">Air-gapped operations</p>
        </div>
      </div>
      <nav className="flex-1 space-y-1 p-3" aria-label="Primary">
        {sections.map((item) => {
          const active = pathname === item.href;
          const Icon = item.icon;
          if (!item.enabled) {
            return (
              <div key={item.href} className="flex cursor-not-allowed items-center gap-3 rounded-lg px-3 py-2 text-sm text-muted-foreground opacity-60" aria-disabled="true">
                <Icon className="h-4 w-4" />
                <span>{item.label}</span>
                <span className="ml-auto text-xs uppercase tracking-wide">Soon</span>
              </div>
            );
          }
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors hover:bg-accent hover:text-accent-foreground",
                active && "bg-accent text-accent-foreground",
              )}
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>
      <div className="border-t p-4 text-xs text-muted-foreground">
        <p>Backend-driven UI scaffold</p>
        <p>No remote assets or telemetry</p>
      </div>
    </aside>
  );
}
