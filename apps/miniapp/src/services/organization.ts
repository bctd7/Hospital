import {
  organizationAdminApi,
  organizationDirectoryApi,
} from "@/api/staffManagement";
import type {
  CampusSummary,
  DepartmentSummary,
  DoctorSummary,
  OrganizationContext,
} from "@/types/staffManagement";

const CACHE_TTL_MS = 30_000;

interface CacheEntry<T> {
  value?: T;
  expiresAt: number;
  pending?: Promise<T>;
}

class QueryCache<T> {
  private readonly entries = new Map<string, CacheEntry<T>>();

  async load(
    key: string,
    loader: () => Promise<T>,
    clone: (value: T) => T,
    force = false,
  ): Promise<T> {
    const current = this.entries.get(key);
    if (!force && current?.value !== undefined && current.expiresAt > Date.now()) {
      return clone(current.value);
    }
    if (!force && current?.pending) {
      return current.pending.then(clone);
    }

    const next: CacheEntry<T> = { expiresAt: 0 };
    const pending = loader()
      .then((value) => {
        if (this.entries.get(key) === next) {
          next.value = value;
          next.expiresAt = Date.now() + CACHE_TTL_MS;
          next.pending = undefined;
        }
        return value;
      })
      .catch((error) => {
        if (this.entries.get(key) === next) {
          this.entries.delete(key);
        }
        throw error;
      });
    next.pending = pending;
    this.entries.set(key, next);
    return pending.then(clone);
  }

  invalidate(...keys: string[]) {
    if (keys.length === 0) {
      this.entries.clear();
      return;
    }
    keys.forEach((key) => this.entries.delete(key));
  }
}

const contextCache = new QueryCache<OrganizationContext>();
const campusCache = new QueryCache<CampusSummary[]>();
const departmentCache = new QueryCache<DepartmentSummary[]>();
const doctorCache = new QueryCache<DoctorSummary[]>();

export interface DepartmentOption {
  department: DepartmentSummary;
  campus: CampusSummary;
  label: string;
}

const cloneContext = (value: OrganizationContext): OrganizationContext => ({
  hospital: { ...value.hospital },
  campuses: value.campuses.map((campus) => ({ ...campus })),
});
const cloneCampuses = (value: CampusSummary[]) =>
  value.map((campus) => ({ ...campus }));
const cloneDepartments = (value: DepartmentSummary[]) =>
  value.map((department) => ({ ...department }));
const cloneDoctors = (value: DoctorSummary[]) =>
  value.map((doctor) => ({ ...doctor }));

export function loadOrganizationContext(force = false): Promise<OrganizationContext> {
  return contextCache.load(
    "current",
    () => organizationDirectoryApi.getOrganizationContext(),
    cloneContext,
    force,
  );
}

export function loadManagedCampuses(
  hospitalId: string,
  force = false,
): Promise<CampusSummary[]> {
  return campusCache.load(
    hospitalId,
    () => organizationAdminApi.listCampuses(hospitalId, true),
    cloneCampuses,
    force,
  );
}

export function loadDepartments(
  campusId: string,
  includeDisabled = false,
  force = false,
): Promise<DepartmentSummary[]> {
  const key = `${campusId}:${includeDisabled ? "all" : "active"}`;
  return departmentCache.load(
    key,
    () =>
      includeDisabled
        ? organizationAdminApi.listDepartments(campusId, true)
        : organizationDirectoryApi.listDepartments(campusId),
    cloneDepartments,
    force,
  );
}

export async function loadDepartmentOptions(
  force = false,
): Promise<DepartmentOption[]> {
  const context = await loadOrganizationContext(force);
  const campuses = context.campuses.filter((campus) => campus.status === "active");
  const departmentsByCampus = await Promise.all(
    campuses.map(async (campus) => ({
      campus,
      departments: await loadDepartments(campus.campusId, false, force),
    })),
  );

  return departmentsByCampus.flatMap(({ campus, departments }) =>
    departments
      .filter((department) => department.status === "active")
      .map((department) => ({
        department,
        campus,
        label: `${campus.name} / ${department.name}`,
      })),
  );
}

export function loadDoctors(
  departmentId: string,
  force = false,
): Promise<DoctorSummary[]> {
  return doctorCache.load(
    departmentId,
    () => organizationDirectoryApi.listDoctors(departmentId),
    cloneDoctors,
    force,
  );
}

export function invalidateOrganizationContext() {
  contextCache.invalidate();
}

export function invalidateManagedCampuses(...hospitalIds: Array<string | undefined>) {
  const keys = hospitalIds.filter((value): value is string => Boolean(value));
  campusCache.invalidate(...keys);
}

export function invalidateDepartments(...campusIds: Array<string | undefined>) {
  const ids = campusIds.filter((value): value is string => Boolean(value));
  if (ids.length === 0) {
    departmentCache.invalidate();
    return;
  }
  departmentCache.invalidate(
    ...ids.flatMap((campusId) => [`${campusId}:active`, `${campusId}:all`]),
  );
}

export function invalidateDoctors(...departmentIds: Array<string | undefined>) {
  const ids = departmentIds.filter((value): value is string => Boolean(value));
  doctorCache.invalidate(...ids);
}

export function clearOrganizationCache() {
  contextCache.invalidate();
  campusCache.invalidate();
  departmentCache.invalidate();
  doctorCache.invalidate();
}
