import { appointmentMessageApi } from "@/api/appointment";
import { sessionState } from "@/stores/session";
import { hasRole } from "@/utils/appointmentManagement";
import { currentStaffDepartmentId } from "@/utils/staffDepartmentContext";

let refreshGeneration = 0;

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
    if (unreadCount > 0) {
      uni.setTabBarBadge({ index: 2, text: unreadCount > 99 ? "99+" : String(unreadCount) });
    } else {
      clearMessageBadge();
    }
  } catch {
    // 消息角标失败不阻塞页面主体；进入消息页后会再次刷新。
  }
}

function clearMessageBadge() {
  uni.removeTabBarBadge({ index: 2, fail: () => undefined });
}
