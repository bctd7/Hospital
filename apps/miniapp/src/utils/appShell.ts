import type {
  AppVariant,
  CurrentIdentityResponse,
} from "@/types/auth";

const DEPARTMENT_DOCTOR_ROLE = "department_doctor";
const SUPER_ADMIN_ROLE = "super_admin";

/**
 * 工作人员业务尚未实现，当前复用四个 Tab 页面并显示对应占位内容。
 * 后续继续按权限扩展工作人员视图，不拆分医生版和超管版。
 */
export const STAFF_APP_ENABLED = true;

const TAB_LABELS: Record<AppVariant, string[]> = {
  patient: ["首页", "挂号", "消息", "我的"],
  staff: ["工作台", "业务管理", "消息", "我的"],
};

let appliedVariant: AppVariant | null = null;
let applyingVariant: AppVariant | null = null;
let applyGeneration = 0;

/** 仅根据后端身份事实判断应使用哪一套应用，医生和超管共享 staff 版本。 */
export function resolveIdentityAppVariant(
  principal: CurrentIdentityResponse | null,
): AppVariant {
  const roles = principal?.roles ?? [];
  return roles.includes(DEPARTMENT_DOCTOR_ROLE) || roles.includes(SUPER_ADMIN_ROLE)
    ? "staff"
    : "patient";
}

/**
 * 界面转换的唯一入口。后台版本未开启时，即使身份属于工作人员也保持当前患者端。
 * 测试或未来启用时可以显式传入 true，不需要修改会话和页面判断。
 */
export function resolveAppVariant(
  principal: CurrentIdentityResponse | null,
  staffAppEnabled = STAFF_APP_ENABLED,
): AppVariant {
  const identityVariant = resolveIdentityAppVariant(principal);
  return identityVariant === "staff" && staffAppEnabled ? "staff" : "patient";
}

/** 保留用户主动选择的患者端；只有真实工作人员身份才能选择工作人员端。 */
export function normalizeAppVariant(
  selected: unknown,
  principal: CurrentIdentityResponse | null,
): AppVariant {
  return selected === "staff" && resolveAppVariant(principal) === "staff"
    ? "staff"
    : "patient";
}

/** 前端只控制可见性；最终权限必须由后端按 Token 中的 permission 校验。 */
export function hasIdentityPermission(
  principal: CurrentIdentityResponse | null,
  permission: string,
): boolean {
  return principal?.permissions.includes(permission) ?? false;
}

export function applyAppVariantNavigation(variant: AppVariant) {
  if (appliedVariant === variant || applyingVariant === variant) {
    return;
  }

  applyingVariant = variant;
  const generation = ++applyGeneration;
  let pending = TAB_LABELS[variant].length;
  let failed = false;

  TAB_LABELS[variant].forEach((text, index) => {
    uni.setTabBarItem({
      index,
      text,
      fail: () => {
        failed = true;
      },
      complete: () => {
        pending -= 1;
        if (pending > 0 || generation !== applyGeneration) {
          return;
        }

        applyingVariant = null;
        if (!failed) {
          appliedVariant = variant;
        }
      },
    });
  });
}

/** 根据已经解析出的版本进入对应页面树，是登录后界面转换的统一入口。 */
export function openAppVariant(
  variant: AppVariant,
  onFailure?: () => void,
) {
  applyAppVariantNavigation(variant);
  uni.switchTab({
    url: "/pages/home/index",
    fail: onFailure,
  });
}
