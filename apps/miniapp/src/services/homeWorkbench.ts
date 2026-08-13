import type { AppVariant } from "@/types/auth";
import type {
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
      description: "智能导诊",
      symbol: "诊",
      tone: "cyan",
      badge: "建设中",
      target: { type: "unavailable", message: "智能导诊功能正在建设中" },
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
            id: "insurance-profile",
            title: "医保在线建档",
            description: "在线维护医保信息",
            symbol: "档",
            tone: "cyan",
            badge: "建设中",
            target: { type: "unavailable", message: "医保在线建档功能正在建设中" },
          },
        ],
      },
      {
        id: "during-visit",
        title: "诊中服务",
        actions: [
          {
            id: "payment",
            title: "在线缴费",
            description: "查看并支付待缴费用",
            symbol: "缴",
            tone: "blue",
            badge: "建设中",
            target: { type: "unavailable", message: "在线缴费功能正在建设中" },
          },
          {
            id: "insurance-code",
            title: "医保电子凭证",
            description: "出示医保电子凭证",
            symbol: "保",
            tone: "cyan",
            badge: "建设中",
            target: { type: "unavailable", message: "医保电子凭证功能正在建设中" },
          },
        ],
      },
      {
        id: "after-visit",
        title: "诊后服务",
        actions: [
          {
            id: "reports",
            title: "检验报告查询",
            description: "查询检验检查报告",
            symbol: "查",
            tone: "blue",
            badge: "建设中",
            target: { type: "unavailable", message: "报告查询功能正在建设中" },
          },
          {
            id: "invoice",
            title: "电子票据",
            description: "查看医疗电子票据",
            symbol: "票",
            tone: "cyan",
            badge: "建设中",
            target: { type: "unavailable", message: "电子票据功能正在建设中" },
          },
          {
            id: "inpatient-copy",
            title: "住院病案复印",
            description: "申请住院病案材料",
            symbol: "案",
            tone: "green",
            badge: "建设中",
            target: { type: "unavailable", message: "住院病案复印功能正在建设中" },
          },
          {
            id: "imaging",
            title: "影像查询",
            description: "查看医学影像资料",
            symbol: "影",
            tone: "blue",
            badge: "建设中",
            target: { type: "unavailable", message: "影像查询功能正在建设中" },
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
  _principal: HomeWorkbenchIdentity | null,
): HomeWorkbenchView {
  const view = { ...patientHome(), variant };
  if (variant === "staff") {
    view.primaryAction = {
      id: "manage-examination-items",
      title: "检查项目管理",
      description: "维护项目、房间与每周开放时间",
      symbol: "检",
      tone: "cyan",
      badge: "真实数据",
      target: {
        type: "navigate",
        url: "/pages/admin/appointment/index",
      },
    };
    view.notice = "检查项目、房间和每周配置已接入真实 Appointment 数据。";
    view.mock = false;
  }
  return view;
}

export const homeWorkbenchAdapter: HomeWorkbenchAdapter = {
  async load(variant, principal) {
    return Promise.resolve(buildHomeWorkbench(variant, principal));
  },
};
