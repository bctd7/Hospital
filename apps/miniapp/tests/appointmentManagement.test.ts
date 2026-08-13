import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import { patientAppointmentMockAdapter } from "@/mocks/appointmentBooking";
import {
  canReadAppointmentManagement,
  itemWindowTimeValid,
  roomWindowTimeValid,
} from "@/utils/appointmentManagement";

describe("appointment management view rules", () => {
  it("requires both a staff role and appointment read permission", () => {
    expect(canReadAppointmentManagement({ roles: ["department_doctor"], permissions: ["appointment.read"] })).toBe(true);
    expect(canReadAppointmentManagement({ roles: [], permissions: ["appointment.read"] })).toBe(false);
    expect(canReadAppointmentManagement({ roles: ["department_doctor"], permissions: [] })).toBe(false);
  });

  it("validates the two different weekly time models", () => {
    expect(roomWindowTimeValid("08:00", "12:00")).toBe(true);
    expect(roomWindowTimeValid("12:00", "08:00")).toBe(false);
    expect(itemWindowTimeValid("09:00", "11:30", "12:00")).toBe(true);
    expect(itemWindowTimeValid("09:00", "12:00", "12:00")).toBe(false);
  });

  it("keeps doctors out of the patient mock model", async () => {
    const departments = await patientAppointmentMockAdapter.listDepartments();
    const serialized = JSON.stringify(departments);
    expect(serialized).not.toContain("doctor");
    expect(departments[0]?.items[0]?.rooms.length).toBeGreaterThan(0);
  });

  it("keeps the admin resource workspace progressive and free of removed actions", () => {
    const source = readFileSync(
      new URL("../src/pages/admin/appointment/index.vue", import.meta.url),
      "utf8",
    );

    expect(source).toContain("workspace-track--room-items");
    expect(source).toContain("width: 150%");
    expect(source).toContain("translateX(-33.333333%)");
    expect(source).toContain("返回科室");
    expect(source).toContain("园区：");
    expect(source).toContain("楼栋：");
    expect(source).toContain("楼层：");
    expect(source).toContain("房间号：");
    expect(source).toContain("prefers-reduced-motion");
    expect(source).not.toContain("管理当前房间");
    expect(source).not.toContain("查看停用资源");
    expect(source).not.toContain("停用房间");
    expect(source).not.toContain("恢复房间");
    expect(source).not.toContain("grid-template-columns: 190rpx 220rpx");
  });
});
