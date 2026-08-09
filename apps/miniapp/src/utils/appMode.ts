import type { AppMode } from "@/types/auth";

const TAB_LABELS: Record<AppMode, string[]> = {
  patient: ["首页", "挂号", "消息", "我的"],
  doctor: ["工作台", "预约管理", "消息", "我的"],
};

let appliedMode: AppMode | null = null;
let applyingMode: AppMode | null = null;
let applyGeneration = 0;

/**
 * 标准 TabBar 的页面路径保持不变，仅根据当前使用模式调整用户可见文案。
 * 模式不是授权依据；医生接口仍由后端根据 Token 角色校验。
 */
export function applyAppModeNavigation(mode: AppMode) {
  if (appliedMode === mode || applyingMode === mode) {
    return;
  }

  applyingMode = mode;
  const generation = ++applyGeneration;
  let pending = TAB_LABELS[mode].length;
  let failed = false;

  TAB_LABELS[mode].forEach((text, index) => {
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

        applyingMode = null;
        if (!failed) {
          appliedMode = mode;
        }
      },
    });
  });
}
