import { request } from "@/api/client";
import type { OrganizationDirectoryApi } from "@/api/management/contracts";
import type {
  DepartmentSummary,
  DoctorSummary,
  OrganizationContext,
} from "@/types/staffManagement";

interface DepartmentResponse {
  department_id: string;
  campus_id?: string;
  parent_id?: string;
  code: string;
  name: string;
  doctor_count?: number;
  status: "active" | "disabled";
  version: number;
}

interface OrganizationContextResponse {
  hospital: {
    hospital_id: string;
    code: string;
    name: string;
    version: number;
  };
  campuses: Array<{
    campus_id: string;
    hospital_id: string;
    code: string;
    name: string;
    department_count: number;
    status: "active" | "disabled";
    version: number;
  }>;
}

interface DoctorResponse {
  account_id: string;
  display_name: string;
  department_id: string;
  avatar_url?: string;
  description?: string;
  version: number;
}

function organizationContextFromResponse(
  value: OrganizationContextResponse,
): OrganizationContext {
  return {
    hospital: {
      hospitalId: value.hospital.hospital_id,
      code: value.hospital.code,
      name: value.hospital.name,
      version: value.hospital.version,
    },
    campuses: value.campuses.map((campus) => ({
      campusId: campus.campus_id,
      hospitalId: campus.hospital_id,
      code: campus.code,
      name: campus.name,
      departmentCount: campus.department_count,
      status: campus.status,
      version: campus.version,
    })),
  };
}

export function departmentFromResponse(value: DepartmentResponse): DepartmentSummary {
  return {
    departmentId: value.department_id,
    parentId: value.parent_id ?? value.campus_id,
    code: value.code,
    name: value.name,
    doctorCount: value.doctor_count ?? 0,
    status: value.status,
    version: value.version,
  };
}

function doctorFromResponse(value: DoctorResponse): DoctorSummary {
  return {
    accountId: value.account_id,
    displayName: value.display_name,
    departmentId: value.department_id,
    avatarUrl: value.avatar_url,
    description: value.description,
    version: value.version,
  };
}

export const httpOrganizationDirectoryApi: OrganizationDirectoryApi = {
  async getOrganizationContext() {
    const response = await request<OrganizationContextResponse>({
      path: "/api/v1/directory/organization-context",
    });
    return organizationContextFromResponse(response);
  },

  async listDepartments(campusId) {
    const response = await request<{ items: DepartmentResponse[] }>({
      path: `/api/v1/directory/departments?campus_id=${encodeURIComponent(campusId)}`,
      authenticated: false,
    });
    return response.items.map(departmentFromResponse);
  },

  async listDoctors(departmentId) {
    const response = await request<{
      items: DoctorResponse[];
      page: number;
      page_size: number;
      total: number;
    }>({
      path: `/api/v1/directory/departments/${encodeURIComponent(departmentId)}/doctors?page=1&page_size=100`,
    });
    return response.items.map(doctorFromResponse);
  },
};
