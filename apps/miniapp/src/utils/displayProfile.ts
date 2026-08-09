import type { DisplayProfile, DisplayProfileInput } from "@/types/profile";

const DISPLAY_PROFILE_STORAGE_KEY = "hospital:display-profile";

export const DEFAULT_DISPLAY_PROFILE: Readonly<DisplayProfile> = {
  avatarUrl: "",
  nickname: "微信用户",
};

let runtimeProfile: DisplayProfile | undefined;

function normalizeProfile(input?: DisplayProfileInput): DisplayProfile {
  const nickname = input?.nickname?.trim().slice(0, 32);

  return {
    avatarUrl: input?.avatarUrl?.trim() ?? "",
    nickname: nickname || DEFAULT_DISPLAY_PROFILE.nickname,
  };
}

export function getDisplayProfile(): DisplayProfile {
  if (runtimeProfile) {
    return { ...runtimeProfile };
  }

  try {
    const storedProfile = uni.getStorageSync(DISPLAY_PROFILE_STORAGE_KEY) as DisplayProfileInput | undefined;
    runtimeProfile = normalizeProfile(storedProfile);
  } catch {
    runtimeProfile = { ...DEFAULT_DISPLAY_PROFILE };
  }

  return { ...runtimeProfile };
}

export function saveDisplayProfile(input: DisplayProfileInput): DisplayProfile {
  runtimeProfile = normalizeProfile(input);

  try {
    uni.setStorageSync(DISPLAY_PROFILE_STORAGE_KEY, runtimeProfile);
  } catch {
    // 当前运行时仍保留展示资料；本地存储失败不阻塞用户进入主应用。
  }

  return { ...runtimeProfile };
}
