export interface DisplayProfile {
  avatarUrl: string;
  nickname: string;
}

export interface DisplayProfileInput {
  avatarUrl?: string;
  nickname?: string;
}

export interface SelfPatientPreferences {
  phoneMasked: string;
}

export interface SelfPatientPreferencesInput {
  phoneMasked?: string;
}
