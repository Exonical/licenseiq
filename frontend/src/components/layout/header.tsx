"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Bell, ChevronDown, LogOut, MoonStar, Settings2, SunMedium } from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useTheme } from "@/components/providers/theme-provider";
import { useAuth } from "@/components/providers/auth-provider";

export function Header() {
  const { theme, toggleTheme } = useTheme();
  const auth = useAuth();
  const pathname = usePathname();
  const title = pathname.startsWith("/licenses")
    ? "Licenses"
    : pathname.startsWith("/vendors")
      ? "Vendors"
      : pathname.startsWith("/products")
        ? "Products"
        : pathname.startsWith("/settings")
          ? "Auth settings"
          : "Dashboard";

  return (
    <header className="flex h-16 items-center justify-between border-b bg-background/80 px-6 backdrop-blur">
      <div>
        <p className="text-sm font-medium text-muted-foreground">LicenseIQ</p>
        <h1 className="text-lg font-semibold">{title}</h1>
      </div>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="icon" onClick={toggleTheme} aria-label="Toggle theme">
          {theme === "dark" ? <SunMedium className="h-4 w-4" /> : <MoonStar className="h-4 w-4" />}
        </Button>
        <Button variant="outline" size="icon" aria-label="Notifications">
          <Bell className="h-4 w-4" />
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" className="gap-2 pl-2 pr-3">
              <Avatar className="h-7 w-7">
                <AvatarFallback>{auth.keyId ? auth.keyId.slice(0, 2).toUpperCase() : "LI"}</AvatarFallback>
              </Avatar>
              <span className="text-sm">{auth.authenticated ? auth.label : "Not signed in"}</span>
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-64">
            <DropdownMenuLabel>Session</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem disabled>{auth.authenticated ? "Authenticated" : "No credential loaded"}</DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/settings" className="flex w-full items-center gap-2">
                <Settings2 className="h-4 w-4" />
                Auth settings
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={auth.signOut} disabled={!auth.authenticated} className="flex items-center gap-2">
              <LogOut className="h-4 w-4" />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
