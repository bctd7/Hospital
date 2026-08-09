export interface DisplayProfile {
  avatarUrl: string;
  nickname: string;
}

export interface DisplayProfileInput {
  avatarUrl?: string;
  nickname?: string;
}

export interface SelfPatientPreferences {
  enabled: boolean;
  phoneMasked: string;
}

export interface SelfPatientPreferencesInput {
  enabled?: boolean;
  phoneMasked?: string;
}
