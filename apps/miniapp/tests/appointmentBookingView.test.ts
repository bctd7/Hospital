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
    expect(source).toContain("@tap=\"switchCampus\"");
    expect(source).toContain("selectedCampusId");
    expect(source).toContain("campusDepartments");
    expect(source).toContain("搜索当前院区科室");
    expect(source).toContain("patientAppointmentApi.listItemRooms(id)");
    expect(source).toContain("该房间已配置，但本周暂无可预约时段");
    expect(source).toContain("room-card--unavailable");
    expect(source).toContain("本周无可预约时段");
    expect(source).not.toContain("options.value.forEach((value) => values.set(value.roomId, value))");
    expect(source).not.toContain(">院区可切换</text>");
    expect(source).not.toContain("grid-template-columns:190rpx 238rpx minmax(0,1fr)");
  });

  it("refreshes the patient booking state after a call deadline expires", () => {
    const source = readFileSync(
      new URL("../src/pages/profile/appointments/index.vue", import.meta.url),
      "utf8",
    );
    expect(source).toContain("setInterval(tickBookingClock, 1000)");
    expect(source).toContain("hasExpiredCall");
    expect(source).toContain("void loadBookings()");
  });
});
