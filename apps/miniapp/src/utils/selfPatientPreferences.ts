import type {
  SelfPatientPreferences,
  SelfPatientPreferencesInput,
} from "@/types/profile";

const SELF_PATIENT_PREFERENCES_STORAGE_KEY = "hospital:self-patient-preferences";

const DEFAULT_SELF_PATIENT_PREFERENCES: Readonly<SelfPatientPreferences> = {
  enabled: true,
  phoneMasked: "",
};

let runtimePreferences: SelfPatientPreferences | undefined;

function normalizePreferences(
  input?: SelfPatientPreferencesInput,
): SelfPatientPreferences {
  return {
    enabled: input?.enabled ?? DEFAULT_SELF_PATIENT_PREFERENCES.enabled,
    phoneMasked: input?.phoneMasked?.trim().slice(0, 32) ?? "",
  };
}

export function getSelfPatientPreferences(): SelfPatientPreferences {
  if (runtimePreferences) {
    return { ...runtimePreferences };
  }

  try {
    const storedPreferences = uni.getStorageSync(
      SELF_PATIENT_PREFERENCES_STORAGE_KEY,
    ) as SelfPatientPreferencesInput | undefined;
    runtimePreferences = normalizePreferences(storedPreferences);
  } catch {
    runtimePreferences = { ...DEFAULT_SELF_PATIENT_PREFERENCES };
  }

  return { ...runtimePreferences };
}

export function saveSelfPatientPreferences(
  input: SelfPatientPreferencesInput,
): SelfPatientPreferences {
  runtimePreferences = normalizePreferences({
    ...getSelfPatientPreferences(),
    ...input,
  });

  try {
    uni.setStorageSync(SELF_PATIENT_PREFERENCES_STORAGE_KEY, runtimePreferences);
  } catch {
    // 本地存储失败时保留当前运行时状态，不保存手机号明文。
  }

  return { ...runtimePreferences };
}
