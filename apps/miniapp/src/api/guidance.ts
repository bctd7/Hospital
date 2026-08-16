import { request } from "@/api/client";
import type {
  CalculateWalkingRouteInput,
  SearchGuidancePlacesInput,
  SearchGuidancePlacesResult,
  WalkingRoute,
} from "@/types/guidance";

export const guidanceApi = {
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
