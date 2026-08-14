import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("@/api/client", () => ({ request: requestMock }));

const bookingResponse = {
  booking_id: "booking-a",
  patient_account_id: "patient-a",
  patient_display_name: "测试患者",
  patient_phone_masked: "134****4556",
  department_id: "department-a",
  department_name: "放射科",
  item_id: "item-a",
  item_name: "胸部 CT",
  room_id: "room-a",
  room_display_name: "门诊楼 · 3层 · 301室",
  campus_id: "campus-a",
  campus_name: "南院区",
  building: "门诊楼",
  floor_number: 3,
  room_number: "301",
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

const reportResponse = {
  report_id: "report-a",
  booking_id: "booking-a",
  patient_account_id: "patient-a",
  patient_display_name: "测试患者",
  patient_phone_masked: "134****4556",
  department_id: "department-a",
  department_name: "放射科",
  item_id: "item-a",
  item_name: "胸部 CT",
  room_id: "room-a",
  campus_id: "campus-a",
  campus_name: "南院区",
  building: "门诊楼",
  floor_number: 3,
  room_number: "301",
  room_display_name: "门诊楼 · 3层 · 301室",
  status: "published",
  performed_by: "staff-a",
  performed_by_display_name: "张医生",
  examination_started_at: "2026-08-14T09:00:00+08:00",
  examination_completed_at: "2026-08-14T09:30:00+08:00",
  version: 1,
  current_version: {
    version_id: "version-a",
    version_no: 1,
    version_kind: "original",
    status: "published",
    objective_findings: "未见明显异常",
    impression: "胸部 CT 未见明显异常",
    recommendation: "",
    notes: "",
    authored_by: "staff-a",
    authored_by_display_name: "张医生",
    published_by: "staff-a",
    published_by_display_name: "张医生",
    published_at: "2026-08-14T09:30:00+08:00",
    created_at: "2026-08-14T09:20:00+08:00",
    updated_at: "2026-08-14T09:30:00+08:00",
  },
  created_at: "2026-08-14T09:20:00+08:00",
  updated_at: "2026-08-14T09:30:00+08:00",
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

  it("lets staff start a confirmed examination with optimistic concurrency", async () => {
    requestMock.mockResolvedValueOnce({ ...bookingResponse, status: "in_progress", version: 2 });
    const { staffBookingApi } = await import("@/api/appointment");

    await staffBookingApi.startExamination({ bookingId: "booking-a", version: 1 });

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/admin/appointment/bookings/booking-a/start-examination",
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

  it("searches only the selected department bookings by patient keyword", async () => {
    requestMock.mockResolvedValueOnce({ bookings: [], page: 1, page_size: 20, total: 0 });
    const { staffBookingApi } = await import("@/api/appointment");

    await staffBookingApi.listBookings("department-a", { patientKeyword: "4556", status: "in_progress" }, 1, 20);

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/admin/appointment/bookings?department_id=department-a&patient_keyword=4556&status=in_progress&view=active&page=1&page_size=20",
      authenticated: true,
    });
  });

  it("separates completed patient bookings into the post-visit view", async () => {
    requestMock.mockResolvedValueOnce({ bookings: [], page: 1, page_size: 20, total: 0 });
    const { patientAppointmentApi } = await import("@/api/appointment");

    await patientAppointmentApi.listMyBookings(1, 20, "completed");

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/appointment/bookings?view=completed&page=1&page_size=20",
      authenticated: true,
    });
  });

  it("maps the patient report snapshot without exposing another version endpoint", async () => {
    requestMock.mockResolvedValueOnce(reportResponse);
    const { patientAppointmentApi } = await import("@/api/appointment");

    const report = await patientAppointmentApi.getMyReport("booking-a");

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/appointment/bookings/booking-a/report",
      authenticated: true,
    });
    expect(report).toMatchObject({
      patientDisplayName: "测试患者",
      departmentName: "放射科",
      campusName: "南院区",
      performedByDisplayName: "张医生",
      currentVersion: { versionNo: 1, impression: "胸部 CT 未见明显异常" },
    });
  });

  it("lists published reports within the selected staff department", async () => {
    requestMock.mockResolvedValueOnce({ reports: [], page: 1, page_size: 50, total: 0 });
    const { staffBookingApi } = await import("@/api/appointment");

    await staffBookingApi.listReports("department-a", { keyword: "测试患者" }, 1, 50);

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/admin/appointment/reports?department_id=department-a&keyword=%E6%B5%8B%E8%AF%95%E6%82%A3%E8%80%85&page=1&page_size=50",
      authenticated: true,
    });
  });

  it("publishes the report with both booking and report optimistic versions", async () => {
    requestMock.mockResolvedValueOnce(reportResponse);
    const { staffBookingApi } = await import("@/api/appointment");

    await staffBookingApi.completeAndPublishReport(
      { ...bookingResponse, bookingId: "booking-a", version: 3 } as never,
      { objectiveFindings: "未见明显异常", impression: "正常", recommendation: "", notes: "" },
      2,
    );

    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({
      path: "/api/v1/admin/appointment/bookings/booking-a/report/complete-and-publish",
      method: "POST",
      data: expect.objectContaining({ expected_booking_version: 3, expected_report_version: 2 }),
    }));
  });
});
