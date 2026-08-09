import { STAFF_DATA_SOURCE } from "@/config/environment";
import type {
  AccountListQuery,
  AdminAccountDetail,
  AdminAccountSummary,
  DepartmentDraft,
  DepartmentSummary,
  DoctorProfileDraft,
  DoctorSummary,
  PagedResult,
} from "@/types/staffManagement";

import { httpStaffManagementApi } from "./staffManagement.http";
import { mockStaffManagementApi } from "./staffManagement.mock";

export interface StaffManagementApi {
  listDepartments(includeDisabled?: boolean): Promise<DepartmentSummary[]>;
  listDoctors(departmentId: string): Promise<DoctorSummary[]>;
  createDepartment(input: DepartmentDraft): Promise<DepartmentSummary>;
  updateDepartment(
    departmentId: string,
    input: DepartmentDraft,
    version: number,
  ): Promise<DepartmentSummary>;
  setDepartmentEnabled(
    departmentId: string,
    enabled: boolean,
    version: number,
  ): Promise<DepartmentSummary>;

  listAccounts(query: AccountListQuery): Promise<PagedResult<AdminAccountSummary>>;
  searchAccountByPhone(phone: string): Promise<AdminAccountSummary | null>;
  getAccount(accountId: string): Promise<AdminAccountDetail>;
  promoteDoctor(
    accountId: string,
    departmentId: string,
    profile: DoctorProfileDraft,
    managementVersion: number,
  ): Promise<AdminAccountDetail>;
  updateDoctor(
    accountId: string,
    profile: DoctorProfileDraft,
    managementVersion: number,
  ): Promise<AdminAccountDetail>;
  changeDoctorDepartment(
    accountId: string,
    departmentId: string,
    managementVersion: number,
  ): Promise<AdminAccountDetail>;
  revokeDoctor(
    accountId: string,
    managementVersion: number,
  ): Promise<AdminAccountDetail>;
  setAccountEnabled(
    accountId: string,
    enabled: boolean,
    managementVersion: number,
  ): Promise<AdminAccountDetail>;
}

export const STAFF_MANAGEMENT_USES_MOCK = STAFF_DATA_SOURCE === "mock";
export const staffManagementApi: StaffManagementApi = STAFF_MANAGEMENT_USES_MOCK
  ? mockStaffManagementApi
  : httpStaffManagementApi;
