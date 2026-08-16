import { request } from "@/api/client";
import type { CalculateWalkingRouteInput, WalkingRoute } from "@/types/guidance";

export const guidanceApi = {
  calculateWalkingRoute(input: CalculateWalkingRouteInput) {
    return request<WalkingRoute, CalculateWalkingRouteInput>({
      path: "/api/v1/guidance/routes/walking",
      method: "POST",
      data: input,
      authenticated: true,
    });
  },
};
