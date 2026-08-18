import { request } from "@/api/client";
import { operationId } from "@/api/management/httpShared";
import { normalizeGuidancePlaces, normalizeGuidanceRoute, normalizePreparationPreview } from "@/features/guidance/contracts";
import type {
  CalculateGuidanceRouteInput,
  ConfigureExaminationItemInput,
  ExaminationItemConfiguration,
  PreparationRulePreview,
  SearchGuidancePlacesInput,
  SearchGuidancePlacesResult,
  GuidanceRoute,
	GuidancePatientReminder,
  SmartAppointmentPlan,
  TodayExaminationRecommendation,
} from "@/types/guidance";

export const guidanceApi = {
	getExaminationItemPatientReminders(itemId: string) {
		return request<{ item_id: string; reminders: GuidancePatientReminder[] }>({
			path: `/api/v1/guidance/examination-items/${encodeURIComponent(itemId)}/reminders`,
			method: "GET",
			authenticated: true,
		});
	},
	async previewPreparationRules(description: string) {
		const value = await request<PreparationRulePreview, { description: string }>({
			path: "/api/v1/admin/guidance/preparation-rules/preview",
			method: "POST",
			data: { description },
			authenticated: true,
			timeoutMs: 32000,
		});
		return normalizePreparationPreview(value);
	},
	getExaminationItemConfiguration(itemId: string) {
		return request<ExaminationItemConfiguration>({
			path: `/api/v1/admin/guidance/examination-items/${encodeURIComponent(itemId)}/configuration`,
			method: "GET",
			authenticated: true,
		});
	},
	configureExaminationItem(input: ConfigureExaminationItemInput) {
		return request<ExaminationItemConfiguration, ConfigureExaminationItemInput>({
			path: "/api/v1/admin/guidance/examination-items/configuration",
			method: "POST",
			data: input,
			authenticated: true,
		});
	},
  generateSmartAppointmentPlans(itemIds: string[], candidateAvailability: Array<{ service_date: string; sessions: Array<"morning" | "afternoon"> }>) {
    return request<{ plans: SmartAppointmentPlan[] }, { item_ids: string[]; candidate_availability: Array<{ service_date: string; sessions: Array<"morning" | "afternoon"> }> }>({
      path: "/api/v1/guidance/smart-appointment/plans",
      method: "POST",
      data: { item_ids: itemIds, candidate_availability: candidateAvailability },
      authenticated: true,
    });
  },
  confirmSmartAppointmentPlan(planId: string) {
    return request<{ plan_id: string; booking_ids: string[] }, { operation_id: string }>({
      path: `/api/v1/guidance/smart-appointment/plans/${encodeURIComponent(planId)}/confirm`,
      method: "POST",
      data: { operation_id: operationId() },
      authenticated: true,
    });
  },
  getTodayExaminationRecommendation() {
    return request<TodayExaminationRecommendation>({
      path: "/api/v1/guidance/today/recommendation",
      method: "GET",
      authenticated: true,
    });
  },
  async searchPlaces(input: SearchGuidancePlacesInput) {
		const value = await request<SearchGuidancePlacesResult, SearchGuidancePlacesInput>({
      path: "/api/v1/guidance/places/search",
      method: "GET",
      data: input,
			authenticated: true,
		});
		return { places: normalizeGuidancePlaces(value.places) };
	},
  async calculateRoute(input: CalculateGuidanceRouteInput) {
		const value = await request<GuidanceRoute, CalculateGuidanceRouteInput>({
      path: "/api/v1/guidance/routes",
      method: "POST",
      data: input,
			authenticated: true,
		});
		return normalizeGuidanceRoute(value);
	},
};
