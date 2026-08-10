import type {
  AccountListQuery,
  AdminAccountDetail,
  AdminAccountSummary,
  CampusDraft,
  CampusSummary,
  DepartmentDraft,
  DepartmentSummary,
  DoctorProfileDraft,
  DoctorSummary,
  OrganizationContext,
  PagedResult,
} from "@/types/staffManagement";

import { httpStaffManagementApi } from "./staffManagement.http";

export interface StaffManagementApi {
  getOrganizationContext(): Promise<OrganizationContext>;
  createCampus(input: CampusDraft): Promise<CampusSummary>;
  listDepartments(
    campusId: string,
    includeDisabled?: boolean,
  ): Promise<DepartmentSummary[]>;
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

export const staffManagementApi: StaffManagementApi = httpStaffManagementApi;
