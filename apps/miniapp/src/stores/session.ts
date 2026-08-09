import { reactive, readonly } from "vue";

import {
  getCurrentIdentity,
  refreshToken as requestTokenRefresh,
  revokeToken,
  wechatLogin,
} from "@/api/auth";
import { configureAuthAdapter } from "@/api/client";
import type {
  CurrentIdentityResponse,
  SessionTokenPair,
  SessionView,
  TokenResponse,
} from "@/types/auth";
import { getWechatLoginCode } from "@/utils/wechatCode";

const SESSION_STORAGE_KEY = "hospital:session";
const SESSION_STORAGE_VERSION = 1;
const EXPIRY_CLOCK_SKEW_MS = 5000;

interface StoredSession {
  version: number;
  tokens: SessionTokenPair;
  principal: CurrentIdentityResponse;
}

const state = reactive<SessionView>({
  status: "idle",
  principal: null,
});

let tokens: SessionTokenPair | null = null;
let refreshTask: Promise<boolean> | null = null;

export const sessionState = readonly(state);

function isNonEmptyString(value: unknown): value is string {
  return typeof value === "string" && value.trim().length > 0;
}

function isStoredSession(value: unknown): value is StoredSession {
  if (!value || typeof value !== "object") {
    return false;
  }

  const stored = value as Partial<StoredSession>;
  const storedTokens = stored.tokens as Partial<SessionTokenPair> | undefined;

  return (
    stored.version === SESSION_STORAGE_VERSION &&
    !!stored.principal &&
    isNonEmptyString(storedTokens?.accessToken) &&
    isNonEmptyString(storedTokens?.refreshToken) &&
    typeof storedTokens?.accessExpiresAt === "number" &&
    typeof storedTokens?.refreshExpiresAt === "number"
  );
}

function tokenPairFromResponse(response: TokenResponse): SessionTokenPair {
  const now = Date.now();

  if (
    !isNonEmptyString(response.access_token) ||
    !isNonEmptyString(response.refresh_token) ||
    !Number.isFinite(response.access_expires_in_seconds) ||
    !Number.isFinite(response.refresh_expires_in_seconds) ||
    response.access_expires_in_seconds <= 0 ||
    response.refresh_expires_in_seconds <= 0
  ) {
    throw new Error("后端返回的会话数据不完整");
  }

  return {
    accessToken: response.access_token,
    refreshToken: response.refresh_token,
    accessExpiresAt: now + response.access_expires_in_seconds * 1000,
    refreshExpiresAt: now + response.refresh_expires_in_seconds * 1000,
  };
}

function persistSession() {
  if (!tokens || !state.principal) {
    return;
  }

  const stored: StoredSession = {
    version: SESSION_STORAGE_VERSION,
    tokens,
    principal: state.principal,
  };

  try {
    uni.setStorageSync(SESSION_STORAGE_KEY, stored);
  } catch {
    // 内存会话仍可继续使用；Storage 失败不输出 Token，也不阻塞当前请求。
  }
}

function removePersistedSession() {
  try {
    uni.removeStorageSync(SESSION_STORAGE_KEY);
  } catch {
    // 本地清理失败不能导致 Token 出现在日志或错误提示中。
  }
}

function setGuestSession() {
  tokens = null;
  state.principal = null;
  state.status = "guest";
  removePersistedSession();
}

export function restoreSession() {
  try {
    const stored = uni.getStorageSync(SESSION_STORAGE_KEY) as unknown;
    if (!isStoredSession(stored) || stored.tokens.refreshExpiresAt <= Date.now()) {
      setGuestSession();
      return;
    }

    tokens = stored.tokens;
    state.principal = stored.principal;
    state.status = "authenticated";
  } catch {
    setGuestSession();
  }
}

export function getAccessToken(): string {
  return tokens?.accessToken ?? "";
}

function hasUsableAccessToken(): boolean {
  return !!tokens && tokens.accessExpiresAt - EXPIRY_CLOCK_SKEW_MS > Date.now();
}

async function performRefresh(): Promise<boolean> {
  const currentRefreshToken = tokens?.refreshToken;
  if (!currentRefreshToken || !tokens || tokens.refreshExpiresAt <= Date.now()) {
    setGuestSession();
    return false;
  }

  try {
    tokens = tokenPairFromResponse(await requestTokenRefresh(currentRefreshToken));
    state.principal = await getCurrentIdentity(false);
    state.status = "authenticated";
    persistSession();
    return true;
  } catch {
    setGuestSession();
    return false;
  }
}

export function refreshOnce(): Promise<boolean> {
  if (refreshTask) {
    return refreshTask;
  }

  refreshTask = performRefresh().finally(() => {
    refreshTask = null;
  });

  return refreshTask;
}

export async function initializeFromWechat(): Promise<boolean> {
  if (state.status === "authenticated" && hasUsableAccessToken() && state.principal) {
    return true;
  }

  if (tokens?.refreshToken && (await refreshOnce())) {
    return true;
  }

  state.status = "authenticating";

  try {
    const loginCode = await getWechatLoginCode();
    tokens = tokenPairFromResponse(await wechatLogin(loginCode));
    state.principal = await getCurrentIdentity(false);
    state.status = "authenticated";
    persistSession();
    return true;
  } catch {
    setGuestSession();
    return false;
  }
}

export async function logout(): Promise<void> {
  const currentRefreshToken = tokens?.refreshToken;
  setGuestSession();

  if (!currentRefreshToken) {
    return;
  }

  try {
    await revokeToken(currentRefreshToken);
  } catch {
    // 本地退出必须完成；服务端撤销失败由 Refresh Token 自身过期兜底。
  }
}

configureAuthAdapter({
  getAccessToken,
  refreshOnce,
});
