import { describe, expect, it } from "vitest";

import type { AppointmentMessage } from "@/types/appointment";
import {
  appointmentMessageDetail,
  appointmentMessageSchedule,
  appointmentMessageTarget,
  appointmentMessageTime,
  appointmentMessageTitle,
} from "@/utils/appointmentMessages";

function message(overrides: Partial<AppointmentMessage> = {}): AppointmentMessage {
  return {
    messageKey: "booking:booking-1:patient:created",
    messageType: "booking_created",
    occurredAt: "2026-08-15T07:30:00Z",
    booking: {
      bookingId: "booking-1",
      patientAccountId: "patient-1",
      patientDisplayName: "张明",
      patientPhoneMasked: "153****8538",
      departmentId: "department-1",
      departmentName: "放射科",
      itemId: "item-1",
      itemName: "胸部CT平扫",
      roomId: "room-1",
      roomDisplayName: "影像楼 · 2层 · CT201室",
      campusId: "campus-1",
      campusName: "总院区",
      building: "影像楼",
      floorNumber: 2,
      roomNumber: "CT201",
      serviceDate: "2026-08-16",
      session: "morning",
      status: "confirmed",
      roomOpenTime: "08:00:00",
      roomCloseTime: "12:00:00",
      itemStartTime: "09:00:00",
      itemEndTime: "12:00:00",
      bookingCutoffTime: "11:30:00",
      version: 1,
      createdAt: "2026-08-15T07:30:00Z",
      updatedAt: "2026-08-15T07:30:00Z",
    },
    ...overrides,
  };
}

describe("appointment message presentation", () => {
  it("uses role-specific booking titles and complete message details", () => {
    const value = message();

    expect(appointmentMessageTitle(value.messageType, false)).toBe("预约成功");
    expect(appointmentMessageTitle(value.messageType, true)).toBe("收到新的检查预约");
    expect(appointmentMessageDetail(value, true)).toContain("张明（153****8538）");
    expect(appointmentMessageDetail(value, false)).toBe("胸部CT平扫 · 放射科");
    expect(appointmentMessageSchedule(value)).toBe(
      "2026-08-16 09:00–12:00 · 总院区 · 影像楼 · 2层 · CT201室",
    );
  });

  it("renders backend UTC instants in the hospital timezone", () => {
    expect(appointmentMessageTime("2026-08-15T07:30:00Z")).toBe("2026-08-15 15:30");
    expect(appointmentMessageTime("2026-08-15 15:30:00")).toBe("2026-08-15 15:30");
  });

  it("routes staff, active patient and report messages to their existing pages", () => {
    const value = message();
    expect(appointmentMessageTarget(value, true)).toBe(
      "/pages/admin/appointment/booking-detail?booking_id=booking-1",
    );
    expect(appointmentMessageTarget(value, false)).toBe(
      "/pages/profile/appointments/index?booking_id=booking-1",
    );
    expect(appointmentMessageTarget(message({ messageType: "report_published" }), false)).toBe(
      "/pages/profile/reports/detail?booking_id=booking-1",
    );
    const completed = message({
      booking: { ...value.booking, status: "completed" },
    });
    expect(appointmentMessageTarget(completed, false)).toContain("view=completed");
  });
});
