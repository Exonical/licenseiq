import type { LicensePage, ReportTable } from "@/lib/api/contracts";
import { offsetSchema, upcomingRenewalsWindowSchema } from "@/lib/validation";

export type ClientOptions = {
  baseUrl?: string;
  getAuthToken?: () => string | undefined;
};

const defaultBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api/v1";

function normalizeBaseUrl(baseUrl?: string) {
  return (baseUrl ?? defaultBaseUrl).replace(/\/$/, "");
}

async function readJson<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(await buildErrorMessage(response));
  }
  return response.json() as Promise<T>;
}

async function buildErrorMessage(response: Response) {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      const body = (await response.json()) as { message?: string; error?: string };
      return body.message ?? body.error ?? `Request failed with status ${response.status}`;
    } catch {
      return `Request failed with status ${response.status}`;
    }
  }
  return `Request failed with status ${response.status}`;
}

function buildHeaders(getAuthToken?: () => string | undefined) {
  const headers = new Headers({ Accept: "application/json" });
  const token = getAuthToken?.();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  return headers;
}

async function request<T>(path: string, options: ClientOptions & { query?: Record<string, string | number | undefined> } = {}) {
  const baseUrl = normalizeBaseUrl(options.baseUrl);
  const target = `${baseUrl}${path.startsWith("/") ? "" : "/"}${path}`;
  const url = target.startsWith("http://") || target.startsWith("https://") ? new URL(target) : new URL(target, "http://localhost");
  for (const [key, value] of Object.entries(options.query ?? {})) {
    if (value !== undefined && value !== "") {
      url.searchParams.set(key, String(value));
    }
  }
  const requestUrl = target.startsWith("http://") || target.startsWith("https://") ? url.toString() : `${url.pathname}${url.search}`;
  const response = await fetch(requestUrl, {
    headers: buildHeaders(options.getAuthToken),
    cache: "no-store",
  });
  return readJson<T>(response);
}

export const api = {
  getUpcomingRenewals(input: { windowDays?: number; asOf?: string | Date } = {}) {
    return request<ReportTable>("/reports/renewals", {
      query: {
        format: "json",
        windowDays: upcomingRenewalsWindowSchema.parse(input.windowDays ?? 90),
        asOf: input.asOf instanceof Date ? input.asOf.toISOString() : input.asOf,
      },
    });
  },
  getExpiredLicenses(input: { asOf?: string | Date } = {}) {
    return request<ReportTable>("/reports/expired", {
      query: {
        format: "json",
        asOf: input.asOf instanceof Date ? input.asOf.toISOString() : input.asOf,
      },
    });
  },
  getVendorSpend(input: { asOf?: string | Date } = {}) {
    return request<ReportTable>("/reports/vendor-spend", {
      query: {
        format: "json",
        asOf: input.asOf instanceof Date ? input.asOf.toISOString() : input.asOf,
      },
    });
  },
  getLicenseUtilization(input: { asOf?: string | Date } = {}) {
    return request<ReportTable>("/reports/utilization", {
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
};
