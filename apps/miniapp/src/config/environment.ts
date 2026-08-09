const DEFAULT_API_BASE_URL = "http://127.0.0.1:8888";

function normalizeBaseUrl(value?: string): string {
  const normalized = value?.trim().replace(/\/+$/, "");
  return normalized || DEFAULT_API_BASE_URL;
}

export const API_BASE_URL = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL);
export const API_REQUEST_TIMEOUT_MS = 8000;
export const API_TRANSPORT =
  import.meta.env.VITE_API_TRANSPORT === "cloudbase" ? "cloudbase" : "direct";
export const CLOUDBASE_ENV_ID = import.meta.env.VITE_CLOUDBASE_ENV_ID?.trim() ?? "";
export const ANYSERVICE_NAME = import.meta.env.VITE_ANYSERVICE_NAME?.trim() ?? "";
const configuredStaffDataSource = import.meta.env.VITE_STAFF_DATA_SOURCE;

export function resolveStaffDataSource(
  mode: string,
  configured: string | undefined,
  isDevelopment: boolean,
): "mock" | "http" {
  return mode === "mock" || configured === "mock" || (isDevelopment && configured !== "http")
    ? "mock"
    : "http";
}

// The experience build may explicitly keep directory/admin demo data in mock
// mode while authentication and sessions still use the real CloudBase backend.
export const STAFF_DATA_SOURCE = resolveStaffDataSource(
  import.meta.env.MODE,
  configuredStaffDataSource,
  import.meta.env.DEV,
);
