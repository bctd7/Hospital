import { request } from "@/api/client";
import { operationId } from "@/api/management/httpShared";
import type {
  CalculateWalkingRouteInput,
  ConfigureExaminationItemInput,
  ExaminationItemConfiguration,
  PreparationRulePreview,
  SearchGuidancePlacesInput,
  SearchGuidancePlacesResult,
  WalkingRoute,
  SmartAppointmentPlan,
  TodayExaminationRecommendation,
} from "@/types/guidance";

export const guidanceApi = {
	previewPreparationRules(description: string) {
		return request<PreparationRulePreview, { description: string }>({
			path: "/api/v1/admin/guidance/preparation-rules/preview",
			method: "POST",
			data: { description },
			authenticated: true,
		});
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
  generateSmartAppointmentPlans(itemIds: string[], candidateDates: string[]) {
    return request<{ plans: SmartAppointmentPlan[] }, { item_ids: string[]; candidate_dates: string[] }>({
      path: "/api/v1/guidance/smart-appointment/plans",
      method: "POST",
      data: { item_ids: itemIds, candidate_dates: candidateDates },
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
  searchPlaces(input: SearchGuidancePlacesInput) {
    return request<SearchGuidancePlacesResult, SearchGuidancePlacesInput>({
      path: "/api/v1/guidance/places/search",
      method: "GET",
      data: input,
      authenticated: true,
    });
  },
  calculateWalkingRoute(input: CalculateWalkingRouteInput) {
    return request<WalkingRoute, CalculateWalkingRouteInput>({
      path: "/api/v1/guidance/routes/walking",
      method: "POST",
      data: input,
      authenticated: true,
    });
  },
};
