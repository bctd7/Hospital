import {
  API_BASE_URL,
  API_REQUEST_TIMEOUT_MS,
  API_TRANSPORT,
} from "@/config/environment";
import { callAnyService, type CloudContainerResult } from "@/platform/cloudbase";

type RequestMethod = "GET" | "POST" | "PUT" | "DELETE";

interface RequestOptions<TData = unknown> {
  path: string;
  method?: RequestMethod;
  data?: TData;
  authenticated?: boolean;
  retryOnUnauthorized?: boolean;
  timeoutMs?: number;
}

interface AuthAdapter {
  getAccessToken: () => string;
  prepareAuthenticatedRequest?: () => Promise<boolean>;
  refreshOnce: () => Promise<boolean>;
  handleUnauthorized: (rejectedAccessToken: string) => Promise<void>;
}

interface ErrorPayload {
  code?: string;
  message?: string;
  error?: string;
}

let authAdapter: AuthAdapter | undefined;

export class ApiError extends Error {
  readonly code: string;
  readonly statusCode: number;
  readonly networkError: boolean;

  constructor(message: string, statusCode = 0, networkError = false, code = "") {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.statusCode = statusCode;
    this.networkError = networkError;
  }
}

export function configureAuthAdapter(adapter: AuthAdapter) {
  authAdapter = adapter;
}

function errorMessage(data: unknown, fallback: string): string {
  if (typeof data === "string" && data.trim()) {
    return data.trim();
  }

  if (data && typeof data === "object") {
    const payload = data as ErrorPayload;
    return payload.message?.trim() || payload.error?.trim() || fallback;
  }

  return fallback;
}

function normalizedPath(path: string): string {
  return path.startsWith("/") ? path : `/${path}`;
}

function directRequest<TData>(
  options: RequestOptions<TData>,
  headers: Record<string, string>,
): Promise<CloudContainerResult> {
  return new Promise((resolve, reject) => {
    uni.request({
      url: `${API_BASE_URL}${normalizedPath(options.path)}`,
      method: options.method ?? "GET",
      data: options.data as UniApp.RequestOptions["data"],
      header: headers,
      timeout: options.timeoutMs ?? API_REQUEST_TIMEOUT_MS,
      success: (response) => {
        resolve({ statusCode: response.statusCode, data: response.data });
      },
      fail: () => {
        reject(new ApiError("网络连接失败，请稍后重试", 0, true));
      },
    });
  });
}

async function transportRequest<TData>(
  options: RequestOptions<TData>,
  headers: Record<string, string>,
): Promise<CloudContainerResult> {
  if (API_TRANSPORT === "cloudbase") {
    try {
      return await callAnyService({
        path: normalizedPath(options.path),
        method: options.method ?? "GET",
        data: options.data,
        header: headers,
      });
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      throw new ApiError("网络连接失败，请稍后重试", 0, true);
    }
  }

  return directRequest(options, headers);
}

async function sendRequest<TResponse, TData>(
  options: RequestOptions<TData>,
  retried: boolean,
): Promise<TResponse> {
  const authenticated = options.authenticated ?? false;
  let accessToken = authenticated ? authAdapter?.getAccessToken() ?? "" : "";

  if (authenticated && !accessToken) {
    throw new ApiError("需要登录后才能执行此操作", 401);
  }

  // 冷启动恢复的是本地快照，服务端会话可能已经重启或失效。首批业务请求
  // 共享一次预检刷新，避免多个页面拿旧 Token 同时撞出一串 401。
  if (authenticated && !retried && authAdapter?.prepareAuthenticatedRequest) {
    const rejectedAccessToken = accessToken;
    if (!await authAdapter.prepareAuthenticatedRequest()) {
      await authAdapter.handleUnauthorized(rejectedAccessToken);
      throw new ApiError("登录状态已失效，请重新登录", 401, false, "UNAUTHENTICATED");
    }
    accessToken = authAdapter.getAccessToken();
    if (!accessToken) {
      throw new ApiError("登录状态已失效，请重新登录", 401, false, "UNAUTHENTICATED");
    }
  }

  const headers: Record<string, string> = {
    Accept: "application/json",
    "Content-Type": "application/json",
  };

  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }

  const response = await transportRequest(options, headers);
  const { statusCode } = response;

  if (statusCode >= 200 && statusCode < 300) {
    return response.data as TResponse;
  }

  const canRefresh =
    statusCode === 401 &&
    authenticated &&
    !retried &&
    (options.retryOnUnauthorized ?? true) &&
    authAdapter;

  if (canRefresh) {
    // 另一个并发请求可能已经用同一个旧 Token 完成刷新。此时直接用新
    // Token 重试，不能再次旋转 Refresh Token。
    const currentAccessToken = authAdapter!.getAccessToken();
    if (currentAccessToken && currentAccessToken !== accessToken) {
      return sendRequest<TResponse, TData>(options, true);
    }
    if (await authAdapter!.refreshOnce()) {
      return sendRequest<TResponse, TData>(options, true);
    }
  }

  if (statusCode === 401 && authenticated && authAdapter) {
    await authAdapter.handleUnauthorized(accessToken);
  }

  throw new ApiError(
    errorMessage(response.data, `请求失败（${statusCode}）`),
    statusCode,
    false,
    response.data && typeof response.data === "object"
      ? (response.data as ErrorPayload).code ?? ""
      : "",
  );
}

export function request<TResponse, TData = unknown>(
  options: RequestOptions<TData>,
): Promise<TResponse> {
  return sendRequest<TResponse, TData>(options, false);
}
