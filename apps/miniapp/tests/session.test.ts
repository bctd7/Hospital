import { beforeEach, describe, expect, it, vi } from "vitest";

const authMocks = vi.hoisted(() => ({
  getCurrentIdentity: vi.fn(),
  phoneLogin: vi.fn(),
  refreshToken: vi.fn(),
  revokeToken: vi.fn(),
}));

vi.mock("@/api/auth", () => authMocks);

import {
  availableAppVariants,
  initializeFromPhone,
  handleUnauthorized,
  logout,
  prepareAuthenticatedRequest,
  refreshOnce,
  restoreSession,
  sessionState,
  setAppVariant,
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
      reLaunch: vi.fn((options) => options.complete?.()),
      setStorageSync: vi.fn((key: string, value: unknown) => storage.set(key, value)),
      showToast: vi.fn(),
    });
    await logout();
    vi.clearAllMocks();
  });

  it("uses a verified phone code to create the Hospital session", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    await expect(initializeFromPhone("13800138000", "123456")).resolves.toBe(true);

    expect(authMocks.phoneLogin).toHaveBeenCalledWith("13800138000", "123456");
    expect(sessionState.status).toBe("authenticated");
    expect(sessionState.principal).toEqual(principal);
  });

  it("falls back to guest when phone authentication fails", async () => {
    authMocks.phoneLogin.mockRejectedValue(new Error("invalid verification code"));

    await expect(initializeFromPhone("13800138000", "000000")).resolves.toBe(false);
    expect(sessionState.status).toBe("guest");
    expect(storage.has("hospital:session")).toBe(false);
  });

  it("selects the shared staff placeholder for a doctor", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(doctorPrincipal);

    await initializeFromPhone("13800138000", "123456");

    expect(sessionState.appVariant).toBe("staff");
  });

  it("allows a staff identity to switch to patient and persists the choice", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(doctorPrincipal);
    await initializeFromPhone("13800138000", "123456");

    expect(availableAppVariants()).toEqual(["patient", "staff"]);
    expect(setAppVariant("patient")).toBe(true);
    expect(sessionState.appVariant).toBe("patient");

    const persisted = storage.get("hospital:session") as { appVariant?: string };
    expect(persisted.appVariant).toBe("patient");
  });

  it("does not allow a patient identity to enter the staff app", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);
    await initializeFromPhone("13800138000", "123456");

    expect(availableAppVariants()).toEqual(["patient"]);
    expect(setAppVariant("staff")).toBe(false);
    expect(sessionState.appVariant).toBe("patient");
  });

  it("derives the app variant again when restoring a session", () => {
    storage.set("hospital:session", {
      version: 1,
      tokens: {
        accessToken: "access-token",
        refreshToken: "refresh-token",
        accessExpiresAt: Date.now() + 60000,
        refreshExpiresAt: Date.now() + 120000,
      },
      principal: doctorPrincipal,
      // 旧版会话字段不得绕过当前角色与应用版本校验。
      activeMode: "doctor",
    });

    restoreSession();

    expect(sessionState.appVariant).toBe("staff");
  });

  it("recomputes the app variant after refreshing the principal", async () => {
    storage.set("hospital:session", {
      version: 1,
      tokens: {
        accessToken: "expired-access-token",
        refreshToken: "current-refresh-token",
        accessExpiresAt: Date.now() - 1000,
        refreshExpiresAt: Date.now() + 60000,
      },
      principal: doctorPrincipal,
    });
    restoreSession();
    authMocks.refreshToken.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    await expect(refreshOnce()).resolves.toBe(true);

    expect(sessionState.appVariant).toBe("patient");
  });

  it("resets the app variant when the user logs out", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(doctorPrincipal);
    await initializeFromPhone("13800138000", "123456");

    await logout();

    expect(sessionState.status).toBe("guest");
    expect(sessionState.appVariant).toBe("patient");
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

  it("validates one restored session only once before concurrent business requests", async () => {
    storage.set("hospital:session", {
      version: 1,
      tokens: {
        accessToken: "stored-access-token",
        refreshToken: "stored-refresh-token",
        accessExpiresAt: Date.now() + 60000,
        refreshExpiresAt: Date.now() + 120000,
      },
      principal,
    });
    restoreSession();
    let finishRefresh: ((value: typeof tokenResponse) => void) | undefined;
    authMocks.refreshToken.mockReturnValue(new Promise((resolve) => { finishRefresh = resolve; }));
    authMocks.getCurrentIdentity.mockResolvedValue(principal);

    const first = prepareAuthenticatedRequest();
    const second = prepareAuthenticatedRequest();
    expect(first).toBe(second);
    expect(authMocks.refreshToken).toHaveBeenCalledTimes(1);
    finishRefresh?.(tokenResponse);
    await expect(Promise.all([first, second])).resolves.toEqual([true, true]);
    expect(authMocks.refreshToken).toHaveBeenCalledTimes(1);
  });

  it("clears a restored session when its refresh token is no longer valid", async () => {
    storage.set("hospital:session", {
      version: 1,
      tokens: {
        accessToken: "stored-access-token",
        refreshToken: "invalid-refresh-token",
        accessExpiresAt: Date.now() + 60000,
        refreshExpiresAt: Date.now() + 120000,
      },
      principal,
    });
    restoreSession();
    authMocks.refreshToken.mockRejectedValue({ statusCode: 401, code: "UNAUTHENTICATED" });

    await expect(prepareAuthenticatedRequest()).resolves.toBe(false);
    expect(sessionState.status).toBe("guest");
    expect(storage.has("hospital:session")).toBe(false);
  });

  it("clears the rejected session and returns to entry when identity changes", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(doctorPrincipal);
    await initializeFromPhone("13800138000", "123456");

    await handleUnauthorized("access-token");

    expect(sessionState.status).toBe("guest");
    expect(sessionState.principal).toBeNull();
    expect(storage.has("hospital:session")).toBe(false);
    expect(uni.reLaunch).toHaveBeenCalledWith(
      expect.objectContaining({ url: "/pages/entry/index" }),
    );
    expect(uni.showToast).toHaveBeenCalledWith(
      expect.objectContaining({ title: "身份已变更，请重新登录" }),
    );
  });

  it("does not let a late unauthorized response clear a newer login", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);
    await initializeFromPhone("13800138000", "123456");

    await handleUnauthorized("old-access-token");

    expect(sessionState.status).toBe("authenticated");
    expect(sessionState.principal).toEqual(principal);
    expect(uni.reLaunch).not.toHaveBeenCalled();
  });

  it("shares one forced reauthentication across concurrent unauthorized responses", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(principal);
    await initializeFromPhone("13800138000", "123456");

    let finishReLaunch: (() => void) | undefined;
    vi.mocked(uni.reLaunch).mockImplementation((options) => {
      finishReLaunch = () => options.complete?.({ errMsg: "reLaunch:ok" });
      return undefined as never;
    });

    const first = handleUnauthorized("access-token");
    const second = handleUnauthorized("access-token");

    expect(first).toBe(second);
    expect(uni.reLaunch).toHaveBeenCalledTimes(1);
    finishReLaunch?.();
    await Promise.all([first, second]);
    expect(uni.showToast).toHaveBeenCalledTimes(1);
  });

  it("allows phone login again after forced reauthentication", async () => {
    authMocks.phoneLogin.mockResolvedValue(tokenResponse);
    authMocks.getCurrentIdentity.mockResolvedValue(doctorPrincipal);
    await initializeFromPhone("13800138000", "123456");
    await handleUnauthorized("access-token");

    authMocks.getCurrentIdentity.mockResolvedValue(principal);
    await expect(initializeFromPhone("13800138000", "654321")).resolves.toBe(true);

    expect(sessionState.status).toBe("authenticated");
    expect(sessionState.principal).toEqual(principal);
    expect(authMocks.phoneLogin).toHaveBeenLastCalledWith("13800138000", "654321");
  });
});
