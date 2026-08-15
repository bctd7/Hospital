export type AppointmentStatus = "active" | "disabled";
export type AppointmentSession = "morning" | "afternoon";
export type PatientBookingStatus = "confirmed" | "queued" | "called" | "in_progress" | "report_pending" | "completed" | "no_show" | "canceled";
export type AppointmentListView = "active" | "completed";
export type ExaminationReportStatus = "draft" | "published";
export type ExaminationReportVersionKind = "original" | "correction";

export interface ExaminationItem {
  itemId: string;
  ownerDepartmentId: string;
  name: string;
  description: string;
  estimatedDurationMinutes: number;
  status: AppointmentStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface ExaminationReportContent {
  objectiveFindings: string;
  impression: string;
  recommendation: string;
  notes: string;
}

export interface ExaminationItemReportTemplate extends ExaminationReportContent {
  itemId: string;
  version: number;
  updatedAt: string;
}

export interface AppointmentRoom {
  roomId: string;
  departmentId: string;
  campusId: string;
  building: string;
  floorNumber: number;
  roomNumber: string;
  displayName: string;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface RoomExaminationItem {
  relationId: string;
  roomId: string;
  itemId: string;
  roomDisplayName: string;
  campusId: string;
  building: string;
  floorNumber: number;
  roomNumber: string;
  itemName: string;
  status: AppointmentStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface RoomWeeklyWindow {
  windowId: string;
  roomId: string;
  weekday: number;
  session: AppointmentSession;
  openTime: string;
  closeTime: string;
  activeCapacity: number;
  status: AppointmentStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface ItemWeeklyWindow {
  windowId: string;
  itemId: string;
  weekday: number;
  session: AppointmentSession;
  startTime: string;
  bookingCutoffTime: string;
  endTime: string;
  status: AppointmentStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface AppointmentPage<T> {
  items: T[];
  page: number;
  pageSize: number;
  total: number;
}

export interface SaveRoomWindowInput {
  windowId?: string;
  weekday: number;
  session: AppointmentSession;
  openTime: string;
  closeTime: string;
  activeCapacity: number;
  expectedVersion: number;
}

export interface RoomLocationInput {
  campusId: string;
  building: string;
  floorNumber: number;
  roomNumber: string;
}

export interface SaveItemWindowInput {
  windowId?: string;
  weekday: number;
  session: AppointmentSession;
  startTime: string;
  bookingCutoffTime: string;
  endTime: string;
  expectedVersion: number;
}

export interface AppointmentManagementApi {
  listItems(departmentId: string, status: AppointmentStatus, page?: number, pageSize?: number): Promise<AppointmentPage<ExaminationItem>>;
  getItem(itemId: string): Promise<ExaminationItem>;
  createItem(departmentId: string, name: string, description: string, estimatedDurationMinutes: number): Promise<ExaminationItem>;
  updateItem(item: ExaminationItem, name: string, description: string, estimatedDurationMinutes: number): Promise<ExaminationItem>;
  setItemEnabled(item: ExaminationItem, enabled: boolean): Promise<ExaminationItem>;
  getItemReportTemplate(itemId: string): Promise<ExaminationItemReportTemplate>;
  saveItemReportTemplate(template: ExaminationItemReportTemplate): Promise<ExaminationItemReportTemplate>;
  listRooms(departmentId: string, page?: number, pageSize?: number): Promise<AppointmentPage<AppointmentRoom>>;
  getRoom(roomId: string): Promise<AppointmentRoom>;
  createRoom(departmentId: string, location: RoomLocationInput): Promise<AppointmentRoom>;
  updateRoom(room: AppointmentRoom, location: RoomLocationInput): Promise<AppointmentRoom>;
  retireRoom(room: AppointmentRoom): Promise<AppointmentRoom>;
  listRoomItems(roomId: string, status: AppointmentStatus, page?: number, pageSize?: number): Promise<AppointmentPage<RoomExaminationItem>>;
  addRoomItem(roomId: string, itemId: string): Promise<RoomExaminationItem>;
  setRoomItemEnabled(relation: RoomExaminationItem, enabled: boolean): Promise<RoomExaminationItem>;
  listItemRooms(itemId: string): Promise<RoomExaminationItem[]>;
  listRoomWindows(roomId: string): Promise<RoomWeeklyWindow[]>;
  saveRoomWindow(roomId: string, input: SaveRoomWindowInput): Promise<RoomWeeklyWindow>;
  disableRoomWindow(window: RoomWeeklyWindow): Promise<RoomWeeklyWindow>;
  listItemWindows(itemId: string): Promise<ItemWeeklyWindow[]>;
  saveItemWindow(itemId: string, input: SaveItemWindowInput): Promise<ItemWeeklyWindow>;
  disableItemWindow(window: ItemWeeklyWindow): Promise<ItemWeeklyWindow>;
}

export interface BookingOption {
  itemId: string;
  roomId: string;
  roomDisplayName: string;
  campusId: string;
  building: string;
  floorNumber: number;
  roomNumber: string;
  serviceDate: string;
  session: AppointmentSession;
  roomOpenTime: string;
  roomCloseTime: string;
  itemStartTime: string;
  itemEndTime: string;
  bookingCutoffTime: string;
  totalCapacity: number;
  remainingCapacity: number;
  estimatedDurationMinutes: number;
}

export interface PatientBooking {
  bookingId: string;
  patientAccountId: string;
  patientDisplayName: string;
  patientPhoneMasked: string;
  departmentId: string;
  departmentName: string;
  itemId: string;
  itemName: string;
  roomId: string;
  roomDisplayName: string;
  campusId: string;
  campusName: string;
  building: string;
  floorNumber: number;
  roomNumber: string;
  serviceDate: string;
  session: AppointmentSession;
  status: PatientBookingStatus;
  roomOpenTime: string;
  roomCloseTime: string;
  itemStartTime: string;
  itemEndTime: string;
  bookingCutoffTime: string;
  estimatedDurationMinutes: number;
  version: number;
  createdAt: string;
  updatedAt: string;
  startedAt?: string;
  startedBy?: string;
  startedByDisplayName?: string;
  examinationEndedAt?: string;
  examinationEndedBy?: string;
  examinationEndedByDisplayName?: string;
  completedAt?: string;
  completedBy?: string;
  completedByDisplayName?: string;
  reportId?: string;
  reportStatus?: ExaminationReportStatus;
  reportVersion?: number;
  queueNumber: number;
  currentCalledQueueNumber: number;
  peopleAhead: number;
  checkedInAt?: string;
  calledAt?: string;
  callDeadline?: string;
  callAttempts: number;
}

export type AppointmentMessageType =
  | "booking_created"
  | "arrival_60m"
  | "arrival_30m"
  | "booking_canceled"
  | "booking_no_show"
  | "booking_called"
  | "booking_deferred"
  | "report_due"
  | "report_overdue"
  | "report_published"
  | "report_corrected";

export interface AppointmentMessage {
  messageKey: string;
  messageType: AppointmentMessageType;
  occurredAt: string;
  readAt?: string;
  booking: PatientBooking;
  reportId?: string;
  reportVersionId?: string;
  reportVersionNo?: number;
}

export interface DepartmentUnreadCount {
  departmentId: string;
  unreadCount: number;
}

export interface AppointmentMessagePage extends AppointmentPage<AppointmentMessage> {
  unreadCount: number;
  departmentUnreadCounts: DepartmentUnreadCount[];
}

export interface AppointmentMessageApi {
  listMine(page?: number, pageSize?: number): Promise<AppointmentMessagePage>;
  markMineRead(messageKey: string): Promise<AppointmentMessage>;
  listDepartment(departmentId: string, page?: number, pageSize?: number): Promise<AppointmentMessagePage>;
  markDepartmentRead(departmentId: string, messageKey: string): Promise<AppointmentMessage>;
}

export interface ExaminationReportVersion extends ExaminationReportContent {
  versionId: string;
  versionNo: number;
  versionKind: ExaminationReportVersionKind;
  status: ExaminationReportStatus;
  correctionReason?: string;
  authoredBy: string;
  authoredByDisplayName: string;
  publishedBy?: string;
  publishedByDisplayName?: string;
  publishedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ExaminationReport {
  reportId: string;
  bookingId: string;
  patientAccountId: string;
  patientDisplayName: string;
  patientPhoneMasked: string;
  departmentId: string;
  departmentName: string;
  itemId: string;
  itemName: string;
  roomId: string;
  campusId: string;
  campusName: string;
  building: string;
  floorNumber: number;
  roomNumber: string;
  roomDisplayName: string;
  status: ExaminationReportStatus;
  performedBy?: string;
  performedByDisplayName?: string;
  examinationStartedAt?: string;
  examinationCompletedAt?: string;
  version: number;
  currentVersion: ExaminationReportVersion;
  createdAt: string;
  updatedAt: string;
}

export interface BookingOptionsResult {
  options: BookingOption[];
  weekStartDate: string;
  weekEndDate: string;
}

export interface PatientAppointmentApi {
  listItems(departmentId: string): Promise<ExaminationItem[]>;
  listItemRooms(itemId: string): Promise<RoomExaminationItem[]>;
  listBookingOptions(itemId: string): Promise<BookingOptionsResult>;
  createBooking(itemId: string, roomId: string, serviceDate: string, session: AppointmentSession): Promise<PatientBooking>;
  listMyBookings(page?: number, pageSize?: number, view?: AppointmentListView): Promise<AppointmentPage<PatientBooking>>;
  getMyBooking(bookingId: string): Promise<PatientBooking>;
  checkIn(booking: PatientBooking): Promise<PatientBooking>;
  deleteMyBooking(bookingId: string, reason?: string): Promise<void>;
  listMyReports(page?: number, pageSize?: number): Promise<AppointmentPage<ExaminationReport>>;
  getMyReport(bookingId: string): Promise<ExaminationReport>;
}

export interface StaffBookingFilters {
  patientKeyword?: string;
  status?: PatientBookingStatus;
  serviceDate?: string;
  roomId?: string;
  view?: AppointmentListView;
}

export interface StaffReportFilters {
  keyword?: string;
}

export interface StaffBookingApi {
  listBookings(departmentId: string, filters?: StaffBookingFilters, page?: number, pageSize?: number): Promise<AppointmentPage<PatientBooking>>;
  listReports(departmentId: string, filters?: StaffReportFilters, page?: number, pageSize?: number): Promise<AppointmentPage<ExaminationReport>>;
  getBooking(bookingId: string): Promise<PatientBooking>;
  startExamination(booking: PatientBooking): Promise<PatientBooking>;
  endExamination(booking: PatientBooking): Promise<PatientBooking>;
  callNext(departmentId: string, roomId: string, serviceDate: string): Promise<PatientBooking>;
  getReport(bookingId: string): Promise<ExaminationReport>;
  saveReportDraft(bookingId: string, content: ExaminationReportContent, expectedReportVersion: number): Promise<ExaminationReport>;
  completeAndPublishReport(booking: PatientBooking, content: ExaminationReportContent, expectedReportVersion: number): Promise<ExaminationReport>;
  correctReport(report: ExaminationReport, content: ExaminationReportContent, correctionReason: string): Promise<ExaminationReport>;
  listReportVersions(reportId: string): Promise<ExaminationReportVersion[]>;
  deleteBooking(bookingId: string, reason?: string): Promise<void>;
}
