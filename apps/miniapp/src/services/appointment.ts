import { appointmentManagementApi } from "@/api/appointment";
import type {
  AppointmentPage,
  AppointmentRoom,
  AppointmentStatus,
  ExaminationItem,
  ItemWeeklyWindow,
  RoomExaminationItem,
  RoomWeeklyWindow,
} from "@/types/appointment";

const CACHE_TTL_MS = 20_000;

interface CacheEntry<T> {
  value?: T;
  expiresAt: number;
  pending?: Promise<T>;
}

class AppointmentQueryCache<T> {
  private readonly entries = new Map<string, CacheEntry<T>>();

  load(key: string, loader: () => Promise<T>, force = false): Promise<T> {
    const current = this.entries.get(key);
    if (!force && current?.value !== undefined && current.expiresAt > Date.now()) {
      return Promise.resolve(current.value);
    }
    if (!force && current?.pending) return current.pending;
    const entry: CacheEntry<T> = { expiresAt: 0 };
    entry.pending = loader()
      .then((value) => {
        if (this.entries.get(key) === entry) {
          entry.value = value;
          entry.expiresAt = Date.now() + CACHE_TTL_MS;
          entry.pending = undefined;
        }
        return value;
      })
      .catch((error) => {
        if (this.entries.get(key) === entry) this.entries.delete(key);
        throw error;
      });
    this.entries.set(key, entry);
    return entry.pending;
  }

  invalidate(...prefixes: string[]) {
    if (!prefixes.length) {
      this.entries.clear();
      return;
    }
    Array.from(this.entries.keys()).forEach((key) => {
      if (prefixes.some((prefix) => key.startsWith(prefix))) this.entries.delete(key);
    });
  }
}

const itemPages = new AppointmentQueryCache<AppointmentPage<ExaminationItem>>();
const roomPages = new AppointmentQueryCache<AppointmentPage<AppointmentRoom>>();
const relationPages = new AppointmentQueryCache<AppointmentPage<RoomExaminationItem>>();
const itemWindows = new AppointmentQueryCache<ItemWeeklyWindow[]>();
const roomWindows = new AppointmentQueryCache<RoomWeeklyWindow[]>();

export function loadExaminationItems(
  departmentId: string,
  status: AppointmentStatus,
  page = 1,
  pageSize = 50,
  force = false,
) {
  const key = `${departmentId}:${status}:${page}:${pageSize}`;
  return itemPages.load(
    key,
    () => appointmentManagementApi.listItems(departmentId, status, page, pageSize),
    force,
  );
}

export function loadAppointmentRooms(
  departmentId: string,
  page = 1,
  force = false,
) {
  const key = `${departmentId}:${page}`;
  return roomPages.load(key, () => appointmentManagementApi.listRooms(departmentId, page), force);
}

export function loadRoomExaminationItems(
  roomId: string,
  status: AppointmentStatus,
  force = false,
) {
  const key = `${roomId}:${status}`;
  return relationPages.load(key, () => appointmentManagementApi.listRoomItems(roomId, status), force);
}

export function loadRoomWeeklyWindows(roomId: string, force = false) {
  return roomWindows.load(roomId, () => appointmentManagementApi.listRoomWindows(roomId), force);
}

export function loadItemWeeklyWindows(itemId: string, force = false) {
  return itemWindows.load(itemId, () => appointmentManagementApi.listItemWindows(itemId), force);
}

export function invalidateDepartmentAppointment(departmentId: string) {
  itemPages.invalidate(`${departmentId}:`);
  roomPages.invalidate(`${departmentId}:`);
}

export function invalidateItemAppointment(itemId: string, departmentId?: string) {
  itemWindows.invalidate(itemId);
  relationPages.invalidate();
  if (departmentId) itemPages.invalidate(`${departmentId}:`);
}

export function invalidateRoomAppointment(roomId: string, departmentId?: string) {
  roomWindows.invalidate(roomId);
  relationPages.invalidate(`${roomId}:`);
  if (departmentId) roomPages.invalidate(`${departmentId}:`);
}

export function clearAppointmentCache() {
  itemPages.invalidate();
  roomPages.invalidate();
  relationPages.invalidate();
  itemWindows.invalidate();
  roomWindows.invalidate();
}
