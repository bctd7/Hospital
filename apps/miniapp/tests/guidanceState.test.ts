import { describe, expect, it } from "vitest";

import { distinctSmartAppointmentPlans, normalizeGuidanceRoute, normalizePreparationPreview } from "@/features/guidance/contracts";
import { routeEndpointsChanged } from "@/features/guidance/routeState";
import { createLatestRequestGuard } from "@/features/shared/latestRequest";
import type { SmartAppointmentPlan } from "@/types/guidance";

function plan(planId: string, roomId = "room-1"): SmartAppointmentPlan {
  return {
    plan_id: planId,
    title: planId,
    summary: "summary",
    expires_at: "2026-08-18T12:00:00Z",
    items: [{
      item_id: "item-1", item_name: "腹部彩超", room_id: roomId, room_display_name: "US301室",
      campus_id: "campus-1", building: "2号楼", floor_number: 3, room_number: "US301",
      service_date: "2026-08-19", session: "morning", estimated_duration_minutes: 25,
      reason: "reason", planned_start_time: "09:00", planned_end_time: "09:25",
      travel_minutes: 0, travel_time_estimated: false,
    }],
  };
}

describe("guidance stable page state", () => {
  it("normalizes model null collections before a page reads length", () => {
    const result = normalizePreparationPreview({
      preparation_rules: null,
      reminders: null,
      unresolved_fragments: null,
    } as never);
    expect(result.preparation_rules).toEqual([]);
    expect(result.reminders).toEqual([]);
    expect(result.unresolved_fragments).toEqual([]);
  });

  it("normalizes optional route collections before mounting the native map", () => {
    const point = { name: "A", address: "", latitude: 31.2, longitude: 121.4 };
    const result = normalizeGuidanceRoute({
      origin: point, destination: point, distance_meters: 0, duration_seconds: 0,
      polyline: null, steps: null, provider: "local", mode: "walking",
    } as never);
    expect(result.polyline).toEqual([]);
    expect(result.steps).toEqual([]);
  });

  it("does not show a duplicate alternative plan", () => {
    expect(distinctSmartAppointmentPlans([plan("primary"), plan("duplicate"), plan("different", "room-2")]))
      .toHaveLength(2);
  });

  it("rejects stale async results", () => {
    const guard = createLatestRequestGuard();
    const first = guard.begin();
    const second = guard.begin();
    expect(guard.isCurrent(first)).toBe(false);
    expect(guard.isCurrent(second)).toBe(true);
  });

  it("keeps an existing map mounted when resolved endpoints did not change", () => {
    const point = { name: "1号楼", address: "上海", latitude: 31.2, longitude: 121.4 };
    expect(routeEndpointsChanged(point, point, { ...point }, { ...point })).toBe(false);
  });
});
