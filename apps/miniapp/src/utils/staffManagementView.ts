import type { DepartmentSummary } from "@/types/staffManagement";

/** 保证小程序组件在异步数据到达前也收到结构完整的展示对象。 */
export function createEmptyDepartment(): DepartmentSummary {
  return {
    departmentId: "",
    code: "",
    name: "",
    doctorCount: 0,
    status: "active",
    version: 0,
  };
}
