"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/components/providers/auth-provider";

export function AuthSettingsCard() {
  const auth = useAuth();
  const [token, setToken] = useState(auth.token ?? "");

  return (
    <div className="mx-auto grid max-w-2xl gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Authentication</CardTitle>
          <CardDescription>
            Enter a real LicenseIQ API key or bearer token. The token is stored locally in your browser and attached to all API requests.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="token">Credential</Label>
            <Input id="token" value={token} onChange={(event) => setToken(event.target.value)} placeholder="liq_keyid.secret or bearer token" autoComplete="off" spellCheck={false} />
          </div>
          <p className="text-sm text-muted-foreground">
            For air-gapped use, paste the bootstrap administrator API key or another issued token. OIDC remains available for interactive logins later.
          </p>
          <div className="flex flex-wrap gap-2">
            <Button
              onClick={() => {
                auth.setToken(token.trim() || null);
                toast.success(token.trim() ? "Credential saved" : "Credential cleared");
              }}
            >
              Save credential
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                setToken("");
                auth.signOut();
                toast.message("Signed out");
              }}
            >
              Clear credential
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
