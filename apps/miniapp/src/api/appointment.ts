import { request } from "@/api/client";
import { operationId, queryPath } from "@/api/management/httpShared";
import type {
  AppointmentManagementApi,
  AppointmentPage,
  AppointmentRoom,
  AppointmentSession,
  AppointmentStatus,
  BookingOption,
  BookingOptionsResult,
  ExaminationItemReportTemplate,
  ExaminationReport,
  ExaminationReportContent,
  ExaminationReportStatus,
  ExaminationReportVersion,
  ExaminationReportVersionKind,
  ExaminationItem,
  ItemWeeklyWindow,
  RoomExaminationItem,
  RoomLocationInput,
  RoomWeeklyWindow,
  SaveItemWindowInput,
  SaveRoomWindowInput,
  PatientAppointmentApi,
  PatientBooking,
  StaffBookingApi,
  PatientBookingStatus,
} from "@/types/appointment";

interface ItemResponse {
  item_id: string;
  owner_department_id: string;
  name: string;
  description: string;
  status: AppointmentStatus;
  version: number;
  created_at: string;
  updated_at: string;
}

interface RoomResponse {
  room_id: string;
  department_id: string;
  campus_id: string;
  building: string;
  floor_number: number;
  room_number: string;
  display_name: string;
  version: number;
  created_at: string;
  updated_at: string;
}

interface RelationResponse {
  relation_id: string;
  room_id: string;
  item_id: string;
  room_display_name: string;
  campus_id: string;
  building: string;
  floor_number: number;
  room_number: string;
  item_name: string;
  status: AppointmentStatus;
  version: number;
  created_at: string;
  updated_at: string;
}

interface RoomWindowResponse {
  window_id: string;
  room_id: string;
  weekday: number;
  session: AppointmentSession;
  open_time: string;
  close_time: string;
  active_capacity: number;
  status: AppointmentStatus;
  version: number;
  created_at: string;
  updated_at: string;
}

interface ItemWindowResponse {
  window_id: string;
  item_id: string;
  weekday: number;
  session: AppointmentSession;
  start_time: string;
  booking_cutoff_time: string;
  end_time: string;
  status: AppointmentStatus;
  version: number;
  created_at: string;
  updated_at: string;
}

interface BookingOptionResponse {
  item_id: string;
  room_id: string;
  room_display_name: string;
  campus_id: string;
  building: string;
  floor_number: number;
  room_number: string;
  service_date: string;
  session: AppointmentSession;
  room_open_time: string;
  room_close_time: string;
  item_start_time: string;
  item_end_time: string;
  booking_cutoff_time: string;
  total_capacity: number;
  remaining_capacity: number;
}

interface BookingResponse {
  booking_id: string;
  patient_account_id: string;
  patient_display_name: string;
  patient_phone_masked: string;
  department_id: string;
  department_name: string;
  item_id: string;
  item_name: string;
  room_id: string;
  room_display_name: string;
  campus_id: string;
  campus_name: string;
  building: string;
  floor_number: number;
  room_number: string;
  service_date: string;
  session: AppointmentSession;
  status: PatientBookingStatus;
  room_open_time: string;
  room_close_time: string;
  item_start_time: string;
  item_end_time: string;
  booking_cutoff_time: string;
  version: number;
  created_at: string;
  updated_at: string;
  started_at?: string;
  started_by?: string;
  started_by_display_name?: string;
  completed_at?: string;
  completed_by?: string;
  completed_by_display_name?: string;
}

interface ReportContentResponse {
  objective_findings: string;
  impression: string;
  recommendation: string;
  notes: string;
}

interface ItemReportTemplateResponse extends ReportContentResponse {
  item_id: string;
  version: number;
  updated_at: string;
}

interface ReportVersionResponse extends ReportContentResponse {
  version_id: string;
  version_no: number;
  version_kind: ExaminationReportVersionKind;
  status: ExaminationReportStatus;
  correction_reason?: string;
  authored_by: string;
  authored_by_display_name: string;
  published_by?: string;
  published_by_display_name?: string;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

interface ExaminationReportResponse {
  report_id: string;
  booking_id: string;
  patient_account_id: string;
  patient_display_name: string;
  patient_phone_masked: string;
  department_id: string;
  department_name: string;
  item_id: string;
  item_name: string;
  room_id: string;
  campus_id: string;
  campus_name: string;
  building: string;
  floor_number: number;
  room_number: string;
  room_display_name: string;
  status: ExaminationReportStatus;
  performed_by?: string;
  performed_by_display_name?: string;
  examination_started_at?: string;
  examination_completed_at?: string;
  version: number;
  current_version: ReportVersionResponse;
  created_at: string;
  updated_at: string;
}

const item = (value: ItemResponse): ExaminationItem => ({
  itemId: value.item_id,
  ownerDepartmentId: value.owner_department_id,
  name: value.name,
  description: value.description,
  status: value.status,
  version: value.version,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const itemReportTemplate = (value: ItemReportTemplateResponse): ExaminationItemReportTemplate => ({
  itemId: value.item_id,
  objectiveFindings: value.objective_findings,
  impression: value.impression,
  recommendation: value.recommendation,
  notes: value.notes,
  version: value.version,
  updatedAt: value.updated_at,
});

const reportVersion = (value: ReportVersionResponse): ExaminationReportVersion => ({
  versionId: value.version_id,
  versionNo: value.version_no,
  versionKind: value.version_kind,
  status: value.status,
  objectiveFindings: value.objective_findings,
  impression: value.impression,
  recommendation: value.recommendation,
  notes: value.notes,
  correctionReason: value.correction_reason,
  authoredBy: value.authored_by,
  authoredByDisplayName: value.authored_by_display_name,
  publishedBy: value.published_by,
  publishedByDisplayName: value.published_by_display_name,
  publishedAt: value.published_at,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const examinationReport = (value: ExaminationReportResponse): ExaminationReport => ({
  reportId: value.report_id,
  bookingId: value.booking_id,
  patientAccountId: value.patient_account_id,
  patientDisplayName: value.patient_display_name,
  patientPhoneMasked: value.patient_phone_masked,
  departmentId: value.department_id,
  departmentName: value.department_name,
  itemId: value.item_id,
  itemName: value.item_name,
  roomId: value.room_id,
  campusId: value.campus_id,
  campusName: value.campus_name,
  building: value.building,
  floorNumber: value.floor_number,
  roomNumber: value.room_number,
  roomDisplayName: value.room_display_name,
  status: value.status,
  performedBy: value.performed_by,
  performedByDisplayName: value.performed_by_display_name,
  examinationStartedAt: value.examination_started_at,
  examinationCompletedAt: value.examination_completed_at,
  version: value.version,
  currentVersion: reportVersion(value.current_version),
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const room = (value: RoomResponse): AppointmentRoom => ({
  roomId: value.room_id,
  departmentId: value.department_id,
  campusId: value.campus_id,
  building: value.building,
  floorNumber: value.floor_number,
  roomNumber: value.room_number,
  displayName: value.display_name,
  version: value.version,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const relation = (value: RelationResponse): RoomExaminationItem => ({
  relationId: value.relation_id,
  roomId: value.room_id,
  itemId: value.item_id,
  roomDisplayName: value.room_display_name,
  campusId: value.campus_id,
  building: value.building,
  floorNumber: value.floor_number,
  roomNumber: value.room_number,
  itemName: value.item_name,
  status: value.status,
  version: value.version,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const roomWindow = (value: RoomWindowResponse): RoomWeeklyWindow => ({
  windowId: value.window_id,
  roomId: value.room_id,
  weekday: value.weekday,
  session: value.session,
  openTime: value.open_time,
  closeTime: value.close_time,
  activeCapacity: value.active_capacity,
  status: value.status,
  version: value.version,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const itemWindow = (value: ItemWindowResponse): ItemWeeklyWindow => ({
  windowId: value.window_id,
  itemId: value.item_id,
  weekday: value.weekday,
  session: value.session,
  startTime: value.start_time,
  bookingCutoffTime: value.booking_cutoff_time,
  endTime: value.end_time,
  status: value.status,
  version: value.version,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const bookingOption = (value: BookingOptionResponse): BookingOption => ({
  itemId: value.item_id,
  roomId: value.room_id,
  roomDisplayName: value.room_display_name,
  campusId: value.campus_id,
  building: value.building,
  floorNumber: value.floor_number,
  roomNumber: value.room_number,
  serviceDate: value.service_date,
  session: value.session,
  roomOpenTime: value.room_open_time,
  roomCloseTime: value.room_close_time,
  itemStartTime: value.item_start_time,
  itemEndTime: value.item_end_time,
  bookingCutoffTime: value.booking_cutoff_time,
  totalCapacity: value.total_capacity,
  remainingCapacity: value.remaining_capacity,
});

const booking = (value: BookingResponse): PatientBooking => ({
  bookingId: value.booking_id,
  patientAccountId: value.patient_account_id,
  patientDisplayName: value.patient_display_name,
  patientPhoneMasked: value.patient_phone_masked,
  departmentId: value.department_id,
  departmentName: value.department_name,
  itemId: value.item_id,
  itemName: value.item_name,
  roomId: value.room_id,
  roomDisplayName: value.room_display_name,
  campusId: value.campus_id,
  campusName: value.campus_name,
  building: value.building,
  floorNumber: value.floor_number,
  roomNumber: value.room_number,
  serviceDate: value.service_date,
  session: value.session,
  status: value.status,
  roomOpenTime: value.room_open_time,
  roomCloseTime: value.room_close_time,
  itemStartTime: value.item_start_time,
  itemEndTime: value.item_end_time,
  bookingCutoffTime: value.booking_cutoff_time,
  version: value.version,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
  startedAt: value.started_at,
  startedBy: value.started_by,
  startedByDisplayName: value.started_by_display_name,
  completedAt: value.completed_at,
  completedBy: value.completed_by,
  completedByDisplayName: value.completed_by_display_name,
});

const retryOperationIds = new Map<string, string>();

async function mutation<T>(
  key: string,
  path: string,
  method: "POST" | "PUT",
  data: Record<string, unknown>,
): Promise<T> {
  const currentOperationId = retryOperationIds.get(key) ?? operationId();
  retryOperationIds.set(key, currentOperationId);
  const result = await request<T>({
    path,
    method,
    authenticated: true,
    data: { ...data, operation_id: currentOperationId },
  });
  retryOperationIds.delete(key);
  return result;
}

function statusBody(resourceId: string, version: number) {
  return { resource_id: resourceId, expected_version: version };
}

function page<T>(values: T[], value: { page: number; page_size: number; total: number }): AppointmentPage<T> {
  return { items: values, page: value.page, pageSize: value.page_size, total: value.total };
}

function arrayOrEmpty<T>(values: T[] | null | undefined): T[] {
  return Array.isArray(values) ? values : [];
}

export const appointmentManagementApi: AppointmentManagementApi = {
  async listItems(departmentId, status, currentPage = 1, pageSize = 50) {
    const value = await request<{ items: ItemResponse[] | null; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/admin/appointment/examination-items", {
        owner_department_id: departmentId,
        status,
        page: currentPage,
        page_size: pageSize,
      }),
      authenticated: true,
    });
    return page(arrayOrEmpty(value.items).map(item), value);
  },

  async getItem(itemId) {
    return item(await request<ItemResponse>({ path: `/api/v1/admin/appointment/examination-items/${encodeURIComponent(itemId)}`, authenticated: true }));
  },

  async createItem(departmentId, name, description) {
    const key = `item:create:${departmentId}:${name}:${description}`;
    return item(await mutation<ItemResponse>(key, "/api/v1/admin/appointment/examination-items", "POST", {
      owner_department_id: departmentId, name, description,
    }));
  },

  async updateItem(value, name, description) {
    const key = `item:update:${value.itemId}:${value.version}:${name}:${description}`;
    return item(await mutation<ItemResponse>(key, `/api/v1/admin/appointment/examination-items/${encodeURIComponent(value.itemId)}`, "PUT", {
      name, description, expected_version: value.version,
    }));
  },

  async setItemEnabled(value, enabled) {
    const key = `item:status:${value.itemId}:${value.version}:${enabled}`;
    return item(await mutation<ItemResponse>(key, `/api/v1/admin/appointment/examination-items/${encodeURIComponent(value.itemId)}/${enabled ? "enable" : "disable"}`, "POST", statusBody(value.itemId, value.version)));
  },

  async getItemReportTemplate(itemId) {
    return itemReportTemplate(await request<ItemReportTemplateResponse>({
      path: `/api/v1/admin/appointment/examination-items/${encodeURIComponent(itemId)}/report-template`,
      authenticated: true,
    }));
  },

  async saveItemReportTemplate(value) {
    const data = {
      objective_findings: value.objectiveFindings,
      impression: value.impression,
      recommendation: value.recommendation,
      notes: value.notes,
      expected_template_version: value.version,
    };
    return itemReportTemplate(await mutation<ItemReportTemplateResponse>(
      `item-report-template:${value.itemId}:${value.version}:${JSON.stringify(data)}`,
      `/api/v1/admin/appointment/examination-items/${encodeURIComponent(value.itemId)}/report-template`,
      "PUT",
      data,
    ));
  },

  async listRooms(departmentId, currentPage = 1, pageSize = 50) {
    const value = await request<{ rooms: RoomResponse[] | null; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/admin/appointment/rooms", { department_id: departmentId, page: currentPage, page_size: pageSize }),
      authenticated: true,
    });
    return page(arrayOrEmpty(value.rooms).map(room), value);
  },

  async getRoom(roomId) {
    return room(await request<RoomResponse>({ path: `/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}`, authenticated: true }));
  },

  async createRoom(departmentId, location: RoomLocationInput) {
    const data = { department_id: departmentId, campus_id: location.campusId, building: location.building, floor_number: location.floorNumber, room_number: location.roomNumber };
    return room(await mutation<RoomResponse>(`room:create:${JSON.stringify(data)}`, "/api/v1/admin/appointment/rooms", "POST", data));
  },

  async updateRoom(value, location: RoomLocationInput) {
    const data = { campus_id: location.campusId, building: location.building, floor_number: location.floorNumber, room_number: location.roomNumber, expected_version: value.version };
    return room(await mutation<RoomResponse>(`room:update:${value.roomId}:${JSON.stringify(data)}`, `/api/v1/admin/appointment/rooms/${encodeURIComponent(value.roomId)}`, "PUT", data));
  },

  async retireRoom(value) {
    return room(await mutation<RoomResponse>(`room:retire:${value.roomId}:${value.version}`, `/api/v1/admin/appointment/rooms/${encodeURIComponent(value.roomId)}/retire`, "POST", statusBody(value.roomId, value.version)));
  },

  async listRoomItems(roomId, status, currentPage = 1, pageSize = 100) {
    const value = await request<{ relations: RelationResponse[] | null; page: number; page_size: number; total: number }>({
      path: queryPath(`/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}/examination-items`, { status, page: currentPage, page_size: pageSize }),
      authenticated: true,
    });
    return page(arrayOrEmpty(value.relations).map(relation), value);
  },

  async addRoomItem(roomId, itemId) {
    const key = `relation:add:${roomId}:${itemId}`;
    return relation(await mutation<RelationResponse>(key, `/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}/examination-items`, "POST", { item_id: itemId }));
  },

  async setRoomItemEnabled(value, enabled) {
    const key = `relation:status:${value.relationId}:${value.version}:${enabled}`;
    return relation(await mutation<RelationResponse>(key, `/api/v1/admin/appointment/room-examination-items/${encodeURIComponent(value.relationId)}/${enabled ? "enable" : "disable"}`, "POST", statusBody(value.relationId, value.version)));
  },

  async listItemRooms(itemId) {
    const value = await request<{ relations: RelationResponse[] | null }>({
      path: `/api/v1/appointment/examination-items/${encodeURIComponent(itemId)}/rooms`,
      authenticated: true,
    });
    return arrayOrEmpty(value.relations).map(relation);
  },

  async listRoomWindows(roomId) {
    const value = await request<{ windows: RoomWindowResponse[] | null }>({
      path: `/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}/weekly-windows`,
      authenticated: true,
    });
    return arrayOrEmpty(value.windows).map(roomWindow);
  },

  async saveRoomWindow(roomId, input: SaveRoomWindowInput) {
    const data = {
      window_id: input.windowId ?? "", weekday: input.weekday, session: input.session,
      open_time: input.openTime, close_time: input.closeTime,
      active_capacity: input.activeCapacity, expected_version: input.expectedVersion,
    };
    return roomWindow(await mutation<RoomWindowResponse>(`room-window:set:${roomId}:${JSON.stringify(data)}`, `/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}/weekly-window`, "PUT", data));
  },

  async disableRoomWindow(value) {
    const key = `room-window:disable:${value.windowId}:${value.version}`;
    return roomWindow(await mutation<RoomWindowResponse>(key, `/api/v1/admin/appointment/room-weekly-windows/${encodeURIComponent(value.windowId)}/disable`, "POST", statusBody(value.windowId, value.version)));
  },

  async listItemWindows(itemId) {
    const value = await request<{ windows: ItemWindowResponse[] | null }>({
      path: `/api/v1/admin/appointment/examination-items/${encodeURIComponent(itemId)}/weekly-windows`,
      authenticated: true,
    });
    return arrayOrEmpty(value.windows).map(itemWindow);
  },

  async saveItemWindow(itemId, input: SaveItemWindowInput) {
    const data = {
      window_id: input.windowId ?? "", weekday: input.weekday, session: input.session,
      start_time: input.startTime, booking_cutoff_time: input.bookingCutoffTime,
      end_time: input.endTime, expected_version: input.expectedVersion,
    };
    return itemWindow(await mutation<ItemWindowResponse>(`item-window:set:${itemId}:${JSON.stringify(data)}`, `/api/v1/admin/appointment/examination-items/${encodeURIComponent(itemId)}/weekly-window`, "PUT", data));
  },

  async disableItemWindow(value) {
    const key = `item-window:disable:${value.windowId}:${value.version}`;
    return itemWindow(await mutation<ItemWindowResponse>(key, `/api/v1/admin/appointment/item-weekly-windows/${encodeURIComponent(value.windowId)}/disable`, "POST", statusBody(value.windowId, value.version)));
  },
};

export const patientAppointmentApi: PatientAppointmentApi = {
  async listItems(departmentId) {
    const value = await request<{ items: ItemResponse[] | null }>({
      path: queryPath("/api/v1/appointment/examination-items", {
        owner_department_id: departmentId,
        page: 1,
        page_size: 100,
      }),
      authenticated: true,
    });
    return arrayOrEmpty(value.items).map(item);
  },

  async listItemRooms(itemId) {
    const value = await request<{ relations: RelationResponse[] | null }>({
      path: `/api/v1/appointment/examination-items/${encodeURIComponent(itemId)}/rooms`,
      authenticated: true,
    });
    return arrayOrEmpty(value.relations).map(relation);
  },

  async listBookingOptions(itemId): Promise<BookingOptionsResult> {
    const value = await request<{
      options: BookingOptionResponse[] | null;
      week_start_date: string;
      week_end_date: string;
    }>({
      path: `/api/v1/appointment/examination-items/${encodeURIComponent(itemId)}/booking-options`,
      authenticated: true,
    });
    return {
      options: arrayOrEmpty(value.options).map(bookingOption),
      weekStartDate: value.week_start_date,
      weekEndDate: value.week_end_date,
    };
  },

  async createBooking(itemId, roomId, serviceDate, session) {
    const data = { item_id: itemId, room_id: roomId, service_date: serviceDate, session };
    return booking(await mutation<BookingResponse>(
      `booking:create:${itemId}:${roomId}:${serviceDate}:${session}`,
      "/api/v1/appointment/bookings",
      "POST",
      data,
    ));
  },

  async listMyBookings(currentPage = 1, pageSize = 20, view = "active") {
    const value = await request<{ bookings: BookingResponse[] | null; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/appointment/bookings", { view, page: currentPage, page_size: pageSize }),
      authenticated: true,
    });
    return page(arrayOrEmpty(value.bookings).map(booking), value);
  },

  async getMyBooking(bookingId) {
    return booking(await request<BookingResponse>({
      path: `/api/v1/appointment/bookings/${encodeURIComponent(bookingId)}`,
      authenticated: true,
    }));
  },

  async deleteMyBooking(bookingId, reason = "") {
    await request<{ booking_id: string; deleted: boolean }>({
      path: `/api/v1/appointment/bookings/${encodeURIComponent(bookingId)}`,
      method: "DELETE",
      authenticated: true,
      data: { operation_id: operationId(), reason },
    });
  },

  async listMyReports(currentPage = 1, pageSize = 20) {
    const value = await request<{ reports: ExaminationReportResponse[] | null; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/appointment/reports", { page: currentPage, page_size: pageSize }),
      authenticated: true,
    });
    return page(arrayOrEmpty(value.reports).map(examinationReport), value);
  },

  async getMyReport(bookingId) {
    return examinationReport(await request<ExaminationReportResponse>({
      path: `/api/v1/appointment/bookings/${encodeURIComponent(bookingId)}/report`,
      authenticated: true,
    }));
  },
};

export const staffBookingApi: StaffBookingApi = {
  async listBookings(departmentId, filters = {}, currentPage = 1, pageSize = 100) {
    const value = await request<{ bookings: BookingResponse[] | null; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/admin/appointment/bookings", {
        department_id: departmentId,
        patient_keyword: filters.patientKeyword,
        status: filters.status,
        view: filters.view ?? "active",
        page: currentPage,
        page_size: pageSize,
      }),
      authenticated: true,
    });
    return page(arrayOrEmpty(value.bookings).map(booking), value);
  },

  async listReports(departmentId, filters = {}, currentPage = 1, pageSize = 50) {
    const value = await request<{ reports: ExaminationReportResponse[] | null; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/admin/appointment/reports", {
        department_id: departmentId,
        keyword: filters.keyword,
        page: currentPage,
        page_size: pageSize,
      }),
      authenticated: true,
    });
    return page(arrayOrEmpty(value.reports).map(examinationReport), value);
  },

  async getBooking(bookingId) {
    return booking(await request<BookingResponse>({
      path: `/api/v1/admin/appointment/bookings/${encodeURIComponent(bookingId)}`,
      authenticated: true,
    }));
  },

  async startExamination(value) {
    return booking(await mutation<BookingResponse>(
      `booking:start:${value.bookingId}:${value.version}`,
      `/api/v1/admin/appointment/bookings/${encodeURIComponent(value.bookingId)}/start-examination`,
      "POST",
      { expected_version: value.version },
    ));
  },

  async getReport(bookingId) {
    return examinationReport(await request<ExaminationReportResponse>({
      path: `/api/v1/admin/appointment/bookings/${encodeURIComponent(bookingId)}/report`,
      authenticated: true,
    }));
  },

  async saveReportDraft(bookingId, content, expectedReportVersion) {
    const data = reportContentBody(content, { expected_report_version: expectedReportVersion });
    return examinationReport(await mutation<ExaminationReportResponse>(
      `report:draft:${bookingId}:${expectedReportVersion}:${JSON.stringify(data)}`,
      `/api/v1/admin/appointment/bookings/${encodeURIComponent(bookingId)}/report/draft`,
      "PUT",
      data,
    ));
  },

  async completeAndPublishReport(value, content, expectedReportVersion) {
    const data = reportContentBody(content, {
      expected_booking_version: value.version,
      expected_report_version: expectedReportVersion,
    });
    return examinationReport(await mutation<ExaminationReportResponse>(
      `report:publish:${value.bookingId}:${value.version}:${expectedReportVersion}:${JSON.stringify(data)}`,
      `/api/v1/admin/appointment/bookings/${encodeURIComponent(value.bookingId)}/report/complete-and-publish`,
      "POST",
      data,
    ));
  },

  async correctReport(value, content, correctionReason) {
    const data = reportContentBody(content, {
      correction_reason: correctionReason,
      expected_report_version: value.version,
    });
    return examinationReport(await mutation<ExaminationReportResponse>(
      `report:correct:${value.reportId}:${value.version}:${JSON.stringify(data)}`,
      `/api/v1/admin/appointment/reports/${encodeURIComponent(value.reportId)}/corrections`,
      "POST",
      data,
    ));
  },

  async listReportVersions(reportId) {
    const value = await request<{ versions: ReportVersionResponse[] | null }>({
      path: `/api/v1/admin/appointment/reports/${encodeURIComponent(reportId)}/versions`,
      authenticated: true,
    });
    return arrayOrEmpty(value.versions).map(reportVersion);
  },

  async deleteBooking(bookingId, reason = "工作人员删除") {
    await request<{ booking_id: string; deleted: boolean }>({
      path: `/api/v1/admin/appointment/bookings/${encodeURIComponent(bookingId)}`,
      method: "DELETE",
      authenticated: true,
      data: { operation_id: operationId(), reason },
    });
  },
};

function reportContentBody(content: ExaminationReportContent, extra: Record<string, unknown>): Record<string, unknown> {
  return {
    objective_findings: content.objectiveFindings,
    impression: content.impression,
    recommendation: content.recommendation,
    notes: content.notes,
    ...extra,
  };
}
