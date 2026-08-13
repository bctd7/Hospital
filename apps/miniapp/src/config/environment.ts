const DEFAULT_API_BASE_URL = "http://127.0.0.1:8888";

export type DevAuthRole = "patient" | "doctor" | "admin";

function normalizeBaseUrl(value?: string): string {
  const normalized = value?.trim().replace(/\/+$/, "");
  return normalized || DEFAULT_API_BASE_URL;
}

export function resolveDevAuthBypassEnabled(
  development: boolean,
  value?: string,
): boolean {
  return development && value?.trim().toLowerCase() === "true";
}

export function resolveDevAuthRole(value?: string): DevAuthRole {
  const normalized = value?.trim().toLowerCase();
  return normalized === "doctor" || normalized === "admin"
    ? normalized
    : "patient";
}

export const API_BASE_URL = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL);
export const API_REQUEST_TIMEOUT_MS = 8000;
export const API_TRANSPORT =
  import.meta.env.VITE_API_TRANSPORT === "cloudbase" ? "cloudbase" : "direct";
export const CLOUDBASE_ENV_ID = import.meta.env.VITE_CLOUDBASE_ENV_ID?.trim() ?? "";
export const ANYSERVICE_NAME = import.meta.env.VITE_ANYSERVICE_NAME?.trim() ?? "";
export const DEV_AUTH_BYPASS_ENABLED = resolveDevAuthBypassEnabled(
  import.meta.env.DEV,
  import.meta.env.VITE_DEV_AUTH_BYPASS,
);
export const DEV_AUTH_ROLE = resolveDevAuthRole(
  import.meta.env.VITE_DEV_AUTH_ROLE,
);
