import type { GuidanceLocationPoint } from "@/types/guidance";

export function sameLocation(left?: GuidanceLocationPoint, right?: GuidanceLocationPoint): boolean {
  if (!left || !right) return left === right;
  return left.latitude === right.latitude &&
    left.longitude === right.longitude &&
    left.provider_place_id === right.provider_place_id &&
    left.name === right.name;
}

export function routeEndpointsChanged(
  currentOrigin: GuidanceLocationPoint | undefined,
  currentDestination: GuidanceLocationPoint | undefined,
  nextOrigin: GuidanceLocationPoint | undefined,
  nextDestination: GuidanceLocationPoint | undefined,
): boolean {
  return !sameLocation(currentOrigin, nextOrigin) || !sameLocation(currentDestination, nextDestination);
}
