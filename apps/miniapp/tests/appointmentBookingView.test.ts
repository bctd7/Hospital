import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

describe("patient appointment booking view", () => {
  it("uses a progressive two-column workspace for the patient hierarchy", () => {
    const source = readFileSync(
      new URL("../src/pages/appointment/create/index.vue", import.meta.url),
      "utf8",
    );

    expect(source).toContain("workspace-track--item-rooms");
    expect(source).toContain("width:150%");
    expect(source).toContain("translateX(-33.333333%)");
    expect(source).toContain("返回科室");
    expect(source).toContain("园区：");
    expect(source).toContain("楼栋：");
    expect(source).toContain("楼层：");
    expect(source).toContain("房间号：");
    expect(source).toContain("prefers-reduced-motion");
    expect(source).not.toContain("grid-template-columns:190rpx 238rpx minmax(0,1fr)");
  });
});
