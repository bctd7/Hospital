import { appointmentMessageApi } from "@/api/appointment";
import { sessionState } from "@/stores/session";
import { hasRole } from "@/utils/appointmentManagement";
import { currentStaffDepartmentId } from "@/utils/staffDepartmentContext";

let refreshGeneration = 0;
const MESSAGE_TAB_INDEX = 2;
const TAB_BAR_ROUTES = new Set([
  "pages/home/index",
  "pages/registration/index",
  "pages/messages/index",
  "pages/profile/index",
]);

function currentPageIsTabBar(): boolean {
  const pages = getCurrentPages();
  const route = pages[pages.length - 1]?.route ?? "";
  return TAB_BAR_ROUTES.has(route);
}

/** 微信只允许在 TabBar 页面更新角标，详情页和登录页直接忽略。 */
export function updateMessageBadge(unreadCount: number) {
  if (!currentPageIsTabBar()) return;
  if (unreadCount <= 0) {
    clearMessageBadge();
    return;
  }
  uni.setTabBarBadge({
    index: MESSAGE_TAB_INDEX,
    text: unreadCount > 99 ? "99+" : String(unreadCount),
    fail: () => undefined,
  });
}

export async function refreshMessageBadge() {
  const generation = ++refreshGeneration;
  if (sessionState.status !== "authenticated") {
    clearMessageBadge();
    return;
  }
  try {
    let unreadCount = 0;
    if (sessionState.appVariant === "staff") {
      const departmentId = hasRole(sessionState.principal, "department_doctor")
        ? sessionState.principal?.department_id?.trim() ?? ""
        : currentStaffDepartmentId();
      if (!departmentId) {
        unreadCount = (await appointmentMessageApi.listDepartment("", 1, 1)).unreadCount;
      } else {
        unreadCount = (await appointmentMessageApi.listDepartment(departmentId, 1, 1)).unreadCount;
      }
    } else {
      unreadCount = (await appointmentMessageApi.listMine(1, 1)).unreadCount;
    }
    if (generation !== refreshGeneration) return;
    updateMessageBadge(unreadCount);
  } catch {
    // 消息角标失败不阻塞页面主体；进入消息页后会再次刷新。
  }
}

function clearMessageBadge() {
  uni.removeTabBarBadge({ index: MESSAGE_TAB_INDEX, fail: () => undefined });
}
