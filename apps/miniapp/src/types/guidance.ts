export interface GuidanceLocationPoint {
  name: string;
  address: string;
  latitude: number;
  longitude: number;
  provider_place_id?: string;
}

export interface GuidanceRoutePoint {
  latitude: number;
  longitude: number;
}

export type GuidanceTravelMode = "walking" | "transit" | "driving";

export interface GuidanceRouteStep {
  instruction: string;
  road_name: string;
  distance_meters: number;
  duration_seconds: number;
}

export interface GuidanceRoute {
  origin: GuidanceLocationPoint;
  destination: GuidanceLocationPoint;
  distance_meters: number;
  duration_seconds: number;
  polyline: GuidanceRoutePoint[];
  steps: GuidanceRouteStep[];
  provider: string;
  mode: GuidanceTravelMode;
}

export interface CalculateGuidanceRouteInput {
  origin: GuidanceLocationPoint;
  destination: GuidanceLocationPoint;
  mode: GuidanceTravelMode;
}

export interface SearchGuidancePlacesInput {
  keyword: string;
  city?: string;
  limit?: number;
}

export interface SearchGuidancePlacesResult {
  places: GuidanceLocationPoint[];
}

export type PreparationRuleType = "fasting" | "no_water" | "drink_water";
export type PreparationStartMode = "advance_range" | "previous_day_time";

export interface GuidancePreparationRule {
  rule_type: PreparationRuleType;
  start_mode: PreparationStartMode;
  min_advance_minutes: number;
  recommended_advance_minutes: number;
  max_advance_minutes: number;
  previous_day_time: string;
  readiness_hint: string;
  source: "explicit" | "default" | "model" | "manual" | "";
}

export interface GuidancePatientReminder {
  text: string;
  advance_minutes: number;
}

export interface PreparationRulePreview {
  description: string;
  preparation_rules: GuidancePreparationRule[];
  reminders: GuidancePatientReminder[];
  unresolved_fragments: string[];
  parser_mode: "llm" | "deterministic" | string;
  warning: string;
}

export interface ConfiguredPrecedenceRule {
  predecessor_item_id: string;
  successor_item_id: string;
  staff_reason: string;
  patient_message: string;
}

export interface ExaminationItemConfiguration {
  item_id: string;
  owner_department_id: string;
  item_name: string;
  estimated_duration_minutes: number;
  status: string;
  item_version: number;
  description: string;
  precedence_rules: ConfiguredPrecedenceRule[];
  preparation_rules: GuidancePreparationRule[];
  reminders: GuidancePatientReminder[];
  configuration_version: number;
  updated_at: string;
}

export interface ConfigureExaminationItemInput {
  action: "create" | "update";
  item_id?: string;
  owner_department_id: string;
  item_name: string;
  estimated_duration_minutes: number;
  expected_item_version: number;
  description: string;
  precedence_rules: ConfiguredPrecedenceRule[];
  preparation_rules: GuidancePreparationRule[];
  reminders: GuidancePatientReminder[];
  expected_configuration_version: number;
  operation_id: string;
}

export interface SmartAppointmentPlanItem {
  item_id: string;
  item_name: string;
  room_id: string;
  room_display_name: string;
  campus_id: string;
  building: string;
  floor_number: number;
  room_number: string;
  service_date: string;
  session: "morning" | "afternoon";
  estimated_duration_minutes: number;
  reason: string;
  planned_start_time: string;
  planned_end_time: string;
  travel_minutes: number;
  travel_time_estimated: boolean;
}

export interface SmartAppointmentPlan {
  plan_id: string;
  title: string;
  summary: string;
  items: SmartAppointmentPlanItem[];
  expires_at: string;
}

export interface TodayRecommendationStage {
  stage_no: number;
  title: string;
  status: string;
  items: SmartAppointmentPlanItem[];
  focus: string;
}

export interface TodayExaminationRecommendation {
  service_date: string;
  stages: TodayRecommendationStage[];
  updated_at: string;
}
