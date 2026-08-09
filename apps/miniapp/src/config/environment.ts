const DEFAULT_API_BASE_URL = "http://127.0.0.1:8888";

function normalizeBaseUrl(value?: string): string {
  const normalized = value?.trim().replace(/\/+$/, "");
  return normalized || DEFAULT_API_BASE_URL;
}

export const API_BASE_URL = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL);
export const API_REQUEST_TIMEOUT_MS = 8000;
const configuredStaffDataSource = import.meta.env.VITE_STAFF_DATA_SOURCE;

// Mock 仅允许开发构建启用；生产构建始终走真实 HTTP，避免演示数据误入发布版本。
export const STAFF_DATA_SOURCE = import.meta.env.DEV
  ? configuredStaffDataSource === "http"
    ? "http"
    : "mock"
  : "http";
