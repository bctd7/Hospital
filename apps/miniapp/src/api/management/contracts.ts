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

export interface OrganizationDirectoryApi {
  getOrganizationContext(): Promise<OrganizationContext>;
  listDepartments(campusId: string): Promise<DepartmentSummary[]>;
  listDoctors(departmentId: string): Promise<DoctorSummary[]>;
}

export interface OrganizationAdminApi {
  createCampus(input: CampusDraft): Promise<CampusSummary>;
  listCampuses(hospitalId: string, includeDisabled?: boolean): Promise<CampusSummary[]>;
  updateCampus(campusId: string, input: CampusDraft, version: number): Promise<CampusSummary>;
  setCampusEnabled(campusId: string, enabled: boolean, version: number): Promise<CampusSummary>;
  listDepartments(campusId: string, includeDisabled?: boolean): Promise<DepartmentSummary[]>;
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
}

export interface IdentityAdminApi {
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
  revokeDoctor(accountId: string, managementVersion: number): Promise<AdminAccountDetail>;
  setAccountEnabled(
    accountId: string,
    enabled: boolean,
    managementVersion: number,
  ): Promise<AdminAccountDetail>;
}
