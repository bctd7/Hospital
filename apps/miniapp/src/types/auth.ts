export type SessionStatus = "idle" | "authenticating" | "authenticated" | "guest";

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  access_expires_in_seconds: number;
  refresh_expires_in_seconds: number;
}

export interface CurrentIdentityResponse {
  account_id: string;
  account_type: string;
  status: string;
  roles: string[];
  department_id?: string;
  permissions: string[];
  authorization_version: number;
}

export interface RevokeTokenResponse {
  revoked: boolean;
}

export interface SessionTokenPair {
  accessToken: string;
  refreshToken: string;
  accessExpiresAt: number;
  refreshExpiresAt: number;
}

export interface SessionView {
  status: SessionStatus;
  principal: CurrentIdentityResponse | null;
}
