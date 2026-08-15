import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import {
  formatEstimatedDuration,
  canReadAppointmentManagement,
  itemWindowTimeValid,
  roomWindowTimeValid,
  windowFitsSession,
} from "@/utils/appointmentManagement";

describe("appointment management view rules", () => {
  it("formats estimated examination duration for shared booking displays", () => {
    expect(formatEstimatedDuration(20)).toBe("预计用时约 20 分钟");
    expect(formatEstimatedDuration(90)).toBe("预计用时约 1 小时 30 分钟");
    expect(formatEstimatedDuration(120)).toBe("预计用时约 2 小时");
  });
  it("requires both a staff role and appointment read permission", () => {
    expect(canReadAppointmentManagement({ roles: ["department_doctor"], permissions: ["appointment.read"] })).toBe(true);
    expect(canReadAppointmentManagement({ roles: [], permissions: ["appointment.read"] })).toBe(false);
    expect(canReadAppointmentManagement({ roles: ["department_doctor"], permissions: [] })).toBe(false);
  });

  it("only exposes examination start after staff has called the patient", () => {
    for (const page of ["bookings.vue", "booking-detail.vue"]) {
      const source = readFileSync(
        new URL(`../src/pages/admin/appointment/${page}`, import.meta.url),
        "utf8",
      );
      expect(source).toContain("status === \"called\"");
      expect(source).toContain("开始检查");
    }
  });

  it("validates the two different weekly time models", () => {
    expect(roomWindowTimeValid("08:00", "12:00")).toBe(true);
    expect(roomWindowTimeValid("12:00", "08:00")).toBe(false);
    expect(itemWindowTimeValid("09:00", "11:30", "12:00")).toBe(true);
    expect(itemWindowTimeValid("09:00", "12:00", "12:00")).toBe(false);
    expect(windowFitsSession("morning", "09:00", "12:00")).toBe(true);
    expect(windowFitsSession("morning", "13:00", "18:00")).toBe(false);
    expect(windowFitsSession("afternoon", "13:00", "18:00")).toBe(true);
    expect(windowFitsSession("afternoon", "09:00", "12:00")).toBe(false);
  });

  it("resets weekly-window times when the selected session changes", () => {
    const itemDetail = readFileSync(
      new URL("../src/pages/admin/appointment/item-detail.vue", import.meta.url),
      "utf8",
    );
    const roomDetail = readFileSync(
      new URL("../src/pages/admin/appointment/room-detail.vue", import.meta.url),
      "utf8",
    );

    for (const source of [itemDetail, roomDetail]) {
      expect(source).toContain('"09:00"');
      expect(source).toContain('"12:00"');
      expect(source).toContain('"13:00"');
      expect(source).toContain('"18:00"');
      expect(source).toContain("时段与时间不一致");
    }
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

  it("explains room-window conflicts when saving an item window", () => {
    const source = readFileSync(
      new URL("../src/pages/admin/appointment/item-detail.vue", import.meta.url),
      "utf8",
    );

    expect(source).toContain("ITEM_ROOM_WINDOW_CONFLICT");
    expect(source).toContain("项目时间超出房间开放范围");
    expect(source).toContain("项目预约时间必须完整落在所有已关联房间");
  });

  it("keeps weekly-window editors open after native picker confirmation", () => {
    const itemDetail = readFileSync(
      new URL("../src/pages/admin/appointment/item-detail.vue", import.meta.url),
      "utf8",
    );
    const roomDetail = readFileSync(
      new URL("../src/pages/admin/appointment/room-detail.vue", import.meta.url),
      "utf8",
    );

    expect(itemDetail).not.toContain('class="dialog-mask" @tap.self');
    expect(roomDetail).not.toContain('class="dialog-mask" @tap.self');
    expect(itemDetail).toContain('@tap="editorVisible = false">取消');
    expect(roomDetail).toContain('@tap="editorVisible = false">取消');
  });
});
