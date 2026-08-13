import { describe, expect, it, vi } from "vitest";

vi.mock("@/config/environment", () => ({
  DEV_AUTH_BYPASS_ENABLED: true,
  DEV_AUTH_ROLE: "doctor",
}));
vi.mock("@/api/auth", () => ({
  getCurrentIdentity: vi.fn(),
  phoneLogin: vi.fn(),
  refreshToken: vi.fn(),
  revokeToken: vi.fn(),
  wechatLogin: vi.fn(),
}));
vi.mock("@/utils/wechatCode", () => ({ getWechatLoginCode: vi.fn() }));

import {
  initializeDevelopmentSession,
  sessionState,
} from "@/stores/session";

describe("development session", () => {
  it("enters the staff app without requesting the backend", () => {
    expect(initializeDevelopmentSession()).toBe(true);
    expect(sessionState.status).toBe("authenticated");
    expect(sessionState.appVariant).toBe("staff");
    expect(sessionState.principal?.roles).toContain("department_doctor");
  });
});
