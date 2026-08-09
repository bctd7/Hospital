import { afterEach, describe, expect, it, vi } from "vitest";

import {
  hasIdentityPermission,
  openAppVariant,
  resolveAppVariant,
  resolveIdentityAppVariant,
} from "@/utils/appShell";
import type { CurrentIdentityResponse } from "@/types/auth";

function principal(
  roles: string[],
  permissions: string[] = [],
): CurrentIdentityResponse {
  return {
    account_id: "account-1",
    account_type: roles.length > 0 ? "staff" : "patient",
    status: "active",
    roles,
    permissions,
    authorization_version: 1,
  };
}

describe("app shell identity resolution", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("maps doctors and super administrators to the same staff app", () => {
    expect(resolveIdentityAppVariant(principal(["department_doctor"]))).toBe("staff");
    expect(resolveIdentityAppVariant(principal(["super_admin"]))).toBe("staff");
  });

  it("can fall back to patient when the staff app is disabled", () => {
    const doctor = principal(["department_doctor"]);

    expect(resolveAppVariant(doctor)).toBe("staff");
    expect(resolveAppVariant(doctor, false)).toBe("patient");
  });

  it("opens the same staff placeholder for staff identities", () => {
    const reLaunch = vi.fn();
    const switchTab = vi.fn();
    vi.stubGlobal("uni", { reLaunch, switchTab });

    openAppVariant("staff");

    expect(reLaunch).toHaveBeenCalledWith(
      expect.objectContaining({ url: "/pages/staff/index" }),
    );
    expect(switchTab).not.toHaveBeenCalled();
  });

  it("uses permissions rather than separate doctor and admin page trees", () => {
    const doctor = principal(["department_doctor"], ["appointment.read"]);
    const admin = principal(["super_admin"], ["identity.account.manage"]);

    expect(hasIdentityPermission(doctor, "appointment.read")).toBe(true);
    expect(hasIdentityPermission(doctor, "identity.account.manage")).toBe(false);
    expect(hasIdentityPermission(admin, "identity.account.manage")).toBe(true);
  });
});
