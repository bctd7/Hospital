import { request } from "@/api/client";
import type {
  CurrentIdentityResponse,
  RevokeTokenResponse,
  TokenResponse,
} from "@/types/auth";

const AUTH_PREFIX = "/api/v1/auth";

export function wechatLogin(loginCode: string): Promise<TokenResponse> {
  return request<TokenResponse, { login_code: string }>({
    path: `${AUTH_PREFIX}/wechat/login`,
    method: "POST",
    data: { login_code: loginCode },
  });
}

export function getCurrentIdentity(retryOnUnauthorized = true): Promise<CurrentIdentityResponse> {
  return request<CurrentIdentityResponse>({
    path: `${AUTH_PREFIX}/me`,
    authenticated: true,
    retryOnUnauthorized,
  });
}

export function refreshToken(refreshTokenValue: string): Promise<TokenResponse> {
  return request<TokenResponse, { refresh_token: string }>({
    path: `${AUTH_PREFIX}/token/refresh`,
    method: "POST",
    data: { refresh_token: refreshTokenValue },
  });
}

export function revokeToken(refreshTokenValue: string): Promise<RevokeTokenResponse> {
  return request<RevokeTokenResponse, { refresh_token: string }>({
    path: `${AUTH_PREFIX}/token/revoke`,
    method: "POST",
    data: { refresh_token: refreshTokenValue },
  });
}
