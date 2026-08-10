import { staffManagementApi } from "@/api/staffManagement";
import type { DepartmentSummary, DoctorSummary } from "@/types/staffManagement";

const CACHE_TTL_MS = 30_000;

interface CacheEntry<T> {
  value: T;
  expiresAt: number;
}

const departmentCache = new Map<string, CacheEntry<DepartmentSummary[]>>();
const doctorCache = new Map<string, CacheEntry<DoctorSummary[]>>();

function current<T>(entry?: CacheEntry<T>): T | undefined {
  return entry && entry.expiresAt > Date.now() ? entry.value : undefined;
}

export async function loadDepartments(
  campusId: string,
  includeDisabled = false,
  force = false,
): Promise<DepartmentSummary[]> {
  const key = `${campusId}:${includeDisabled ? "all" : "active"}`;
  const cached = force ? undefined : current(departmentCache.get(key));
  if (cached) {
    return cached.map((item) => ({ ...item }));
  }
  const result = await staffManagementApi.listDepartments(campusId, includeDisabled);
  departmentCache.set(key, { value: result, expiresAt: Date.now() + CACHE_TTL_MS });
  return result.map((item) => ({ ...item }));
}

export async function loadDoctors(
  departmentId: string,
  force = false,
): Promise<DoctorSummary[]> {
  const cached = force ? undefined : current(doctorCache.get(departmentId));
  if (cached) {
    return cached.map((item) => ({ ...item }));
  }
  const result = await staffManagementApi.listDoctors(departmentId);
  doctorCache.set(departmentId, {
    value: result,
    expiresAt: Date.now() + CACHE_TTL_MS,
  });
  return result.map((item) => ({ ...item }));
}

export function invalidateDepartments() {
  departmentCache.clear();
}

export function invalidateDoctors(...departmentIds: Array<string | undefined>) {
  departmentIds.forEach((departmentId) => {
    if (departmentId) {
      doctorCache.delete(departmentId);
    }
  });
}

export function clearStaffManagementCache() {
  departmentCache.clear();
  doctorCache.clear();
}
