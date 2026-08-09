import { API_BASE_URL, API_REQUEST_TIMEOUT_MS } from "@/config/environment";

type RequestMethod = "GET" | "POST" | "PUT" | "DELETE";

interface RequestOptions<TData = unknown> {
  path: string;
  method?: RequestMethod;
  data?: TData;
  authenticated?: boolean;
  retryOnUnauthorized?: boolean;
}

interface AuthAdapter {
  getAccessToken: () => string;
  refreshOnce: () => Promise<boolean>;
}

interface ErrorPayload {
  message?: string;
  error?: string;
}

let authAdapter: AuthAdapter | undefined;

export class ApiError extends Error {
  readonly statusCode: number;
  readonly networkError: boolean;

  constructor(message: string, statusCode = 0, networkError = false) {
    super(message);
    this.name = "ApiError";
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

function sendRequest<TResponse, TData>(
  options: RequestOptions<TData>,
  retried: boolean,
): Promise<TResponse> {
  const authenticated = options.authenticated ?? false;
  const accessToken = authenticated ? authAdapter?.getAccessToken() ?? "" : "";

  if (authenticated && !accessToken) {
    return Promise.reject(new ApiError("需要登录后才能执行此操作", 401));
  }

  const headers: Record<string, string> = {
    Accept: "application/json",
    "Content-Type": "application/json",
  };

  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }

  return new Promise<TResponse>((resolve, reject) => {
    uni.request({
      url: `${API_BASE_URL}${options.path.startsWith("/") ? options.path : `/${options.path}`}`,
      method: options.method ?? "GET",
      data: options.data as UniApp.RequestOptions["data"],
      header: headers,
      timeout: API_REQUEST_TIMEOUT_MS,
      success: async (response) => {
        const statusCode = response.statusCode;

        if (statusCode >= 200 && statusCode < 300) {
          resolve(response.data as TResponse);
          return;
        }

        const canRefresh =
          statusCode === 401 &&
          authenticated &&
          !retried &&
          (options.retryOnUnauthorized ?? true) &&
          authAdapter;

        if (canRefresh) {
          try {
            if (await authAdapter!.refreshOnce()) {
              resolve(await sendRequest<TResponse, TData>(options, true));
              return;
            }
          } catch {
            // Refresh errors are converted to the original unauthorized response below.
          }
        }

        reject(
          new ApiError(
            errorMessage(response.data, `请求失败（${statusCode}）`),
            statusCode,
          ),
        );
      },
      fail: () => {
        reject(new ApiError("网络连接失败，请稍后重试", 0, true));
      },
    });
  });
}

export function request<TResponse, TData = unknown>(
  options: RequestOptions<TData>,
): Promise<TResponse> {
  return sendRequest<TResponse, TData>(options, false);
}
