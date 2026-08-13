export type AppointmentStatus = "active" | "disabled";
export type AppointmentSession = "morning" | "afternoon";

export interface ExaminationItem {
  itemId: string;
  ownerDepartmentId: string;
  name: string;
  description: string;
  status: AppointmentStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface AppointmentRoom {
  roomId: string;
  departmentId: string;
  name: string;
  status: AppointmentStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface RoomExaminationItem {
  relationId: string;
  roomId: string;
  itemId: string;
  roomName: string;
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
  createItem(departmentId: string, name: string, description: string): Promise<ExaminationItem>;
  updateItem(item: ExaminationItem, name: string, description: string): Promise<ExaminationItem>;
  setItemEnabled(item: ExaminationItem, enabled: boolean): Promise<ExaminationItem>;
  listRooms(departmentId: string, status: AppointmentStatus, page?: number, pageSize?: number): Promise<AppointmentPage<AppointmentRoom>>;
  getRoom(roomId: string): Promise<AppointmentRoom>;
  createRoom(departmentId: string, name: string): Promise<AppointmentRoom>;
  updateRoom(room: AppointmentRoom, name: string): Promise<AppointmentRoom>;
  setRoomEnabled(room: AppointmentRoom, enabled: boolean): Promise<AppointmentRoom>;
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
