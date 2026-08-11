export type DepartmentStatus = "active" | "disabled";
export type AccountStatus = "active" | "disabled";
export type AccountIdentityType = "patient" | "doctor" | "super_admin";

export type AdminAccountAction =
  | "promote_doctor"
  | "edit_doctor"
  | "change_department"
  | "revoke_doctor"
  | "disable_account"
  | "enable_account";

export interface DepartmentSummary {
  departmentId: string;
  parentId?: string;
  code: string;
  name: string;
  doctorCount: number;
  status: DepartmentStatus;
  version: number;
}

export interface HospitalSummary {
  hospitalId: string;
  code: string;
  name: string;
  version: number;
}

export interface CampusSummary {
  campusId: string;
  hospitalId: string;
  code: string;
  name: string;
  departmentCount: number;
  status: DepartmentStatus;
  version: number;
}

export interface OrganizationContext {
  hospital: HospitalSummary;
  campuses: CampusSummary[];
}

export interface DoctorSummary {
  doctorId: string;
  displayName: string;
  departmentId: string;
  avatarUrl?: string;
  description?: string;
  version: number;
}

export interface AdminAccountSummary {
  accountId: string;
  nickname?: string;
  displayName?: string;
  avatarUrl?: string;
  maskedPhone?: string;
  accountStatus: AccountStatus;
  identityType: AccountIdentityType;
  departmentId?: string;
  departmentName?: string;
  managementVersion: number;
}

export interface AdminAccountDetail extends AdminAccountSummary {
  phoneVerificationStatus?: string;
  staffNo?: string;
  description?: string;
  roles: string[];
  authorizationVersion: number;
  availableActions: AdminAccountAction[];
  createdAt: string;
  updatedAt: string;
  staffStatus?: "active" | "revoked";
}

export interface PagedResult<T> {
  items: T[];
  page: number;
  pageSize: number;
  total: number;
}

export interface AccountListQuery {
  page?: number;
  pageSize?: number;
  nickname?: string;
  identityType?: AccountIdentityType | "all";
  status?: AccountStatus | "all";
  departmentId?: string;
}

export interface DepartmentDraft {
  name: string;
  parentId?: string;
}

export interface CampusDraft {
  name: string;
  hospitalId: string;
}

export interface DoctorProfileDraft {
  displayName: string;
  staffNo?: string;
  avatarUrl?: string;
  description?: string;
}
