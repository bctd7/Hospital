import { request } from "@/api/client";
import type {
	CurrentIdentityResponse,
	RevokeTokenResponse,
	SendPhoneLoginCodeResponse,
	TokenResponse,
} from "@/types/auth";

const AUTH_PREFIX = "/api/v1/auth";

export function sendPhoneLoginCode(phone: string): Promise<SendPhoneLoginCodeResponse> {
  return request<SendPhoneLoginCodeResponse, { phone: string }>({
    path: `${AUTH_PREFIX}/phone/code`,
    method: "POST",
    data: { phone },
  });
}

export function phoneLogin(phone: string, verificationCode: string): Promise<TokenResponse> {
  return request<TokenResponse, { phone: string; verification_code: string }>({
    path: `${AUTH_PREFIX}/phone/login`,
    method: "POST",
    data: { phone, verification_code: verificationCode },
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
