import { beforeEach, describe, expect, it, vi } from "vitest";

describe("display profile", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.unstubAllGlobals();
  });

  it("keeps a selected temporary avatar only for the current runtime", async () => {
    const setStorageSync = vi.fn();
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn(() => undefined),
      setStorageSync,
    });

    const { saveDisplayProfile } = await import("@/utils/displayProfile");
    const temporaryAvatar =
      "http://127.0.0.1:20351/__tmp__/selected-avatar.jpeg";

    expect(
      saveDisplayProfile({
        avatarUrl: temporaryAvatar,
        nickname: "Polo",
      }),
    ).toEqual({
      avatarUrl: temporaryAvatar,
      nickname: "Polo",
    });
    expect(setStorageSync).toHaveBeenCalledWith("hospital:display-profile", {
      avatarUrl: "",
      nickname: "Polo",
    });
  });

  it("removes a stale temporary avatar loaded from storage", async () => {
    const setStorageSync = vi.fn();
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn(() => ({
        avatarUrl: "wxfile://tmp_selected-avatar.jpeg",
        nickname: "Polo",
      })),
      setStorageSync,
    });

    const { getDisplayProfile } = await import("@/utils/displayProfile");

    expect(getDisplayProfile()).toEqual({
      avatarUrl: "",
      nickname: "Polo",
    });
    expect(setStorageSync).toHaveBeenCalledWith("hospital:display-profile", {
      avatarUrl: "",
      nickname: "Polo",
    });
  });

  it("persists a remote OSS avatar URL for future sessions", async () => {
    const setStorageSync = vi.fn();
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn(() => undefined),
      setStorageSync,
    });

    const { saveDisplayProfile } = await import("@/utils/displayProfile");
    const ossAvatar = "https://hospital-avatar.oss-cn-shanghai.aliyuncs.com/users/polo.jpeg";

    saveDisplayProfile({
      avatarUrl: ossAvatar,
      nickname: "Polo",
    });

    expect(setStorageSync).toHaveBeenCalledWith("hospital:display-profile", {
      avatarUrl: ossAvatar,
      nickname: "Polo",
    });
  });
});
