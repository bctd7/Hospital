import { request } from "@/api/client";
import type { IdentityAdminApi } from "@/api/management/contracts";
import { operationId, queryPath, type PagedResponse } from "@/api/management/httpShared";
import type {
  AccountListQuery,
  AdminAccountAction,
  AdminAccountDetail,
  AdminAccountSummary,
} from "@/types/staffManagement";

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
  staff_status?: "active" | "revoked";
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
    staffStatus: value.staff_status,
  };
}

export const httpIdentityAdminApi: IdentityAdminApi = {
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
      path: `/api/v1/admin/identity/accounts/${encodeURIComponent(accountId)}/promote-doctor`,
      method: "POST",
      authenticated: true,
      data: {
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
      path: `/api/v1/admin/identity/doctors/${encodeURIComponent(accountId)}/revoke`,
      method: "POST",
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
