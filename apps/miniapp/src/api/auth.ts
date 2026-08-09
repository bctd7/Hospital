import { request } from "@/api/client";
import type {
	CurrentIdentityResponse,
	PhoneBindingResponse,
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

export function setMyPhone(phone: string): Promise<PhoneBindingResponse> {
  return request<PhoneBindingResponse, { phone: string }>({
    path: `${AUTH_PREFIX}/me/phone`,
    method: "PUT",
    data: { phone },
    authenticated: true,
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
