import { describe, expect, it } from "vitest";

import {
  resolveDevAuthBypassEnabled,
  resolveDevAuthRole,
} from "@/config/environment";

describe("development auth environment", () => {
  it("allows bypass only in development mode", () => {
    expect(resolveDevAuthBypassEnabled(true, "true")).toBe(true);
    expect(resolveDevAuthBypassEnabled(false, "true")).toBe(false);
    expect(resolveDevAuthBypassEnabled(true, "false")).toBe(false);
  });

  it("normalizes supported mock identities", () => {
    expect(resolveDevAuthRole("doctor")).toBe("doctor");
    expect(resolveDevAuthRole("ADMIN")).toBe("admin");
    expect(resolveDevAuthRole("unknown")).toBe("patient");
  });

});
