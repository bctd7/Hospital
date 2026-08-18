import type { SmartAppointmentPlanItem } from "@/types/guidance";
import type { GuidanceLocationPoint, GuidanceTravelMode } from "@/types/guidance";
import { sessionState } from "@/stores/session";

const STORAGE_KEY = "hospital:guidance:route-itinerary:v2";

export interface GuidanceRouteStop {
  itemId: string;
  itemName: string;
  building: string;
  roomDisplayName: string;
  roomNumber: string;
  searchKeyword: string;
}

export interface GuidanceRouteItinerary {
  accountId: string;
  serviceDate: string;
  importedAt: string;
  expiresAt: string;
  transportMode: GuidanceTravelMode;
  initialOrigin?: GuidanceLocationPoint;
  stops: GuidanceRouteStop[];
}

export function saveGuidanceRouteItinerary(serviceDate: string, items: SmartAppointmentPlanItem[]) {
  const stops = items.map((item) => ({
    itemId: item.item_id,
    itemName: item.item_name,
    building: item.building,
    roomDisplayName: item.room_display_name,
    roomNumber: item.room_number,
    searchKeyword: item.building.includes("上海市第二人民医院")
      ? item.building
      : `上海市第二人民医院${item.building}`,
  }));
  const itinerary: GuidanceRouteItinerary = {
    accountId: sessionState.principal?.account_id ?? "",
    serviceDate,
    importedAt: new Date().toISOString(),
    expiresAt: `${serviceDate}T23:59:59+08:00`,
    transportMode: "walking",
    stops,
  };
  uni.setStorageSync(STORAGE_KEY, itinerary);
}

export function loadGuidanceRouteItinerary(): GuidanceRouteItinerary | undefined {
  let value: GuidanceRouteItinerary | undefined;
  try {
    value = uni.getStorageSync(STORAGE_KEY) as GuidanceRouteItinerary | undefined;
  } catch {
    return undefined;
  }
  if (!value || !Array.isArray(value.stops) || !value.stops.length) return undefined;
  if (value.accountId !== (sessionState.principal?.account_id ?? "") || Date.parse(value.expiresAt) < Date.now()) {
    try { uni.removeStorageSync(STORAGE_KEY); } catch { /* 下一次读取时再次校验。 */ }
    return undefined;
  }
  return value;
}

export function updateGuidanceRouteItinerary(patch: Partial<Pick<GuidanceRouteItinerary, "transportMode" | "initialOrigin">>) {
  const current = loadGuidanceRouteItinerary();
  if (!current) return;
  uni.setStorageSync(STORAGE_KEY, { ...current, ...patch });
}
