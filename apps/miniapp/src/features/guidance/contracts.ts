import type {
  GuidanceRoute,
  GuidanceRoutePoint,
  GuidanceRouteStep,
  GuidanceLocationPoint,
  GuidancePatientReminder,
  GuidancePreparationRule,
  PreparationRulePreview,
  SmartAppointmentPlan,
} from "@/types/guidance";

function arrayOf<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : [];
}

export function normalizePreparationPreview(value: Partial<PreparationRulePreview> | null | undefined): PreparationRulePreview {
  return {
    description: value?.description ?? "",
    preparation_rules: arrayOf<GuidancePreparationRule>(value?.preparation_rules),
    reminders: arrayOf<GuidancePatientReminder>(value?.reminders),
    unresolved_fragments: arrayOf<string>(value?.unresolved_fragments),
    parser_mode: value?.parser_mode ?? "deterministic",
    warning: value?.warning ?? "",
  };
}

export function visiblePlanKey(plan: SmartAppointmentPlan): string {
  return plan.items.map((item) => [
    item.item_id,
    item.room_id,
    item.service_date,
    item.session,
    item.planned_start_time,
    item.planned_end_time,
  ].join("@")).join("|");
}

export function distinctSmartAppointmentPlans(values: SmartAppointmentPlan[] | null | undefined): SmartAppointmentPlan[] {
  const result: SmartAppointmentPlan[] = [];
  const seen = new Set<string>();
  for (const value of arrayOf(values)) {
    const key = visiblePlanKey(value);
    if (!value?.items?.length || seen.has(key)) continue;
    seen.add(key);
    result.push(value);
  }
  return result;
}

export function normalizeGuidancePlaces(values: GuidanceLocationPoint[] | null | undefined): GuidanceLocationPoint[] {
  return arrayOf(values);
}

export function normalizeGuidanceRoute(value: GuidanceRoute): GuidanceRoute {
  return {
    ...value,
    polyline: arrayOf<GuidanceRoutePoint>(value.polyline),
    steps: arrayOf<GuidanceRouteStep>(value.steps),
  };
}
