import { describe, expect, it } from "vitest";

import { buildHomeWorkbench } from "@/services/homeWorkbench";
import type { CurrentIdentityResponse } from "@/types/auth";

function principal(roles: string[], permissions: string[]): CurrentIdentityResponse {
  return {
    account_id: "account-1",
    account_type: roles.length ? "staff" : "patient",
    status: "active",
    roles,
    permissions,
    authorization_version: 1,
    department_id: roles.includes("department_doctor") ? "department-1" : undefined,
  };
}

describe("home workbench", () => {
  it("uses a patient examination browser rather than a management label", () => {
    const view = buildHomeWorkbench("patient", principal([], []));

    expect(view.primaryAction?.title).toBe("检查项目");
    expect(view.primaryAction?.target).toEqual({
      type: "navigate",
      url: "/pages/appointment/create/index",
    });
    expect(view.primaryAction?.badge).toBe("真实数据");
    expect(view.notice).toContain("真实 Appointment 数据");
    expect(view.mock).toBe(false);
    expect(view.managementActions).toHaveLength(0);
    expect(view.serviceGroups[0]?.actions[0]).toMatchObject({
      id: "my-appointments",
      title: "我的预约",
      description: "查看预约记录",
      target: {
        type: "navigate",
        url: "/pages/profile/appointments/index",
      },
    });
  });

  it("shows examination item management as the staff primary action", () => {
    const view = buildHomeWorkbench(
      "staff",
      principal(["department_doctor"], ["appointment.read"]),
    );

    expect(view.primaryAction?.title).toBe("检查项目管理");
    expect(view.primaryAction?.description).toBe("维护项目、房间与每周开放时间");
    expect(view.primaryAction?.target).toEqual({
      type: "navigate",
      url: "/pages/admin/appointment/index",
    });
    expect(view.mock).toBe(false);
    expect(view.secondaryAction?.title).toBe("智能导诊");
    expect(view.managementActions).toHaveLength(0);
    expect(view.serviceGroups[0]?.actions[0]).toMatchObject({
      id: "department-appointments",
      title: "科室预约",
      description: "查看当前科室预约情况",
      target: {
        type: "navigate",
        url: "/pages/admin/appointment/bookings?department_id=department-1&department_label=%E7%A7%91%E5%AE%A4%E9%A2%84%E7%BA%A6",
      },
    });
  });

  it("uses the examination management entry for an administrator", () => {
    const view = buildHomeWorkbench(
      "staff",
      principal(["super_admin"], ["identity.account.manage", "appointment.read"]),
    );

    expect(view.primaryAction?.title).toBe("检查项目管理");
    expect(view.managementActions).toHaveLength(0);
    expect(view.serviceGroups[0]?.actions[0]).toMatchObject({
      title: "科室预约",
      description: "选择科室查看预约情况",
      target: { type: "navigate", url: "/pages/admin/appointment/index" },
    });
    expect(view.serviceGroups.map((group) => group.title)).toEqual([
      "诊前服务",
      "诊中服务",
      "诊后服务",
    ]);
  });
});
