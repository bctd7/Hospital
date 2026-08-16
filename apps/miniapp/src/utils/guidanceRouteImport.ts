import type { SmartAppointmentPlanItem } from "@/types/guidance";

const STORAGE_KEY = "hospital:guidance:route-itinerary:v1";

export interface GuidanceRouteStop {
  itemId: string;
  itemName: string;
  building: string;
  roomDisplayName: string;
  roomNumber: string;
  searchKeyword: string;
}

export interface GuidanceRouteItinerary {
  serviceDate: string;
  importedAt: string;
  stops: GuidanceRouteStop[];
}

export function saveGuidanceRouteItinerary(serviceDate: string, items: SmartAppointmentPlanItem[]) {
  const stops = items.map((item) => ({
    itemId: item.item_id,
    itemName: item.item_name,
    building: item.building,
    roomDisplayName: item.room_display_name,
    roomNumber: item.room_number,
    searchKeyword: `${item.building} ${item.room_display_name}`.trim(),
  }));
  const itinerary: GuidanceRouteItinerary = {
    serviceDate,
    importedAt: new Date().toISOString(),
    stops,
  };
  uni.setStorageSync(STORAGE_KEY, itinerary);
}

export function loadGuidanceRouteItinerary(): GuidanceRouteItinerary | undefined {
  const value = uni.getStorageSync(STORAGE_KEY) as GuidanceRouteItinerary | undefined;
  return value && Array.isArray(value.stops) && value.stops.length ? value : undefined;
}
