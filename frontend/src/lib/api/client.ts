import type { LicenseBody, LicenseIssueLink, LicensePage, LicenseResponse, LicenseIssueLinkPage, ProductBody, ProductPage, ProductResponse, VendorBody, VendorPage, VendorResponse, JiraIssueStatusBody, JiraLinkIssueBody } from "@/lib/api/contracts";
import { offsetSchema, upcomingRenewalsWindowSchema } from "@/lib/validation";

export type ClientOptions = {
  baseUrl?: string;
  getAuthToken?: () => string | undefined;
};

export class ApiError extends Error {
  status: number;
  details?: string[];

  constructor(message: string, status: number, details?: string[]) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.details = details;
  }
}

const defaultBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api/v1";
let defaultAuthTokenProvider: (() => string | undefined) | undefined;

export function setAuthTokenProvider(provider?: () => string | undefined) {
  defaultAuthTokenProvider = provider;
}

function normalizeBaseUrl(baseUrl?: string) {
  return (baseUrl ?? defaultBaseUrl).replace(/\/$/, "");
}

function buildUrl(path: string, baseUrl?: string, query?: Record<string, string | number | undefined>) {
  const target = `${normalizeBaseUrl(baseUrl)}${path.startsWith("/") ? "" : "/"}${path}`;
  const url = target.startsWith("http://") || target.startsWith("https://") ? new URL(target) : new URL(target, "http://localhost");
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== "") {
      url.searchParams.set(key, String(value));
    }
  }
  return target.startsWith("http://") || target.startsWith("https://") ? url.toString() : `${url.pathname}${url.search}`;
}

async function buildErrorMessage(response: Response) {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const body = (await response.json()) as {
        message?: string;
        error?: string;
        title?: string;
        detail?: string;
        errors?: Array<{ location?: string; message?: string; value?: unknown }>;
      };
      const details = body.errors?.map((item) => [item.location, item.message].filter(Boolean).join(": ")).filter(Boolean) ?? [];
      return {
        message: body.message ?? body.detail ?? body.title ?? body.error ?? `Request failed with status ${response.status}`,
        details,
      };
    } catch {
      return { message: `Request failed with status ${response.status}` as const, details: [] as string[] };
    }
  }
  return { message: `Request failed with status ${response.status}` as const, details: [] as string[] };
}

function buildHeaders(getAuthToken?: () => string | undefined, hasBody = false) {
  const headers = new Headers({ Accept: "application/json" });
  if (hasBody) {
    headers.set("Content-Type", "application/json");
  }
  const token = getAuthToken?.() ?? defaultAuthTokenProvider?.();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return headers;
}

async function readJson<T>(response: Response): Promise<T> {
  if (response.status === 204 || response.status === 205) {
    return undefined as T;
  }
  if (!response.ok) {
    const error = await buildErrorMessage(response);
    throw new ApiError(error.message, response.status, error.details);
  }
  const text = await response.text();
  if (!text) {
    return undefined as T;
  }
  return JSON.parse(text) as T;
}

async function request<T>(path: string, options: ClientOptions & { method?: string; query?: Record<string, string | number | undefined>; body?: unknown } = {}) {
  const response = await fetch(buildUrl(path, options.baseUrl, options.query), {
    method: options.method ?? (options.body ? "POST" : "GET"),
    headers: buildHeaders(options.getAuthToken, options.body !== undefined),
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    cache: "no-store",
  });
  return readJson<T>(response);
}

export const api = {
  getUpcomingRenewals(input: { windowDays?: number; asOf?: string | Date } = {}) {
    return request<import("@/lib/api/contracts").ReportTable>("/reports/renewals", {
      query: {
        format: "json",
        windowDays: upcomingRenewalsWindowSchema.parse(input.windowDays ?? 90),
        asOf: input.asOf instanceof Date ? input.asOf.toISOString() : input.asOf,
      },
    });
  },
  getExpiredLicenses(input: { asOf?: string | Date } = {}) {
    return request<import("@/lib/api/contracts").ReportTable>("/reports/expired", {
      query: {
        format: "json",
        asOf: input.asOf instanceof Date ? input.asOf.toISOString() : input.asOf,
      },
    });
  },
  getVendorSpend(input: { asOf?: string | Date } = {}) {
    return request<import("@/lib/api/contracts").ReportTable>("/reports/vendor-spend", {
      query: {
        format: "json",
        asOf: input.asOf instanceof Date ? input.asOf.toISOString() : input.asOf,
      },
    });
  },
  getLicenseUtilization(input: { asOf?: string | Date } = {}) {
    return request<import("@/lib/api/contracts").ReportTable>("/reports/utilization", {
      query: {
        format: "json",
        asOf: input.asOf instanceof Date ? input.asOf.toISOString() : input.asOf,
      },
    });
  },
  listLicenses(input: { limit?: number; offset?: number } = {}) {
    return request<LicensePage>("/licenses", {
      query: {
        limit: input.limit ?? 100,
        offset: offsetSchema.parse(input.offset ?? 0),
      },
    });
  },
  getLicense(id: string) {
    return request<LicenseResponse>(`/licenses/${id}`);
  },
  createLicense(body: LicenseBody) {
    return request<LicenseResponse>("/licenses", { method: "POST", body });
  },
  updateLicense(id: string, body: LicenseBody) {
    return request<LicenseResponse>(`/licenses/${id}`, { method: "PUT", body });
  },
  deleteLicense(id: string) {
    return request<void>(`/licenses/${id}`, { method: "DELETE" });
  },
  listVendors(input: { limit?: number; offset?: number } = {}) {
    return request<VendorPage>("/vendors", {
      query: {
        limit: input.limit ?? 100,
        offset: offsetSchema.parse(input.offset ?? 0),
      },
    });
  },
  getVendor(id: string) {
    return request<VendorResponse>(`/vendors/${id}`);
  },
  createVendor(body: VendorBody) {
    return request<VendorResponse>("/vendors", { method: "POST", body });
  },
  updateVendor(id: string, body: VendorBody) {
    return request<VendorResponse>(`/vendors/${id}`, { method: "PUT", body });
  },
  deleteVendor(id: string) {
    return request<void>(`/vendors/${id}`, { method: "DELETE" });
  },
  listProducts(input: { limit?: number; offset?: number } = {}) {
    return request<ProductPage>("/products", {
      query: {
        limit: input.limit ?? 100,
        offset: offsetSchema.parse(input.offset ?? 0),
      },
    });
  },
  getProduct(id: string) {
    return request<ProductResponse>(`/products/${id}`);
  },
  createProduct(body: ProductBody) {
    return request<ProductResponse>("/products", { method: "POST", body });
  },
  updateProduct(id: string, body: ProductBody) {
    return request<ProductResponse>(`/products/${id}`, { method: "PUT", body });
  },
  deleteProduct(id: string) {
    return request<void>(`/products/${id}`, { method: "DELETE" });
  },
  listLicenseIssueLinks(id: string) {
    return request<LicenseIssueLinkPage>(`/licenses/${id}/jira/issues`);
  },
  createLicenseRenewalTicket(id: string) {
    return request<LicenseIssueLink>(`/licenses/${id}/jira/renewal-tickets`, { method: "POST" });
  },
  linkLicenseIssue(id: string, body: JiraLinkIssueBody) {
    return request<LicenseIssueLink>(`/licenses/${id}/jira/issues`, { method: "POST", body });
  },
  updateLicenseIssueStatus(id: string, issueKey: string, body: JiraIssueStatusBody) {
    return request<LicenseIssueLink>(`/licenses/${id}/jira/issues/${issueKey}/status`, { method: "PUT", body });
  },
};
