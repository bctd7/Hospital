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
});
