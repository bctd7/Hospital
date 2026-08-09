import { afterEach, describe, expect, it, vi } from "vitest";

import { getWechatLoginCode } from "@/utils/wechatCode";

describe("getWechatLoginCode", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it("returns the one-time code without writing it to storage", async () => {
    const setStorageSync = vi.fn();
    vi.stubGlobal("uni", {
      login: vi.fn((options) => options.success({ code: "one-time-code" })),
      setStorageSync,
    });

    await expect(getWechatLoginCode()).resolves.toBe("one-time-code");
    expect(setStorageSync).not.toHaveBeenCalled();
  });

  it("rejects when WeChat does not return a code", async () => {
    vi.stubGlobal("uni", {
      login: vi.fn((options) => options.success({ code: "" })),
    });

    await expect(getWechatLoginCode()).rejects.toThrow("微信未返回有效登录凭证");
  });
});
