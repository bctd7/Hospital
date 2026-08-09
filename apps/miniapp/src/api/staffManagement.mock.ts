import type { StaffManagementApi } from "@/api/staffManagement";
import type {
  AccountIdentityType,
  AdminAccountAction,
  AdminAccountDetail,
  AdminAccountSummary,
  DepartmentSummary,
  DoctorProfileDraft,
} from "@/types/staffManagement";

interface MockAccount extends AdminAccountDetail {
  fullPhone: string;
}

const NOW = "2026-08-10T09:00:00+08:00";

const initialDepartments: DepartmentSummary[] = [
  {
    departmentId: "dept-radiology",
    parentId: "campus-main",
    code: "RAD",
    name: "放射科",
    doctorCount: 0,
    status: "active",
    version: 1,
  },
  {
    departmentId: "dept-ultrasound",
    parentId: "campus-main",
    code: "US",
    name: "超声医学科",
    doctorCount: 0,
    status: "active",
    version: 1,
  },
  {
    departmentId: "dept-laboratory",
    parentId: "campus-main",
    code: "LAB",
    name: "检验科",
    doctorCount: 0,
    status: "active",
    version: 1,
  },
  {
    departmentId: "dept-rehabilitation",
    parentId: "campus-main",
    code: "REHAB",
    name: "康复科",
    doctorCount: 0,
    status: "disabled",
    version: 2,
  },
];

const initialAccounts: MockAccount[] = [
  {
    accountId: "account-admin-001",
    fullPhone: "13900000001",
    nickname: "系统管理员",
    maskedPhone: "139****0001",
    accountStatus: "active",
    identityType: "super_admin",
    managementVersion: 1,
    phoneVerificationStatus: "verified",
    roles: ["super_admin"],
    authorizationVersion: 3,
    availableActions: [],
    createdAt: NOW,
    updatedAt: NOW,
  },
  {
    accountId: "account-doctor-lin",
    fullPhone: "13800138001",
    nickname: "林木木",
    displayName: "林医生",
    maskedPhone: "138****8001",
    accountStatus: "active",
    identityType: "doctor",
    departmentId: "dept-radiology",
    departmentName: "放射科",
    managementVersion: 3,
    phoneVerificationStatus: "verified",
    staffNo: "D001",
    description: "擅长常规影像检查与报告解读",
    roles: ["department_doctor"],
    authorizationVersion: 2,
    availableActions: [],
    createdAt: NOW,
    updatedAt: NOW,
  },
  {
    accountId: "account-doctor-li",
    fullPhone: "13800138002",
    nickname: "小李",
    displayName: "李医生",
    maskedPhone: "138****8002",
    accountStatus: "active",
    identityType: "doctor",
    departmentId: "dept-ultrasound",
    departmentName: "超声医学科",
    managementVersion: 2,
    phoneVerificationStatus: "verified",
    staffNo: "D002",
    description: "擅长腹部及浅表器官超声检查",
    roles: ["department_doctor"],
    authorizationVersion: 2,
    availableActions: [],
    createdAt: NOW,
    updatedAt: NOW,
  },
  {
    accountId: "account-doctor-zhou",
    fullPhone: "13800138003",
    nickname: "周周",
    displayName: "周医生",
    maskedPhone: "138****8003",
    accountStatus: "active",
    identityType: "doctor",
    departmentId: "dept-laboratory",
    departmentName: "检验科",
    managementVersion: 2,
    phoneVerificationStatus: "verified",
    staffNo: "D003",
    description: "负责临床检验结果审核",
    roles: ["department_doctor"],
    authorizationVersion: 2,
    availableActions: [],
    createdAt: NOW,
    updatedAt: NOW,
  },
  {
    accountId: "account-patient-wang",
    fullPhone: "13800138004",
    nickname: "王小雨",
    maskedPhone: "138****8004",
    accountStatus: "active",
    identityType: "patient",
    managementVersion: 1,
    phoneVerificationStatus: "verified",
    roles: [],
    authorizationVersion: 1,
    availableActions: [],
    createdAt: NOW,
    updatedAt: NOW,
  },
  {
    accountId: "account-patient-chen",
    fullPhone: "13800138005",
    nickname: "陈晨",
    maskedPhone: "138****8005",
    accountStatus: "disabled",
    identityType: "patient",
    managementVersion: 2,
    phoneVerificationStatus: "verified",
    roles: [],
    authorizationVersion: 2,
    availableActions: [],
    createdAt: NOW,
    updatedAt: NOW,
  },
];

let departments: DepartmentSummary[] = [];
let accounts: MockAccount[] = [];
let initialized = false;

function copyDepartment(value: DepartmentSummary): DepartmentSummary {
  return { ...value };
}

function copyAccount(value: MockAccount): MockAccount {
  return {
    ...value,
    roles: [...value.roles],
    availableActions: [...value.availableActions],
  };
}

export function resetMockStaffManagement() {
  departments = initialDepartments.map(copyDepartment);
  accounts = initialAccounts.map(copyAccount);
  initialized = true;
}

function ensureInitialized() {
  if (!initialized) {
    resetMockStaffManagement();
  }
}

function departmentName(departmentId?: string): string | undefined {
  return departments.find((item) => item.departmentId === departmentId)?.name;
}

function availableActions(account: MockAccount): AdminAccountAction[] {
  if (account.identityType === "super_admin") {
    return [];
  }
  if (account.accountStatus === "disabled") {
    return ["enable_account"];
  }
  if (account.identityType === "doctor") {
    return [
      "edit_doctor",
      "change_department",
      "revoke_doctor",
      "disable_account",
    ];
  }
  return ["promote_doctor", "disable_account"];
}

function withComputedAccount(account: MockAccount): MockAccount {
  return {
    ...copyAccount(account),
    departmentName: departmentName(account.departmentId),
    availableActions: availableActions(account),
  };
}

function publicAccount(account: MockAccount): AdminAccountSummary {
  const computed = withComputedAccount(account);
  return {
    accountId: computed.accountId,
    nickname: computed.nickname,
    displayName: computed.displayName,
    avatarUrl: computed.avatarUrl,
    maskedPhone: computed.maskedPhone,
    accountStatus: computed.accountStatus,
    identityType: computed.identityType,
    departmentId: computed.departmentId,
    departmentName: computed.departmentName,
    managementVersion: computed.managementVersion,
  };
}

function findAccount(accountId: string): MockAccount {
  const account = accounts.find((item) => item.accountId === accountId);
  if (!account) {
    throw new Error("没有找到该用户");
  }
  return account;
}

function assertVersion(account: MockAccount, managementVersion: number) {
  if (account.managementVersion !== managementVersion) {
    throw new Error("用户资料已发生变化，请刷新后重试");
  }
}

function touchAccount(account: MockAccount, authorizationChanged: boolean) {
  account.managementVersion += 1;
  if (authorizationChanged) {
    account.authorizationVersion += 1;
  }
  account.updatedAt = new Date().toISOString();
}

function doctorCount(departmentId: string): number {
  return accounts.filter(
    (account) =>
      account.identityType === "doctor" &&
      account.accountStatus === "active" &&
      account.departmentId === departmentId,
  ).length;
}

function normalizePhone(phone: string): string {
  return phone.replace(/[\s-]/g, "");
}

function matchesIdentity(
  actual: AccountIdentityType,
  expected?: AccountIdentityType | "all",
): boolean {
  return !expected || expected === "all" || actual === expected;
}

export const mockStaffManagementApi: StaffManagementApi = {
  async listDepartments(includeDisabled = false) {
    ensureInitialized();
    return departments
      .filter((item) => includeDisabled || item.status === "active")
      .map((item) => ({ ...copyDepartment(item), doctorCount: doctorCount(item.departmentId) }));
  },

  async listDoctors(departmentId) {
    ensureInitialized();
    return accounts
      .filter(
        (account) =>
          account.identityType === "doctor" &&
          account.accountStatus === "active" &&
          account.departmentId === departmentId,
      )
      .map((account) => ({
        doctorId: account.accountId,
        displayName: account.displayName ?? account.nickname ?? "未命名医生",
        departmentId,
        avatarUrl: account.avatarUrl,
        description: account.description,
        version: account.managementVersion,
      }));
  },

  async createDepartment(input) {
    ensureInitialized();
    const name = input.name.trim();
    if (!name) {
      throw new Error("请输入部门名称");
    }
    if (departments.some((item) => item.name === name && item.status === "active")) {
      throw new Error("已存在同名部门");
    }
    const sequence = departments.length + 1;
    const department: DepartmentSummary = {
      departmentId: `dept-mock-${Date.now()}`,
      parentId: input.parentId ?? "campus-main",
      code: `MOCK-${String(sequence).padStart(2, "0")}`,
      name,
      doctorCount: 0,
      status: "active",
      version: 1,
    };
    departments.push(department);
    return copyDepartment(department);
  },

  async updateDepartment(departmentId, input, version) {
    ensureInitialized();
    const department = departments.find((item) => item.departmentId === departmentId);
    if (!department) {
      throw new Error("没有找到该部门");
    }
    if (department.version !== version) {
      throw new Error("部门资料已变化，请刷新后重试");
    }
    const name = input.name.trim();
    if (!name) {
      throw new Error("请输入部门名称");
    }
    department.name = name;
    department.parentId = input.parentId ?? department.parentId;
    department.version += 1;
    accounts.forEach((account) => {
      if (account.departmentId === departmentId) {
        account.departmentName = name;
      }
    });
    return { ...copyDepartment(department), doctorCount: doctorCount(departmentId) };
  },

  async setDepartmentEnabled(departmentId, enabled, version) {
    ensureInitialized();
    const department = departments.find((item) => item.departmentId === departmentId);
    if (!department) {
      throw new Error("没有找到该部门");
    }
    if (department.version !== version) {
      throw new Error("部门资料已变化，请刷新后重试");
    }
    if (!enabled && doctorCount(departmentId) > 0) {
      throw new Error("该部门仍有有效医生，请先调岗或撤销医生身份");
    }
    department.status = enabled ? "active" : "disabled";
    department.version += 1;
    return { ...copyDepartment(department), doctorCount: doctorCount(departmentId) };
  },

  async listAccounts(query) {
    ensureInitialized();
    const page = Math.max(1, query.page ?? 1);
    const pageSize = Math.min(50, Math.max(1, query.pageSize ?? 20));
    const nickname = query.nickname?.trim().toLocaleLowerCase() ?? "";
    const filtered = accounts.filter((account) => {
      const searchable = `${account.nickname ?? ""} ${account.displayName ?? ""}`
        .trim()
        .toLocaleLowerCase();
      return (
        (!nickname || searchable.includes(nickname)) &&
        matchesIdentity(account.identityType, query.identityType) &&
        (!query.status || query.status === "all" || account.accountStatus === query.status) &&
        (!query.departmentId || account.departmentId === query.departmentId)
      );
    });
    const start = (page - 1) * pageSize;
    return {
      items: filtered.slice(start, start + pageSize).map(publicAccount),
      page,
      pageSize,
      total: filtered.length,
    };
  },

  async searchAccountByPhone(phone) {
    ensureInitialized();
    const normalized = normalizePhone(phone);
    const account = accounts.find((item) => item.fullPhone === normalized);
    return account ? publicAccount(account) : null;
  },

  async getAccount(accountId) {
    ensureInitialized();
    return withComputedAccount(findAccount(accountId));
  },

  async promoteDoctor(accountId, departmentId, profile, managementVersion) {
    ensureInitialized();
    const account = findAccount(accountId);
    assertVersion(account, managementVersion);
    const department = departments.find(
      (item) => item.departmentId === departmentId && item.status === "active",
    );
    if (!department) {
      throw new Error("目标部门不可用");
    }
    if (account.identityType !== "patient") {
      throw new Error("该账号不能重复开通医生身份");
    }
    account.identityType = "doctor";
    account.roles = ["department_doctor"];
    account.departmentId = departmentId;
    account.departmentName = department.name;
    account.displayName = profile.displayName.trim() || account.nickname || "未命名医生";
    account.staffNo = profile.staffNo?.trim();
    account.avatarUrl = profile.avatarUrl?.trim();
    account.description = profile.description?.trim();
    touchAccount(account, true);
    return withComputedAccount(account);
  },

  async updateDoctor(accountId, profile: DoctorProfileDraft, managementVersion) {
    ensureInitialized();
    const account = findAccount(accountId);
    assertVersion(account, managementVersion);
    if (account.identityType !== "doctor") {
      throw new Error("该账号当前不是医生");
    }
    account.displayName = profile.displayName.trim() || account.displayName;
    account.staffNo = profile.staffNo?.trim();
    account.avatarUrl = profile.avatarUrl?.trim();
    account.description = profile.description?.trim();
    touchAccount(account, false);
    return withComputedAccount(account);
  },

  async changeDoctorDepartment(accountId, departmentId, managementVersion) {
    ensureInitialized();
    const account = findAccount(accountId);
    assertVersion(account, managementVersion);
    const department = departments.find(
      (item) => item.departmentId === departmentId && item.status === "active",
    );
    if (!department) {
      throw new Error("目标部门不可用");
    }
    if (account.identityType !== "doctor") {
      throw new Error("该账号当前不是医生");
    }
    account.departmentId = departmentId;
    account.departmentName = department.name;
    touchAccount(account, true);
    return withComputedAccount(account);
  },

  async revokeDoctor(accountId, managementVersion) {
    ensureInitialized();
    const account = findAccount(accountId);
    assertVersion(account, managementVersion);
    if (account.identityType !== "doctor") {
      throw new Error("该账号当前不是医生");
    }
    account.identityType = "patient";
    account.roles = [];
    account.departmentId = undefined;
    account.departmentName = undefined;
    account.displayName = undefined;
    account.staffNo = undefined;
    account.description = undefined;
    touchAccount(account, true);
    return withComputedAccount(account);
  },

  async setAccountEnabled(accountId, enabled, managementVersion) {
    ensureInitialized();
    const account = findAccount(accountId);
    assertVersion(account, managementVersion);
    if (account.identityType === "super_admin") {
      throw new Error("首版不允许变更超级管理员账号状态");
    }
    account.accountStatus = enabled ? "active" : "disabled";
    touchAccount(account, true);
    return withComputedAccount(account);
  },
};
