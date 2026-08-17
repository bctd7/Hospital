import type { AppVariant } from "@/types/auth";
import type {
  HomeAction,
  HomeWorkbenchAdapter,
  HomeWorkbenchIdentity,
  HomeWorkbenchView,
} from "@/types/homeWorkbench";

// 首页结构是静态编排；检查项目和预约数据由目标页面通过真实 API 加载。

function patientHome(): HomeWorkbenchView {
  return {
    variant: "patient",
    eyebrow: "医院服务",
    title: "安心就医，从这里开始",
    description: "预约、就诊和查询服务集中在一个首页。",
    primaryAction: {
      id: "create-appointment",
      title: "检查项目",
      description: "查看项目说明、可检查房间和本周时间",
      symbol: "检",
      tone: "cyan",
      badge: "真实数据",
      target: { type: "navigate", url: "/pages/appointment/create/index" },
    },
    secondaryAction: {
      id: "smart-guide",
      title: "智能导诊",
      description: "按准备规则生成本周预约方案",
      symbol: "诊",
      tone: "cyan",
      badge: "智能规划",
      target: { type: "navigate", url: "/pages/guidance/planning/index" },
    },
    serviceGroups: [
      {
        id: "before-visit",
        title: "诊前服务",
        actions: [
          {
            id: "my-appointments",
            title: "我的预约",
            description: "查看预约记录",
            symbol: "约",
            tone: "blue",
            target: { type: "navigate", url: "/pages/profile/appointments/index" },
          },
          {
            id: "patients",
            title: "就诊人管理",
            description: "维护本人信息",
            symbol: "人",
            tone: "violet",
            target: { type: "navigate", url: "/pages/profile/patients/index" },
          },
          {
            id: "walking-route",
            title: "检查导航",
            description: "选择起终点查看步行路线",
            symbol: "导",
            tone: "cyan",
            target: { type: "navigate", url: "/pages/guidance/route/index" },
          },
        ],
      },
      {
        id: "after-visit",
        title: "诊后服务",
        actions: [
          {
            id: "patient-check-records",
            title: "检查记录",
            description: "查看已完成和未到场的检查",
            symbol: "记",
            tone: "blue",
            target: { type: "navigate", url: "/pages/profile/appointments/index?view=completed" },
          },
        ],
      },
    ],
    metrics: [],
    managementActions: [],
    notice: "检查项目、本周可约房间和预约提交均已接入真实 Appointment 数据。",
    mock: false,
  };
}

export function buildHomeWorkbench(
  variant: AppVariant,
  principal: HomeWorkbenchIdentity | null,
): HomeWorkbenchView {
  const view = { ...patientHome(), variant };
  if (variant === "staff") {
    view.primaryAction = {
      id: "manage-examination-resources",
      title: "检查资源管理",
      description: "维护房间、项目关联与每周开放时间",
      symbol: "检",
      tone: "cyan",
      badge: "真实数据",
      target: {
        type: "navigate",
        url: "/pages/admin/appointment/index",
      },
    };
    view.secondaryAction = {
      id: "manage-guidance-rules",
      title: "导诊管理",
      description: "创建检查项目并配置先后与准备规则",
      symbol: "导",
      tone: "cyan",
      badge: "真实数据",
      target: {
        type: "navigate",
        url: "/pages/admin/guidance/index",
      },
    };
    const departmentId = principal?.department_id?.trim() ?? "";
    const departmentBookingAction: HomeAction = {
      id: "department-appointments",
      title: "科室预约",
      description: departmentId ? "查看当前科室预约情况" : "选择科室查看预约情况",
      symbol: "约",
      tone: "blue",
      target: {
        type: "navigate",
        url: departmentId
          ? `/pages/admin/appointment/bookings?department_id=${encodeURIComponent(departmentId)}&department_label=${encodeURIComponent("科室预约")}`
          : "/pages/admin/appointment/bookings",
      },
    };
    const departmentHistoryAction: HomeAction = {
      id: "appointment-history",
      title: "检查记录",
      description: departmentId ? "查看当前科室已完成检查" : "选择科室查看已完成检查",
      symbol: "记",
      tone: "cyan",
      target: {
        type: "navigate",
        url: departmentId
          ? `/pages/admin/appointment/bookings?view=completed&department_id=${encodeURIComponent(departmentId)}&department_label=${encodeURIComponent("检查记录")}`
          : "/pages/admin/appointment/bookings?view=completed&department_label=%E6%A3%80%E6%9F%A5%E8%AE%B0%E5%BD%95",
      },
    };
    view.serviceGroups = view.serviceGroups.map((group) =>
      group.id === "before-visit"
        ? {
            ...group,
            actions: group.actions
              .filter((action) => action.id !== "walking-route" && action.id !== "today-guidance")
              .map((action) =>
                action.id === "my-appointments" ? departmentBookingAction : action,
              ),
          }
        : group.id === "after-visit"
          ? {
              ...group,
              actions: [departmentHistoryAction],
            }
          : group,
    );
    view.notice = "检查资源由 Appointment 管理，项目创建与导诊规则由 Guidance 一次性配置。";
    view.mock = false;
  }
  return view;
}

export const homeWorkbenchAdapter: HomeWorkbenchAdapter = {
  async load(variant, principal) {
    return Promise.resolve(buildHomeWorkbench(variant, principal));
  },
};
