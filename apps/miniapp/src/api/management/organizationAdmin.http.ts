import { ApiError, request } from "@/api/client";
import type { OrganizationAdminApi } from "@/api/management/contracts";
import { operationId, queryPath } from "@/api/management/httpShared";
import {
  departmentFromResponse,
  httpOrganizationDirectoryApi,
} from "@/api/management/organizationDirectory.http";
import type { CampusSummary, DepartmentSummary } from "@/types/staffManagement";

interface OrganizationUnitResponse {
  unit_id: string;
  parent_id: string;
  code: string;
  name: string;
  child_count?: number;
  doctor_count?: number;
  status: "active" | "disabled";
  version: number;
}

function campusFromResponse(value: OrganizationUnitResponse): CampusSummary {
  return {
    campusId: value.unit_id,
    hospitalId: value.parent_id,
    code: value.code,
    name: value.name,
    departmentCount: value.child_count ?? 0,
    status: value.status,
    version: value.version,
  };
}

function managedDepartmentFromResponse(value: OrganizationUnitResponse): DepartmentSummary {
  return departmentFromResponse({
    department_id: value.unit_id,
    parent_id: value.parent_id,
    code: value.code,
    name: value.name,
    doctor_count: value.doctor_count,
    status: value.status,
    version: value.version,
  });
}

export const httpOrganizationAdminApi: OrganizationAdminApi = {
  async createCampus(input) {
    const response = await request<OrganizationUnitResponse>({
      path: "/api/v1/admin/identity/organization-units",
      method: "POST",
      authenticated: true,
      data: {
        unit_type: "campus",
        parent_id: input.hospitalId,
        name: input.name,
        operation_id: operationId(),
      },
    });
    return campusFromResponse(response);
  },

  async listCampuses(hospitalId, includeDisabled = false) {
    if (!hospitalId) {
      throw new ApiError("医院信息尚未加载", 400);
    }
    if (!includeDisabled) {
      return (await httpOrganizationDirectoryApi.getOrganizationContext()).campuses;
    }
    const response = await request<{ items: OrganizationUnitResponse[] }>({
      path: queryPath("/api/v1/admin/identity/organization-units", {
        unit_type: "campus",
        parent_id: hospitalId,
        status: "all",
      }),
      authenticated: true,
    });
    return response.items.map(campusFromResponse);
  },

  async updateCampus(campusId, input, version) {
    const response = await request<OrganizationUnitResponse>({
      path: `/api/v1/admin/identity/organization-units/${encodeURIComponent(campusId)}`,
      method: "PUT",
      authenticated: true,
      data: { name: input.name, version, operation_id: operationId() },
    });
    return campusFromResponse(response);
  },

  async setCampusEnabled(campusId, enabled, version) {
    const response = await request<OrganizationUnitResponse>({
      path: `/api/v1/admin/identity/organization-units/${encodeURIComponent(campusId)}/${enabled ? "enable" : "disable"}`,
      method: "POST",
      authenticated: true,
      data: { version, operation_id: operationId() },
    });
    return campusFromResponse(response);
  },

  async listDepartments(campusId, includeDisabled = false) {
    if (!campusId) {
      throw new ApiError("请先选择院区", 400);
    }
    if (!includeDisabled) {
      return httpOrganizationDirectoryApi.listDepartments(campusId);
    }
    const path = queryPath("/api/v1/admin/identity/organization-units", {
      unit_type: "department",
      parent_id: campusId,
    });
    const response = await request<{ items: OrganizationUnitResponse[] }>({
      path: `${path}&status=all`,
      authenticated: true,
    });
    return response.items.map(managedDepartmentFromResponse);
  },

  async createDepartment(input) {
    if (!input.parentId) {
      throw new ApiError("请先选择院区", 400);
    }
    const response = await request<OrganizationUnitResponse>({
      path: "/api/v1/admin/identity/organization-units",
      method: "POST",
      authenticated: true,
      data: {
        unit_type: "department",
        parent_id: input.parentId,
        name: input.name,
        operation_id: operationId(),
      },
    });
    return managedDepartmentFromResponse(response);
  },

  async updateDepartment(departmentId, input, version) {
    const response = await request<OrganizationUnitResponse>({
      path: `/api/v1/admin/identity/organization-units/${encodeURIComponent(departmentId)}`,
      method: "PUT",
      authenticated: true,
      data: {
        name: input.name,
        parent_id: input.parentId,
        version,
        operation_id: operationId(),
      },
    });
    return managedDepartmentFromResponse(response);
  },

  async setDepartmentEnabled(departmentId, enabled, version) {
    const response = await request<OrganizationUnitResponse>({
      path: `/api/v1/admin/identity/organization-units/${encodeURIComponent(departmentId)}/${enabled ? "enable" : "disable"}`,
      method: "POST",
      authenticated: true,
      data: { version, operation_id: operationId() },
    });
    return managedDepartmentFromResponse(response);
  },
};
