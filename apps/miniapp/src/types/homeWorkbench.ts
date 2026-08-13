import type { AppVariant, CurrentIdentityResponse } from "@/types/auth";

export type HomeWorkbenchIdentity = Omit<
  CurrentIdentityResponse,
  "roles" | "permissions"
> & {
  readonly roles: readonly string[];
  readonly permissions: readonly string[];
};

export type HomeActionTarget =
  | { type: "navigate"; url: string }
  | { type: "tab"; url: string }
  | { type: "unavailable"; message: string };

export interface HomeAction {
  id: string;
  title: string;
  description: string;
  symbol: string;
  tone: "blue" | "cyan" | "violet" | "amber" | "green";
  badge?: string;
  target: HomeActionTarget;
}

export interface PatientServiceGroup {
  id: string;
  title: string;
  actions: HomeAction[];
}

export interface WorkbenchMetric {
  id: string;
  label: string;
  value: string;
  hint: string;
  tone: "blue" | "cyan" | "violet";
}

export interface HomeWorkbenchView {
  variant: AppVariant;
  eyebrow: string;
  title: string;
  description: string;
  primaryAction?: HomeAction;
  secondaryAction?: HomeAction;
  serviceGroups: PatientServiceGroup[];
  metrics: WorkbenchMetric[];
  managementActions: HomeAction[];
  notice: string;
  mock: boolean;
}

export interface HomeWorkbenchAdapter {
  load(
    variant: AppVariant,
    principal: HomeWorkbenchIdentity | null,
  ): Promise<HomeWorkbenchView>;
}
