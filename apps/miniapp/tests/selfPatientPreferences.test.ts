import { beforeEach, describe, expect, it, vi } from "vitest";

describe("self patient preferences", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.unstubAllGlobals();
  });

  it("uses an enabled empty profile by default", async () => {
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn(() => undefined),
      setStorageSync: vi.fn(),
    });

    const { getSelfPatientPreferences } = await import(
      "@/utils/selfPatientPreferences"
    );

    expect(getSelfPatientPreferences()).toEqual({
      enabled: true,
      phoneMasked: "",
    });
  });

  it("persists only the masked phone and status preference", async () => {
    const setStorageSync = vi.fn();
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn(() => ({
        enabled: true,
        phoneMasked: "",
      })),
      setStorageSync,
    });

    const { saveSelfPatientPreferences } = await import(
      "@/utils/selfPatientPreferences"
    );

    expect(
      saveSelfPatientPreferences({
        enabled: false,
        phoneMasked: "138****5678",
      }),
    ).toEqual({
      enabled: false,
      phoneMasked: "138****5678",
    });
    expect(setStorageSync).toHaveBeenCalledWith(
      "hospital:self-patient-preferences",
      {
        enabled: false,
        phoneMasked: "138****5678",
      },
    );
  });
});
