import type { components } from "@/lib/api/types";

export type ReportTable = {
  title: string;
  columns: string[];
  rows: string[][];
  totals?: SummaryRow[];
};

export type SummaryRow = {
  label: string;
  values: string[];
};

export type LicenseResponse = components["schemas"]["LicenseResponse"];
export type LicenseBody = components["schemas"]["LicenseBody"];
export type LicensePage = components["schemas"]["PageLicenseResponse"];
export type VendorResponse = components["schemas"]["VendorResponse"];
export type VendorBody = components["schemas"]["VendorBody"];
export type VendorPage = components["schemas"]["PageVendorResponse"];
export type ProductResponse = components["schemas"]["ProductResponse"];
export type ProductBody = components["schemas"]["ProductBody"];
export type ProductPage = components["schemas"]["PageProductResponse"];
export type LicenseIssueLink = components["schemas"]["LicenseIssueLinkResponse"];
export type LicenseIssueLinkPage = components["schemas"]["PageLicenseIssueLinkResponse"];
export type JiraLinkIssueBody = components["schemas"]["JiraLinkIssueBody"];
export type JiraIssueStatusBody = components["schemas"]["JiraIssueStatusBody"];
