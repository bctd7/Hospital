import { describe, expect, it } from "vitest";

import { developmentPrincipal } from "@/mocks/developmentIdentity";

describe("development identities", () => {
  it("creates a patient without staff permissions", () => {
    const principal = developmentPrincipal("patient");
    expect(principal.account_type).toBe("patient");
    expect(principal.roles).toEqual([]);
    expect(principal.permissions).toEqual([]);
  });

  it("creates a department-scoped doctor", () => {
    const principal = developmentPrincipal("doctor");
    expect(principal.roles).toContain("department_doctor");
    expect(principal.department_id).toBeTruthy();
    expect(principal.permissions).toContain("appointment.read");
  });

  it("creates an administrator with identity management permissions", () => {
    const principal = developmentPrincipal("admin");
    expect(principal.roles).toContain("super_admin");
    expect(principal.permissions).toContain("identity.account.manage");
  });
});
