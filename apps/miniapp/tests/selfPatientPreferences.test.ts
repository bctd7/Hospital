import { beforeEach, describe, expect, it, vi } from "vitest";

describe("self patient preferences", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.unstubAllGlobals();
  });

  it("uses an empty masked phone by default", async () => {
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn(() => undefined),
      setStorageSync: vi.fn(),
    });

    const { getSelfPatientPreferences } = await import(
      "@/utils/selfPatientPreferences"
    );

    expect(getSelfPatientPreferences()).toEqual({ phoneMasked: "" });
  });

  it("persists only the masked phone", async () => {
    const setStorageSync = vi.fn();
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn(() => ({
        phoneMasked: "",
      })),
      setStorageSync,
    });

    const { saveSelfPatientPreferences } = await import(
      "@/utils/selfPatientPreferences"
    );

    expect(
      saveSelfPatientPreferences({
        phoneMasked: "138****5678",
      }),
    ).toEqual({
      phoneMasked: "138****5678",
    });
    expect(setStorageSync).toHaveBeenCalledWith(
      "hospital:self-patient-preferences",
      {
        phoneMasked: "138****5678",
      },
    );
  });
});
