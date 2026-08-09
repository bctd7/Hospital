const DEFAULT_API_BASE_URL = "http://127.0.0.1:8888";

function normalizeBaseUrl(value?: string): string {
  const normalized = value?.trim().replace(/\/+$/, "");
  return normalized || DEFAULT_API_BASE_URL;
}

export const API_BASE_URL = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL);
export const API_REQUEST_TIMEOUT_MS = 8000;
const configuredStaffDataSource = import.meta.env.VITE_STAFF_DATA_SOURCE;

// 开发服务和显式 mock 模式使用演示数据；普通生产构建始终走真实 HTTP。
export const STAFF_DATA_SOURCE =
  import.meta.env.MODE === "mock" ||
  (import.meta.env.DEV && configuredStaffDataSource !== "http")
    ? "mock"
    : "http";
