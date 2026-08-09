import { afterEach, describe, expect, it, vi } from "vitest";

describe("CloudBase AnyService adapter", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    vi.resetModules();
  });

  it("initializes the selected environment and adds AnyService routing headers", async () => {
    vi.stubEnv("VITE_API_TRANSPORT", "cloudbase");
    vi.stubEnv("VITE_CLOUDBASE_ENV_ID", "hospital-test-environment");
    vi.stubEnv("VITE_ANYSERVICE_NAME", "hospitalapi");

    const init = vi.fn();
    const callContainer = vi.fn(async () => ({
      statusCode: 200,
      data: { status: "ok" },
    }));
    vi.stubGlobal("wx", { cloud: { init, callContainer } });

    const { callAnyService } = await import("@/platform/cloudbase");
    await expect(
      callAnyService({
        path: "/api/v1/health",
        method: "GET",
        header: { Accept: "application/json" },
      }),
    ).resolves.toEqual({ statusCode: 200, data: { status: "ok" } });

    expect(init).toHaveBeenCalledOnce();
    expect(init).toHaveBeenCalledWith({
      env: "hospital-test-environment",
      traceUser: true,
    });
    expect(callContainer).toHaveBeenCalledWith(
      expect.objectContaining({
        path: "/api/v1/health",
        header: expect.objectContaining({
          "X-WX-SERVICE": "tcbanyservice",
          "X-AnyService-Name": "hospitalapi",
        }),
      }),
    );
  });
});
