import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("@/api/client", () => ({ request: requestMock }));

const bookingResponse = {
  booking_id: "booking-a",
  patient_account_id: "patient-a",
  department_id: "department-a",
  item_id: "item-a",
  item_name: "胸部 CT",
  room_id: "room-a",
  room_display_name: "门诊楼 · 3层 · 301室",
  campus_id: "campus-a",
  service_date: "2026-08-14",
  session: "morning",
  status: "confirmed",
  room_open_time: "08:00:00",
  room_close_time: "12:00:00",
  item_start_time: "09:00:00",
  item_end_time: "10:00:00",
  booking_cutoff_time: "08:30:00",
  version: 1,
  created_at: "2026-08-13T10:00:00+08:00",
  updated_at: "2026-08-13T10:00:00+08:00",
};

describe("booking HTTP adapters", () => {
  beforeEach(() => requestMock.mockReset());

  it("keeps patient project lookup scoped to the selected department", async () => {
    requestMock.mockResolvedValueOnce({ items: null });
    const { patientAppointmentApi } = await import("@/api/appointment");

    const items = await patientAppointmentApi.listItems("department-a");

    expect(items).toEqual([]);
    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/appointment/examination-items?owner_department_id=department-a&page=1&page_size=100",
      authenticated: true,
    });
  });

  it("creates a patient booking with an idempotency operation id", async () => {
    requestMock.mockResolvedValueOnce(bookingResponse);
    const { patientAppointmentApi } = await import("@/api/appointment");

    await patientAppointmentApi.createBooking("item-a", "room-a", "2026-08-14", "morning");

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/appointment/bookings",
      method: "POST",
      authenticated: true,
      data: {
        item_id: "item-a",
        room_id: "room-a",
        service_date: "2026-08-14",
        session: "morning",
        operation_id: expect.stringMatching(/^[0-9a-f-]{36}$/),
      },
    });
  });

  it("lets staff check in a booking with optimistic concurrency", async () => {
    requestMock.mockResolvedValueOnce({ ...bookingResponse, status: "checked_in", version: 2 });
    const { staffBookingApi } = await import("@/api/appointment");

    await staffBookingApi.checkInBooking({ bookingId: "booking-a", version: 1 });

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/admin/appointment/bookings/booking-a/check-in",
      method: "POST",
      authenticated: true,
      data: {
        expected_version: 1,
        operation_id: expect.stringMatching(/^[0-9a-f-]{36}$/),
      },
    });
  });

  it("uses separate patient and staff deletion routes", async () => {
    requestMock.mockResolvedValue({ booking_id: "booking-a", deleted: true });
    const { patientAppointmentApi, staffBookingApi } = await import("@/api/appointment");

    await patientAppointmentApi.deleteMyBooking("booking-a", "临时有事");
    await staffBookingApi.deleteBooking("booking-a", "资源调整");

    expect(requestMock.mock.calls[0]?.[0]).toEqual(expect.objectContaining({
      path: "/api/v1/appointment/bookings/booking-a",
      method: "DELETE",
      data: expect.objectContaining({ reason: "临时有事" }),
    }));
    expect(requestMock.mock.calls[1]?.[0]).toEqual(expect.objectContaining({
      path: "/api/v1/admin/appointment/bookings/booking-a",
      method: "DELETE",
      data: expect.objectContaining({ reason: "资源调整" }),
    }));
  });
});
