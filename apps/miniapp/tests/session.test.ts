import { beforeEach, describe, expect, it, vi } from "vitest";

const authMocks = vi.hoisted(() => ({
  getCurrentIdentity: vi.fn(),
  phoneLogin: vi.fn(),
  refreshToken: vi.fn(),
  revokeToken: vi.fn(),
  wechatLogin: vi.fn(),
}));

const codeMocks = vi.hoisted(() => ({
  getWechatLoginCode: vi.fn(),
}));

vi.mock("@/api/auth", () => authMocks);
vi.mock("@/utils/wechatCode", () => codeMocks);

import {
  availableAppModes,
  initializeFromWechat,
  initializeFromPhone,
  logout,
  refreshOnce,
  restoreSession,
  setActiveMode,
  sessionState,
} from "@/stores/session";

const principal = {
  account_id: "account-1",
  account_type: "patient",
  status: "active",
  roles: ["patient"],
  permissions: [],
  authorization_version: 1,
};

const doctorPrincipal = {
  ...principal,
  account_type: "staff",
  roles: ["department_doctor"],
};

const tokenResponse = {
  access_token: "access-token",
  refresh_token: "refresh-token",
  access_expires_in_seconds: 900,
  refresh_expires_in_seconds: 2592000,
};

let storage: Map<string, unknown>;

describe("session store", () => {
  beforeEach(async () => {
    storage = new Map();
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn((key: string) => storage.get(key)),
      removeStorageSync: vi.fn((key: string) => storage.delete(key)),
      setStorageSync: vi.fn((key: string, value: unknown) => storage.set(key, value)),
    });
    await logout();
    vi.clearAllMocks();
  });

  it("exchanges the WeChat code for Hospital tokens and a principal", async () => {
    codeMocks.getWechatLoginCode.mockResolvedValue("one-time-code");
    authMocks.wechatLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    await expect(initializeFromWechat()).resolves.toBe(true);
    expect(authMocks.wechatLogin).toHaveBeenCalledWith("one-time-code");
    expect(sessionState.status).toBe("authenticated");
    expect(sessionState.principal).toEqual(principal);

    const persisted = JSON.stringify(storage.get("hospital:session"));
    expect(persisted).not.toContain("one-time-code");
    expect(persisted.toLowerCase()).not.toContain("openid");
  });

  it("uses a verified phone code to create the Hospital session", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    await expect(initializeFromPhone("13800138000", "123456")).resolves.toBe(true);

    expect(authMocks.phoneLogin).toHaveBeenCalledWith("13800138000", "123456");
    expect(sessionState.status).toBe("authenticated");
    expect(sessionState.principal).toEqual(principal);
  });

  it("falls back to guest and removes incomplete session data", async () => {
    codeMocks.getWechatLoginCode.mockRejectedValue(new Error("wechat unavailable"));

    await expect(initializeFromWechat()).resolves.toBe(false);
    expect(sessionState.status).toBe("guest");
    expect(storage.has("hospital:session")).toBe(false);
  });

  it("does not allow a patient account to select doctor mode", async () => {
    codeMocks.getWechatLoginCode.mockResolvedValue("one-time-code");
    authMocks.wechatLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    await initializeFromWechat();

    expect(availableAppModes()).toEqual(["patient"]);
    expect(setActiveMode("doctor")).toBe(false);
    expect(sessionState.activeMode).toBe("patient");
  });

  it("persists doctor mode only for an account with the doctor role", async () => {
    codeMocks.getWechatLoginCode.mockResolvedValue("one-time-code");
    authMocks.wechatLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(doctorPrincipal);

    await initializeFromWechat();

    expect(availableAppModes()).toEqual(["patient", "doctor"]);
    expect(setActiveMode("doctor")).toBe(true);
    expect(sessionState.activeMode).toBe("doctor");

    const persisted = storage.get("hospital:session") as { activeMode?: string };
    expect(persisted.activeMode).toBe("doctor");
  });

  it("falls back to patient mode when a stored doctor role is no longer present", () => {
    storage.set("hospital:session", {
      version: 1,
      tokens: {
        accessToken: "access-token",
        refreshToken: "refresh-token",
        accessExpiresAt: Date.now() + 60000,
        refreshExpiresAt: Date.now() + 120000,
      },
      principal,
      activeMode: "doctor",
    });

    restoreSession();

    expect(sessionState.activeMode).toBe("patient");
  });

  it("falls back to patient mode when refresh returns a principal without the doctor role", async () => {
    storage.set("hospital:session", {
      version: 1,
      tokens: {
        accessToken: "expired-access-token",
        refreshToken: "current-refresh-token",
        accessExpiresAt: Date.now() - 1000,
        refreshExpiresAt: Date.now() + 60000,
      },
      principal: doctorPrincipal,
      activeMode: "doctor",
    });
    restoreSession();
    authMocks.refreshToken.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    await expect(refreshOnce()).resolves.toBe(true);

    expect(sessionState.activeMode).toBe("patient");
    const persisted = storage.get("hospital:session") as { activeMode?: string };
    expect(persisted.activeMode).toBe("patient");
  });

  it("resets doctor mode when the user logs out", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(doctorPrincipal);
    await initializeFromPhone("13800138000", "123456");
    expect(setActiveMode("doctor")).toBe(true);

    await logout();

    expect(sessionState.status).toBe("guest");
    expect(sessionState.activeMode).toBe("patient");
    expect(storage.has("hospital:session")).toBe(false);
  });

  it("shares one refresh request across concurrent callers", async () => {
    storage.set("hospital:session", {
      version: 1,
      tokens: {
        accessToken: "expired-access-token",
        refreshToken: "current-refresh-token",
        accessExpiresAt: Date.now() - 1000,
        refreshExpiresAt: Date.now() + 60000,
      },
      principal,
    });
    restoreSession();

    let completeRefresh: ((value: typeof tokenResponse) => void) | undefined;
    authMocks.refreshToken.mockReturnValue(
      new Promise((resolve) => {
        completeRefresh = resolve;
      }),
    );
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    const first = refreshOnce();
    const second = refreshOnce();
    expect(first).toBe(second);
    expect(authMocks.refreshToken).toHaveBeenCalledTimes(1);

    completeRefresh?.(tokenResponse);
    await expect(Promise.all([first, second])).resolves.toEqual([true, true]);
    expect(authMocks.refreshToken).toHaveBeenCalledTimes(1);
  });
});
