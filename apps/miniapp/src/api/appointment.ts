import { request } from "@/api/client";
import { operationId, queryPath } from "@/api/management/httpShared";
import type {
  AppointmentManagementApi,
  AppointmentPage,
  AppointmentRoom,
  AppointmentSession,
  AppointmentStatus,
  ExaminationItem,
  ItemWeeklyWindow,
  RoomExaminationItem,
  RoomWeeklyWindow,
  SaveItemWindowInput,
  SaveRoomWindowInput,
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
  name: string;
  status: AppointmentStatus;
  version: number;
  created_at: string;
  updated_at: string;
}

interface RelationResponse {
  relation_id: string;
  room_id: string;
  item_id: string;
  room_name: string;
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

const room = (value: RoomResponse): AppointmentRoom => ({
  roomId: value.room_id,
  departmentId: value.department_id,
  name: value.name,
  status: value.status,
  version: value.version,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
});

const relation = (value: RelationResponse): RoomExaminationItem => ({
  relationId: value.relation_id,
  roomId: value.room_id,
  itemId: value.item_id,
  roomName: value.room_name,
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

export const appointmentManagementApi: AppointmentManagementApi = {
  async listItems(departmentId, status, currentPage = 1, pageSize = 50) {
    const value = await request<{ items: ItemResponse[]; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/admin/appointment/examination-items", {
        owner_department_id: departmentId,
        status,
        page: currentPage,
        page_size: pageSize,
      }),
      authenticated: true,
    });
    return page(value.items.map(item), value);
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

  async listRooms(departmentId, status, currentPage = 1, pageSize = 50) {
    const value = await request<{ rooms: RoomResponse[]; page: number; page_size: number; total: number }>({
      path: queryPath("/api/v1/admin/appointment/rooms", { department_id: departmentId, status, page: currentPage, page_size: pageSize }),
      authenticated: true,
    });
    return page(value.rooms.map(room), value);
  },

  async getRoom(roomId) {
    return room(await request<RoomResponse>({ path: `/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}`, authenticated: true }));
  },

  async createRoom(departmentId, name) {
    const key = `room:create:${departmentId}:${name}`;
    return room(await mutation<RoomResponse>(key, "/api/v1/admin/appointment/rooms", "POST", { department_id: departmentId, name }));
  },

  async updateRoom(value, name) {
    const key = `room:update:${value.roomId}:${value.version}:${name}`;
    return room(await mutation<RoomResponse>(key, `/api/v1/admin/appointment/rooms/${encodeURIComponent(value.roomId)}`, "PUT", { name, expected_version: value.version }));
  },

  async setRoomEnabled(value, enabled) {
    const key = `room:status:${value.roomId}:${value.version}:${enabled}`;
    return room(await mutation<RoomResponse>(key, `/api/v1/admin/appointment/rooms/${encodeURIComponent(value.roomId)}/${enabled ? "enable" : "disable"}`, "POST", statusBody(value.roomId, value.version)));
  },

  async listRoomItems(roomId, status, currentPage = 1, pageSize = 100) {
    const value = await request<{ relations: RelationResponse[]; page: number; page_size: number; total: number }>({
      path: queryPath(`/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}/examination-items`, { status, page: currentPage, page_size: pageSize }),
      authenticated: true,
    });
    return page(value.relations.map(relation), value);
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
    const value = await request<{ relations: RelationResponse[] }>({
      path: `/api/v1/appointment/examination-items/${encodeURIComponent(itemId)}/rooms`,
      authenticated: true,
    });
    return value.relations.map(relation);
  },

  async listRoomWindows(roomId) {
    const value = await request<{ windows: RoomWindowResponse[] }>({
      path: `/api/v1/admin/appointment/rooms/${encodeURIComponent(roomId)}/weekly-windows`,
      authenticated: true,
    });
    return value.windows.map(roomWindow);
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
    const value = await request<{ windows: ItemWindowResponse[] }>({
      path: `/api/v1/admin/appointment/examination-items/${encodeURIComponent(itemId)}/weekly-windows`,
      authenticated: true,
    });
    return value.windows.map(itemWindow);
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
