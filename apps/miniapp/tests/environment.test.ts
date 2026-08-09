import { describe, expect, it } from "vitest";

import { resolveStaffDataSource } from "../src/config/environment";

describe("resolveStaffDataSource", () => {
  it("honors an explicit mock source in a production build", () => {
    expect(resolveStaffDataSource("production", "mock", false)).toBe("mock");
  });

  it("uses HTTP by default in a production build", () => {
    expect(resolveStaffDataSource("production", undefined, false)).toBe("http");
  });

  it("keeps development on mock unless HTTP is explicit", () => {
    expect(resolveStaffDataSource("development", undefined, true)).toBe("mock");
    expect(resolveStaffDataSource("development", "http", true)).toBe("http");
  });
});
