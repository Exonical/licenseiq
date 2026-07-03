"use client";

import { useQueries } from "@tanstack/react-query";
import type { ComponentType, ReactNode } from "react";
import { AlertCircle, ArrowUpRight, PieChart, Rows3, ShieldAlert, WalletCards } from "lucide-react";
import { api } from "@/lib/api/client";
import type { ReportTable } from "@/lib/api/contracts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

const currencyFormat = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 2,
});

type SectionCardProps = {
  title: string;
  description: string;
  value: string;
  loading: boolean;
  error: string | null;
  icon: ComponentType<{ className?: string }>;
  details?: ReactNode;
};

function SectionCard({ title, description, value, loading, error, icon: Icon, details }: SectionCardProps) {
  return (
    <Card className="h-full">
      <CardHeader className="space-y-2">
        <div className="flex items-start justify-between gap-4">
          <div>
            <CardDescription>{description}</CardDescription>
            <CardTitle className="mt-1 text-2xl">{loading ? <Skeleton className="h-8 w-24" /> : value}</CardTitle>
          </div>
          <div className="rounded-full bg-primary/10 p-2 text-primary">
            <Icon className="h-5 w-5" />
          </div>
        </div>
        <h2 className="text-sm font-medium text-foreground">{title}</h2>
      </CardHeader>
      <CardContent className="space-y-3 text-sm text-muted-foreground">
        {loading ? <Skeleton className="h-16 w-full" /> : error ? <div className="flex items-center gap-2 rounded-lg border border-dashed p-3 text-destructive"><AlertCircle className="h-4 w-4" />{error}</div> : details ?? <p>No additional details available.</p>}
      </CardContent>
    </Card>
  );
}

export function DashboardPage() {
  const [renewals, expired, utilization, vendorSpend, licenses] = useQueries({
    queries: [
      { queryKey: ["report", "renewals"], queryFn: () => api.getUpcomingRenewals({ windowDays: 90 }) },
      { queryKey: ["report", "expired"], queryFn: () => api.getExpiredLicenses() },
      { queryKey: ["report", "utilization"], queryFn: () => api.getLicenseUtilization() },
      { queryKey: ["report", "vendor-spend"], queryFn: () => api.getVendorSpend() },
      { queryKey: ["licenses", "unassigned"], queryFn: () => api.listLicenses({ limit: 500 }) },
    ],
  });

  const upcomingRows = renewals.data?.rows ?? [];
  const expiredRows = expired.data?.rows ?? [];
  const utilizationRows = utilization.data?.rows ?? [];
  const vendorRows = vendorSpend.data?.rows ?? [];
  const licenseRows = licenses.data?.data ?? [];
  const unassignedCount = licenseRows.filter((license) => !license.department?.trim()).length;
  const vendorTotals = vendorSpend.data?.totals ?? [];

  const cards = [
    {
      title: "Upcoming renewals",
      description: "Licenses renewing within 90 days",
      value: String(upcomingRows.length),
      loading: renewals.isLoading,
      error: renewals.error ? "Renewals report unavailable" : null,
      icon: ArrowUpRight,
      details: <CompactTable headers={["License", "Renewal"]} rows={upcomingRows.slice(0, 3).map((row) => [row[0], row[3]])} emptyLabel="No upcoming renewals." />,
    },
    {
      title: "Expiring licenses",
      description: "Expired by renewal or maintenance date",
      value: String(expiredRows.length),
      loading: expired.isLoading,
      error: expired.error ? "Expired licenses report unavailable" : null,
      icon: ShieldAlert,
      details: <CompactTable headers={["License", "Expired by"]} rows={expiredRows.slice(0, 3).map((row) => [row[0], row[6]])} emptyLabel="No expired licenses." />,
    },
    {
      title: "License utilization",
      description: "Seat counts and utilization overview",
      value: utilization.data?.totals?.[0]?.values?.[3] ?? "—",
      loading: utilization.isLoading,
      error: utilization.error ? "Utilization report unavailable" : null,
      icon: PieChart,
      details: <CompactTable headers={["License", "Utilization"]} rows={utilizationRows.slice(0, 3).map((row) => [row[0], row[6]])} emptyLabel="No utilization data." />,
    },
    {
      title: "Vendor spend",
      description: "Spend grouped by vendor and currency",
      value: vendorTotals.length > 0 ? currencyFormat.format(Number(vendorTotals[0].values[0])) : vendorRows.length > 0 ? `${vendorRows.length} groups` : "—",
      loading: vendorSpend.isLoading,
      error: vendorSpend.error ? "Vendor spend report unavailable" : null,
      icon: WalletCards,
      details: <CompactTable headers={["Vendor", "Currency", "Total"]} rows={vendorRows.slice(0, 3).map((row) => [row[0], row[1], row[2]])} emptyLabel="No vendor spend data." />,
    },
    {
      title: "Monthly spend",
      description: "No monthly spend endpoint is available yet",
      value: "—",
      loading: false,
      error: null,
      icon: Rows3,
      details: <EmptyState note="No backend endpoint yet. Add a monthly spend report in a later milestone." />,
    },
    {
      title: "Unassigned licenses",
      description: "Licenses missing a department",
      value: licenses.isLoading ? "…" : String(unassignedCount),
      loading: licenses.isLoading,
      error: licenses.error ? "License list unavailable" : null,
      icon: Rows3,
      details: <CompactTable headers={["License", "Department"]} rows={licenseRows.filter((license) => !license.department?.trim()).slice(0, 3).map((license) => [license.licenseKey || license.id, license.department || "Unassigned"]) } emptyLabel="All licenses are assigned to a department." />,
    },
    {
      title: "Compliance overview",
      description: "No compliance endpoint is available yet",
      value: "—",
      loading: false,
      error: null,
      icon: AlertCircle,
      details: <EmptyState note="No backend endpoint yet. This card intentionally stays empty until compliance reporting exists." />,
    },
  ];

  return (
    <div className="space-y-6">
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {cards.map((card) => (
          <SectionCard key={card.title} {...card} />
        ))}
      </section>

      <section className="grid gap-4 xl:grid-cols-2">
        <ReportPreview title={renewals.data?.title ?? "Upcoming renewals"} description="Detailed upcoming renewals table" report={renewals.data} loading={renewals.isLoading} error={renewals.error ? "Renewals report unavailable" : null} />
        <ReportPreview title={vendorSpend.data?.title ?? "Vendor spend"} description="Detailed vendor spend table" report={vendorSpend.data} loading={vendorSpend.isLoading} error={vendorSpend.error ? "Vendor spend report unavailable" : null} />
      </section>
    </div>
  );
}

function ReportPreview({ title, description, report, loading, error }: { title: string; description: string; report?: ReportTable; loading: boolean; error: string | null; }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{description}</CardDescription>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : error ? (
          <div className="rounded-lg border border-dashed p-4 text-sm text-destructive">{error}</div>
        ) : report ? (
          <div className="space-y-4">
            <Table>
              <TableHeader>
                <TableRow>
                  {report.columns.slice(0, 4).map((column) => (
                    <TableHead key={column}>{column}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {report.rows.slice(0, 5).map((row, index) => (
                  <TableRow key={`${report.title}-${index}`}>
                    {row.slice(0, 4).map((cell, cellIndex) => (
                      <TableCell key={`${cell}-${cellIndex}`}>{cell || "—"}</TableCell>
                    ))}
                  </TableRow>
                ))}
                {report.rows.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={report.columns.length} className="py-8 text-center text-muted-foreground">
                      No rows returned.
                    </TableCell>
                  </TableRow>
                ) : null}
              </TableBody>
            </Table>
            {report.totals?.length ? (
              <div className="rounded-lg border bg-muted/40 p-4 text-sm">
                <p className="mb-2 font-medium text-foreground">Totals</p>
                <div className="space-y-1 text-muted-foreground">
                  {report.totals.map((total) => (
                    <p key={total.label}>
                      <span className="font-medium text-foreground">{total.label}:</span> {total.values.join(" • ")}
                    </p>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
        ) : (
          <EmptyState note="No data returned." />
        )}
      </CardContent>
    </Card>
  );
}

function CompactTable({ headers, rows, emptyLabel }: { headers: string[]; rows: string[][]; emptyLabel: string; }) {
  if (rows.length === 0) {
    return <EmptyState note={emptyLabel} />;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {headers.map((header) => (
            <TableHead key={header}>{header}</TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row, index) => (
          <TableRow key={index}>
            {row.map((cell, cellIndex) => (
              <TableCell key={`${cell}-${cellIndex}`}>{cell || "—"}</TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function EmptyState({ note }: { note: string }) {
  return <div className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">{note}</div>;
}
