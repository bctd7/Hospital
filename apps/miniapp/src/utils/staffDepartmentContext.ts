const STAFF_DEPARTMENT_STORAGE_KEY = "hospital:staff-department";

export function currentStaffDepartmentId(): string {
  try {
    const value = uni.getStorageSync(STAFF_DEPARTMENT_STORAGE_KEY);
    return typeof value === "string" ? value.trim() : "";
  } catch {
    return "";
  }
}

export function rememberStaffDepartmentId(departmentId: string) {
  const normalized = departmentId.trim();
  if (!normalized) return;
  try {
    uni.setStorageSync(STAFF_DEPARTMENT_STORAGE_KEY, normalized);
  } catch {
    // 当前页面内的选择仍然有效，本地持久化失败不阻塞业务。
  }
}
