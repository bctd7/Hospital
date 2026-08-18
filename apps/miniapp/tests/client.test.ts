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

  it("does not refresh again when a late 401 belongs to the previous access token", async () => {
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

    const oldRequests: Array<{ success: (value: unknown) => void }> = [];
    vi.stubGlobal("uni", {
      request: vi.fn((options) => {
        if (options.header.Authorization === "Bearer new-access-token") {
          options.success({ statusCode: 200, data: { ok: true } });
          return;
        }
        oldRequests.push(options);
      }),
    });

    const first = request<{ ok: boolean }>({ path: "/api/v1/first", authenticated: true });
    const second = request<{ ok: boolean }>({ path: "/api/v1/second", authenticated: true });
    expect(oldRequests).toHaveLength(2);

    oldRequests[0]!.success({ statusCode: 401, data: "authentication required" });
    await expect(first).resolves.toEqual({ ok: true });
    oldRequests[1]!.success({ statusCode: 401, data: "authentication required" });
    await expect(second).resolves.toEqual({ ok: true });
    expect(refreshOnce).toHaveBeenCalledTimes(1);
  });

  it("validates a restored session before sending the first protected request", async () => {
    let accessToken = "stored-access-token";
    const prepareAuthenticatedRequest = vi.fn(async () => {
      accessToken = "validated-access-token";
      return true;
    });
    configureAuthAdapter({
      getAccessToken: () => accessToken,
      prepareAuthenticatedRequest,
      refreshOnce: vi.fn(),
      handleUnauthorized: vi.fn(),
    });
    const requestMock = vi.fn((options) => {
      options.success({ statusCode: 200, data: { ok: true } });
    });
    vi.stubGlobal("uni", { request: requestMock });

    await expect(request({ path: "/api/v1/protected", authenticated: true })).resolves.toEqual({ ok: true });
    expect(prepareAuthenticatedRequest).toHaveBeenCalledTimes(1);
    expect(requestMock.mock.calls[0]?.[0].header.Authorization).toBe("Bearer validated-access-token");
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

  it("does not clear the session when refresh only fails because of the network", async () => {
    const refreshFailure = new Error("network unavailable");
    const handleUnauthorized = vi.fn(async () => undefined);
    configureAuthAdapter({
      getAccessToken: () => "expired-access-token",
      refreshOnce: vi.fn(async () => { throw refreshFailure; }),
      handleUnauthorized,
    });
    vi.stubGlobal("uni", {
      request: vi.fn((options) => options.success({ statusCode: 401, data: "authentication required" })),
    });

    await expect(request({ path: "/api/v1/protected", authenticated: true })).rejects.toBe(refreshFailure);
    expect(handleUnauthorized).not.toHaveBeenCalled();
  });

  it("preserves the stable backend error code", async () => {
    vi.stubGlobal("uni", {
      request: vi.fn((options) => {
        options.success({
          statusCode: 409,
          data: {
            code: "PATIENT_ITEM_SESSION_OCCUPIED",
            message: "patient already has an active booking for this item in this date and session",
          },
        });
      }),
    });

    await expect(
      request({ path: "/api/v1/patient/appointment/bookings", method: "POST" }),
    ).rejects.toMatchObject({
      code: "PATIENT_ITEM_SESSION_OCCUPIED",
      statusCode: 409,
    });
  });
});
