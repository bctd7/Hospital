import { beforeEach, describe, expect, it, vi } from "vitest";

const { principal } = vi.hoisted(() => ({
  principal: { account_id: "patient-1" },
}));

vi.mock("@/stores/session", () => ({
  sessionState: { principal },
}));

import {
  loadGuidanceRouteItinerary,
  saveGuidanceRouteItinerary,
  updateGuidanceRouteItinerary,
} from "@/utils/guidanceRouteImport";

const item = {
  item_id: "item-1",
  item_name: "腹部彩超",
  room_id: "room-1",
  room_display_name: "2号楼 · US301室",
  campus_id: "campus-1",
  building: "2号楼",
  floor_number: 3,
  room_number: "US301",
  service_date: "2099-08-19",
  session: "morning" as const,
  estimated_duration_minutes: 25,
  reason: "优先完成禁食项目",
  planned_start_time: "09:00",
  planned_end_time: "09:25",
  travel_minutes: 0,
  travel_time_estimated: false,
};

describe("guidance route itinerary snapshot", () => {
  let storage: Map<string, unknown>;

  beforeEach(() => {
    principal.account_id = "patient-1";
    storage = new Map();
    vi.stubGlobal("uni", {
      getStorageSync: vi.fn((key: string) => storage.get(key)),
      removeStorageSync: vi.fn((key: string) => storage.delete(key)),
      setStorageSync: vi.fn((key: string, value: unknown) => storage.set(key, value)),
    });
  });

  it("restores the imported itinerary and its selected transport mode", () => {
    saveGuidanceRouteItinerary("2099-08-19", [item]);
    updateGuidanceRouteItinerary({
      transportMode: "driving",
      initialOrigin: { name: "院外起点", address: "上海", latitude: 31.2, longitude: 121.4 },
    });

    expect(loadGuidanceRouteItinerary()).toMatchObject({
      accountId: "patient-1",
      serviceDate: "2099-08-19",
      transportMode: "driving",
      initialOrigin: { name: "院外起点" },
      stops: [{ itemId: "item-1", building: "2号楼" }],
    });
  });

  it("drops a snapshot that belongs to another account", () => {
    saveGuidanceRouteItinerary("2099-08-19", [item]);
    principal.account_id = "patient-2";
    expect(loadGuidanceRouteItinerary()).toBeUndefined();
    expect(uni.removeStorageSync).toHaveBeenCalledTimes(1);
    principal.account_id = "patient-1";
  });

  it("treats a storage read failure as no snapshot instead of blocking the page", () => {
    vi.mocked(uni.getStorageSync).mockImplementation(() => { throw new Error("storage unavailable"); });
    expect(loadGuidanceRouteItinerary()).toBeUndefined();
  });
});
