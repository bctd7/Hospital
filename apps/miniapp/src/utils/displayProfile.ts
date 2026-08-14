import type { DisplayProfile, DisplayProfileInput } from "@/types/profile";
import { request } from "@/api/client";

const DISPLAY_PROFILE_STORAGE_KEY = "hospital:display-profile";

export const DEFAULT_DISPLAY_PROFILE: Readonly<DisplayProfile> = {
  avatarUrl: "",
  nickname: "微信用户",
};

let runtimeProfile: DisplayProfile | undefined;

const LOCAL_AVATAR_HOST_PATTERN =
  /^https?:\/\/(?:localhost|127(?:\.\d{1,3}){3}|\[::1\]|tmp)(?::\d+)?(?:[/?#]|$)/i;

function normalizeProfile(input?: DisplayProfileInput): DisplayProfile {
  const nickname = input?.nickname?.trim().slice(0, 32);

  return {
    avatarUrl: input?.avatarUrl?.trim() ?? "",
    nickname: nickname || DEFAULT_DISPLAY_PROFILE.nickname,
  };
}

function persistentProfile(profile: DisplayProfile): DisplayProfile {
  const avatarUrl = profile.avatarUrl;
  const isRemoteAvatar = /^https?:\/\//i.test(avatarUrl);

  return {
    ...profile,
    // 本地临时文件地址只在当前运行期间有效，不能跨会话缓存。
    avatarUrl: isRemoteAvatar && !LOCAL_AVATAR_HOST_PATTERN.test(avatarUrl) ? avatarUrl : "",
  };
}

function persistDisplayProfile(profile: DisplayProfile) {
  uni.setStorageSync(DISPLAY_PROFILE_STORAGE_KEY, persistentProfile(profile));
}

export function getDisplayProfile(): DisplayProfile {
  if (runtimeProfile) {
    return { ...runtimeProfile };
  }

  try {
    const storedProfile = uni.getStorageSync(DISPLAY_PROFILE_STORAGE_KEY) as DisplayProfileInput | undefined;
    const normalizedProfile = normalizeProfile(storedProfile);
    runtimeProfile = persistentProfile(normalizedProfile);
    if (runtimeProfile.avatarUrl !== normalizedProfile.avatarUrl) {
      persistDisplayProfile(runtimeProfile);
    }
  } catch {
    runtimeProfile = { ...DEFAULT_DISPLAY_PROFILE };
  }

  return { ...runtimeProfile };
}

export function saveDisplayProfile(input: DisplayProfileInput): DisplayProfile {
  runtimeProfile = normalizeProfile(input);

  try {
    persistDisplayProfile(runtimeProfile);
  } catch {
    // 当前运行时仍保留展示资料；本地存储失败不阻塞用户进入主应用。
  }

  return { ...runtimeProfile };
}

interface DisplayProfileResponse {
  nickname?: string;
  management_version: number;
}

export async function loadDisplayProfileFromServer(): Promise<DisplayProfile> {
  const response = await request<DisplayProfileResponse>({
    path: "/api/v1/auth/me/display-profile",
    authenticated: true,
  });
  const local = getDisplayProfile();
  return saveDisplayProfile({
    avatarUrl: local.avatarUrl,
    nickname: response.nickname || local.nickname,
  });
}

export async function saveDisplayProfileToServer(input: DisplayProfileInput): Promise<DisplayProfile> {
  const local = saveDisplayProfile(input);
  const response = await request<DisplayProfileResponse>({
    path: "/api/v1/auth/me/display-profile",
    method: "PUT",
    authenticated: true,
    data: { nickname: local.nickname === DEFAULT_DISPLAY_PROFILE.nickname ? "" : local.nickname },
  });
  return saveDisplayProfile({
    avatarUrl: local.avatarUrl,
    nickname: response.nickname || DEFAULT_DISPLAY_PROFILE.nickname,
  });
}
