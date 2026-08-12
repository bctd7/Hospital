import { afterEach, describe, expect, it, vi } from "vitest";

import { configureAuthAdapter, request } from "@/api/client";

describe("api client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("refreshes once and retries an unauthorized request with the new token", async () => {
    let accessToken = "old-access-token";
    const refreshOnce = vi.fn(async () => {
      accessToken = "new-access-token";
      return true;
    });
    configureAuthAdapter({
      getAccessToken: () => accessToken,
      refreshOnce,
      handleUnauthorized: vi.fn(),
    });

    const authorizationHeaders: string[] = [];
    let requestCount = 0;
    const requestMock = vi.fn((options) => {
      requestCount += 1;
      authorizationHeaders.push(options.header.Authorization);
      options.success(
        requestCount === 1
          ? { statusCode: 401, data: "authentication required" }
          : { statusCode: 200, data: { ok: true } },
      );
    });
    vi.stubGlobal("uni", { request: requestMock });

    await expect(
      request<{ ok: boolean }>({ path: "/api/v1/protected", authenticated: true }),
    ).resolves.toEqual({ ok: true });
    expect(refreshOnce).toHaveBeenCalledTimes(1);
    expect(authorizationHeaders).toEqual([
      "Bearer old-access-token",
      "Bearer new-access-token",
    ]);
  });

  it("does not send an authenticated request without an access token", async () => {
    const requestMock = vi.fn();
    vi.stubGlobal("uni", { request: requestMock });
    configureAuthAdapter({
      getAccessToken: () => "",
      refreshOnce: vi.fn(),
      handleUnauthorized: vi.fn(),
    });

    await expect(
      request({ path: "/api/v1/protected", authenticated: true }),
    ).rejects.toMatchObject({ statusCode: 401 });
    expect(requestMock).not.toHaveBeenCalled();
  });

  it("forces reauthentication when refresh cannot recover an unauthorized request", async () => {
    const handleUnauthorized = vi.fn(async () => undefined);
    configureAuthAdapter({
      getAccessToken: () => "rejected-access-token",
      refreshOnce: vi.fn(async () => false),
      handleUnauthorized,
    });
    vi.stubGlobal("uni", {
      request: vi.fn((options) => {
        options.success({ statusCode: 401, data: "authentication required" });
      }),
    });

    await expect(
      request({ path: "/api/v1/protected", authenticated: true }),
    ).rejects.toMatchObject({ statusCode: 401 });
    expect(handleUnauthorized).toHaveBeenCalledTimes(1);
    expect(handleUnauthorized).toHaveBeenCalledWith("rejected-access-token");
  });
});
