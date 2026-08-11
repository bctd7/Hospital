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
