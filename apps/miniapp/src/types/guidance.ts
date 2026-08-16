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

export interface WalkingRouteStep {
  instruction: string;
  road_name: string;
  distance_meters: number;
  duration_seconds: number;
}

export interface WalkingRoute {
  origin: GuidanceLocationPoint;
  destination: GuidanceLocationPoint;
  distance_meters: number;
  duration_seconds: number;
  polyline: GuidanceRoutePoint[];
  steps: WalkingRouteStep[];
  provider: string;
}

export interface CalculateWalkingRouteInput {
  origin: GuidanceLocationPoint;
  destination: GuidanceLocationPoint;
}

export interface SearchGuidancePlacesInput {
  keyword: string;
  city?: string;
  limit?: number;
}

export interface SearchGuidancePlacesResult {
  places: GuidanceLocationPoint[];
}
