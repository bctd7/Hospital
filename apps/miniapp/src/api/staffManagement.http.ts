import { request } from "@/api/client";
import type { StaffManagementApi } from "@/api/staffManagement";
import type {
  AccountListQuery,
  AdminAccountAction,
  AdminAccountDetail,
  AdminAccountSummary,
  DepartmentDraft,
  DepartmentSummary,
  DoctorProfileDraft,
  DoctorSummary,
  PagedResult,
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
  campuses: Array<{
    campus_id: string;
  }>;
}

interface DoctorResponse {
  doctor_id: string;
  display_name: string;
  department_id: string;
  avatar_url?: string;
  description?: string;
  version: number;
}

interface AccountResponse {
  account_id: string;
  nickname?: string;
  display_name?: string;
  avatar_url?: string;
  masked_phone?: string;
  account_status: "active" | "disabled";
  identity_type: "patient" | "doctor" | "super_admin";
  department_id?: string;
  department_name?: string;
  management_version: number;
  phone_verification_status?: string;
  staff_no?: string;
  description?: string;
  roles?: string[];
  authorization_version?: number;
  available_actions?: AdminAccountAction[];
  created_at?: string;
  updated_at?: string;
}

interface PagedResponse<T> {
  items: T[];
  page: number;
  page_size: number;
  total: number;
}

function operationId(): string {
  return `miniapp-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function queryPath(path: string, values: Record<string, unknown>): string {
  const query = Object.entries(values)
    .filter(([, value]) => value !== undefined && value !== "" && value !== "all")
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join("&");
  return query ? `${path}?${query}` : path;
}

function departmentFromResponse(value: DepartmentResponse): DepartmentSummary {
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
    doctorId: value.doctor_id,
    displayName: value.display_name,
    departmentId: value.department_id,
    avatarUrl: value.avatar_url,
    description: value.description,
    version: value.version,
  };
}

function accountSummaryFromResponse(value: AccountResponse): AdminAccountSummary {
  return {
    accountId: value.account_id,
    nickname: value.nickname,
    displayName: value.display_name,
    avatarUrl: value.avatar_url,
    maskedPhone: value.masked_phone,
    accountStatus: value.account_status,
    identityType: value.identity_type,
    departmentId: value.department_id,
    departmentName: value.department_name,
    managementVersion: value.management_version,
  };
}

function accountDetailFromResponse(value: AccountResponse): AdminAccountDetail {
  return {
    ...accountSummaryFromResponse(value),
    phoneVerificationStatus: value.phone_verification_status,
    staffNo: value.staff_no,
    description: value.description,
    roles: value.roles ?? [],
    authorizationVersion: value.authorization_version ?? 0,
    availableActions: value.available_actions ?? [],
    createdAt: value.created_at ?? "",
    updatedAt: value.updated_at ?? "",
  };
}

export const httpStaffManagementApi: StaffManagementApi = {
  async listDepartments(includeDisabled = false) {
    const context = await request<OrganizationContextResponse>({
      path: "/api/v1/directory/organization-context",
    });
    const responses = await Promise.all(
      context.campuses.map((campus) =>
        request<{ items: DepartmentResponse[] }>({
          path: includeDisabled
            ? `${queryPath("/api/v1/admin/identity/organization-units", {
                unit_type: "department",
                parent_id: campus.campus_id,
              })}&status=all`
            : queryPath("/api/v1/directory/departments", {
                campus_id: campus.campus_id,
              }),
          authenticated: includeDisabled,
        }),
      ),
    );
    return responses.flatMap((response) => response.items.map(departmentFromResponse));
  },

  async listDoctors(departmentId) {
    const response = await request<PagedResponse<DoctorResponse>>({
      path: `/api/v1/directory/departments/${encodeURIComponent(departmentId)}/doctors?page=1&page_size=100`,
    });
    return response.items.map(doctorFromResponse);
  },

  async createDepartment(input) {
    const response = await request<DepartmentResponse>({
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
    return departmentFromResponse(response);
  },

  async updateDepartment(departmentId, input, version) {
    const response = await request<DepartmentResponse>({
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
    return departmentFromResponse(response);
  },

  async setDepartmentEnabled(departmentId, enabled, version) {
    const response = await request<DepartmentResponse>({
      path: enabled
        ? `/api/v1/admin/identity/organization-units/${encodeURIComponent(departmentId)}/enable`
        : `/api/v1/admin/identity/organization-units/${encodeURIComponent(departmentId)}`,
      method: enabled ? "POST" : "DELETE",
      authenticated: true,
      data: { version, operation_id: operationId() },
    });
    return departmentFromResponse(response);
  },

  async listAccounts(query: AccountListQuery) {
    const response = await request<PagedResponse<AccountResponse>>({
      path: queryPath("/api/v1/admin/identity/accounts", {
        page: query.page ?? 1,
        page_size: query.pageSize ?? 20,
        nickname: query.nickname,
        identity_type: query.identityType,
        status: query.status,
        department_id: query.departmentId,
      }),
      authenticated: true,
    });
    return {
      items: response.items.map(accountSummaryFromResponse),
      page: response.page,
      pageSize: response.page_size,
      total: response.total,
    };
  },

  async searchAccountByPhone(phone) {
    try {
      const response = await request<{ identity: AccountResponse }>({
        path: "/api/v1/admin/identity/accounts/search-by-phone",
        method: "POST",
        authenticated: true,
        data: { phone },
      });
      return accountSummaryFromResponse(response.identity);
    } catch (error) {
      if (error instanceof Error && "statusCode" in error && error.statusCode === 404) {
        return null;
      }
      throw error;
    }
  },

  async getAccount(accountId) {
    const response = await request<AccountResponse>({
      path: `/api/v1/admin/identity/accounts/${encodeURIComponent(accountId)}`,
      authenticated: true,
    });
    return accountDetailFromResponse(response);
  },

  async promoteDoctor(accountId, departmentId, profile, managementVersion) {
    const response = await request<AccountResponse>({
      path: "/api/v1/admin/identity/doctors/promote",
      method: "POST",
      authenticated: true,
      data: {
        account_id: accountId,
        department_id: departmentId,
        display_name: profile.displayName,
        staff_no: profile.staffNo,
        avatar_url: profile.avatarUrl,
        description: profile.description,
        management_version: managementVersion,
        offline_verified: true,
        operation_id: operationId(),
      },
    });
    return accountDetailFromResponse(response);
  },

  async updateDoctor(accountId, profile, managementVersion) {
    const response = await request<AccountResponse>({
      path: `/api/v1/admin/identity/doctors/${encodeURIComponent(accountId)}`,
      method: "PUT",
      authenticated: true,
      data: {
        display_name: profile.displayName,
        staff_no: profile.staffNo,
        avatar_url: profile.avatarUrl,
        description: profile.description,
        management_version: managementVersion,
        operation_id: operationId(),
      },
    });
    return accountDetailFromResponse(response);
  },

  async changeDoctorDepartment(accountId, departmentId, managementVersion) {
    const response = await request<AccountResponse>({
      path: `/api/v1/admin/identity/doctors/${encodeURIComponent(accountId)}/department`,
      method: "PUT",
      authenticated: true,
      data: {
        department_id: departmentId,
        management_version: managementVersion,
        operation_id: operationId(),
      },
    });
    return accountDetailFromResponse(response);
  },

  async revokeDoctor(accountId, managementVersion) {
    const response = await request<AccountResponse>({
      path: `/api/v1/admin/identity/doctors/${encodeURIComponent(accountId)}`,
      method: "DELETE",
      authenticated: true,
      data: { management_version: managementVersion, operation_id: operationId() },
    });
    return accountDetailFromResponse(response);
  },

  async setAccountEnabled(accountId, enabled, managementVersion) {
    const response = await request<AccountResponse>({
      path: `/api/v1/admin/identity/accounts/${encodeURIComponent(accountId)}/${enabled ? "enable" : "disable"}`,
      method: "POST",
      authenticated: true,
      data: { management_version: managementVersion, operation_id: operationId() },
    });
    return accountDetailFromResponse(response);
  },
};
