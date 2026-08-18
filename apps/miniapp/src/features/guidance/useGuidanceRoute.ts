import { onLoad, onShow } from "@dcloudio/uni-app";
import { computed, nextTick, ref } from "vue";

import { ApiError } from "@/api/client";
import { guidanceApi } from "@/api/guidance";
import { routeEndpointsChanged } from "@/features/guidance/routeState";
import { createLatestRequestGuard } from "@/features/shared/latestRequest";
import type { GuidanceLocationPoint, GuidanceRoute, GuidanceTravelMode, TodayExaminationRecommendation } from "@/types/guidance";
import { loadGuidanceRouteItinerary, updateGuidanceRouteItinerary, type GuidanceRouteItinerary } from "@/utils/guidanceRouteImport";

interface MapMarker {
  id: number;
  latitude: number;
  longitude: number;
  title: string;
  width: number;
  height: number;
  label: { content: string; color: string; fontSize: number; borderRadius: number; bgColor: string; padding: number };
}

interface MapPolyline {
  points: Array<{ latitude: number; longitude: number }>;
  color: string;
  width: number;
  arrowLine: boolean;
  borderColor: string;
  borderWidth: number;
}

export function useGuidanceRoute() {
  const origin = ref<GuidanceLocationPoint>();
  const destination = ref<GuidanceLocationPoint>();
  const route = ref<GuidanceRoute>();
  const loading = ref(false);
  const error = ref("");
  const searchTarget = ref<"origin" | "destination">();
  const importedItinerary = ref<GuidanceRouteItinerary>();
  const recommendation = ref<TodayExaminationRecommendation>();
  const selectedStopIndex = ref(-1);
  const resolvedStops = ref<Record<number, GuidanceLocationPoint>>({});
  const refreshingProgress = ref(false);
  const travelMode = ref<GuidanceTravelMode>("walking");
  const outsideRouteLeg = ref(true);
  const pageReady = ref(false);
  const stageRequest = createLatestRequestGuard();
  const progressRequest = createLatestRequestGuard();
  const routeRequest = createLatestRequestGuard();
  const stopRequests = new Map<number, Promise<GuidanceLocationPoint | undefined>>();
  const travelModes: Array<{ value: GuidanceTravelMode; label: string }> = [
    { value: "walking", label: "步行" }, { value: "transit", label: "公交" }, { value: "driving", label: "驾车" },
  ];

  onLoad((query) => { void initializeRoutePage(query?.source === "today"); });
  onShow(() => {
    if (pageReady.value && importedItinerary.value) void refreshProgress(false);
  });

  const canCalculate = computed(() => Boolean(origin.value && destination.value && !loading.value));
  const activeItemIdsInRecommendedOrder = computed(() => recommendation.value?.stages
    .filter((stage) => stage.status !== "completed")
    .flatMap((stage) => stage.items.map((item) => item.item_id)) ?? []);
  const activeItemIds = computed(() => new Set(activeItemIdsInRecommendedOrder.value));
  const completedItemIdsInRecommendedOrder = computed(() => recommendation.value?.stages
    .filter((stage) => stage.status === "completed")
    .flatMap((stage) => stage.items.map((item) => item.item_id)) ?? []);
  const currentStopIndex = computed(() => {
    const itinerary = importedItinerary.value;
    if (!itinerary) return -1;
    if (!recommendation.value) return 0;
    const nextItemId = activeItemIdsInRecommendedOrder.value[0];
    return nextItemId ? itinerary.stops.findIndex((stop) => stop.itemId === nextItemId) : -1;
  });
  const completedStopCount = computed(() => importedItinerary.value?.stops.filter((stop) => recommendation.value && !activeItemIds.value.has(stop.itemId)).length ?? 0);
  const isOutsideStage = computed(() => !importedItinerary.value || outsideRouteLeg.value);
  const effectiveMode = computed<GuidanceTravelMode>(() => isOutsideStage.value ? travelMode.value : "walking");
  const mapCenter = computed(() => {
    const points = route.value?.polyline ?? [];
    if (points.length > 0) return points[Math.floor(points.length / 2)]!;
    return origin.value ?? destination.value ?? { latitude: 31.2304, longitude: 121.4737 };
  });
  const markers = computed<MapMarker[]>(() => route.value ? [marker(1, route.value.origin, "起"), marker(2, route.value.destination, "终")] : []);
  const polylines = computed<MapPolyline[]>(() => route.value?.polyline.length ? [{
    points: route.value.polyline, color: "#168fe4", width: 6, arrowLine: true, borderColor: "#ffffff", borderWidth: 2,
  }] : []);
  const mapPoints = computed(() => route.value?.polyline ?? []);
  const distanceText = computed(() => {
    const meters = route.value?.distance_meters ?? 0;
    return meters >= 1000 ? `${(meters / 1000).toFixed(1)} 公里` : `${meters} 米`;
  });
  const durationText = computed(() => {
    const seconds = route.value?.duration_seconds ?? 0;
    if (seconds <= 0) return "少于 1 分钟";
    const minutes = Math.max(1, Math.ceil(seconds / 60));
    return minutes >= 60 ? `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分钟` : `${minutes} 分钟`;
  });
  const providerText = computed(() => route.value?.provider === "local" ? "起点和终点位于同一位置" : `高德地图${modeLabel(route.value?.mode ?? effectiveMode.value)}路线`);
  const actionLabel = computed(() => `计算${modeLabel(effectiveMode.value)}路线`);
  const durationLabel = computed(() => effectiveMode.value === "walking" ? "预计步行" : effectiveMode.value === "transit" ? "预计公交" : "预计驾车");

  function choosePoint(target: "origin" | "destination") { searchTarget.value = target; }

  async function initializeRoutePage(importToday: boolean) {
    if (importToday) await loadImportedRoute();
    pageReady.value = true;
  }

  async function loadImportedRoute() {
    const itinerary = loadGuidanceRouteItinerary();
    if (!itinerary?.stops.length) return;
    importedItinerary.value = itinerary;
    travelMode.value = itinerary.transportMode ?? "walking";
    origin.value = itinerary.initialOrigin;
    await refreshProgress(false);
  }

  async function resolveStop(index: number) {
    const cached = resolvedStops.value[index];
    if (cached) return cached;
    const pending = stopRequests.get(index);
    if (pending) return pending;
    const stop = importedItinerary.value?.stops[index];
    if (!stop) return undefined;
    const request = guidanceApi.searchPlaces({ keyword: stop.searchKeyword, city: "上海市", limit: 8 })
      .then((result) => {
        const exact = result.places.find((place) => place.name.includes(stop.building)) ?? result.places[0];
        if (exact) resolvedStops.value = { ...resolvedStops.value, [index]: exact };
        return exact;
      })
      .finally(() => stopRequests.delete(index));
    stopRequests.set(index, request);
    return request;
  }

  async function selectStage(index: number, followCurrentProgress = false) {
    const itinerary = importedItinerary.value;
    if (!itinerary || index < 0 || index >= itinerary.stops.length) return;
    const token = stageRequest.begin();
    try {
      const nextDestination = await resolveStop(index);
      let nextOrigin: GuidanceLocationPoint | undefined;
      let nextOutsideRouteLeg = false;
      if (followCurrentProgress) {
        const completedIds = completedItemIdsInRecommendedOrder.value;
        const lastCompletedItemId = completedIds[completedIds.length - 1];
        const completedIndex = lastCompletedItemId ? itinerary.stops.findIndex((stop) => stop.itemId === lastCompletedItemId) : -1;
        if (completedIndex >= 0) nextOrigin = await resolveStop(completedIndex);
        else {
          nextOrigin = itinerary.initialOrigin;
          nextOutsideRouteLeg = true;
        }
      } else if (index === 0) {
        nextOrigin = itinerary.initialOrigin;
        nextOutsideRouteLeg = true;
      } else nextOrigin = await resolveStop(index - 1);
      if (!stageRequest.isCurrent(token)) return;
      const endpointsChanged = routeEndpointsChanged(origin.value, destination.value, nextOrigin, nextDestination);
      selectedStopIndex.value = index;
      origin.value = nextOrigin;
      destination.value = nextDestination;
      outsideRouteLeg.value = nextOutsideRouteLeg;
      error.value = "";
      if (endpointsChanged) {
        routeRequest.cancel();
        loading.value = false;
        route.value = undefined;
      }
      if (!nextOrigin) searchTarget.value = "origin";
    } catch {
      if (stageRequest.isCurrent(token)) error.value = "当前检查楼栋暂时无法定位，请手动选择地点";
    }
  }

  async function refreshProgress(showToast = true) {
    if (!importedItinerary.value || refreshingProgress.value) return;
    const token = progressRequest.begin();
    refreshingProgress.value = true;
    try {
      const value = await guidanceApi.getTodayExaminationRecommendation();
      if (!progressRequest.isCurrent(token)) return;
      recommendation.value = value;
      const nextIndex = currentStopIndex.value;
      if (nextIndex >= 0) await selectStage(nextIndex, true);
      else {
        selectedStopIndex.value = -1;
        route.value = undefined;
        if (showToast) uni.showToast({ title: "今日检查已全部完成", icon: "success" });
      }
    } catch (cause) {
      if (progressRequest.isCurrent(token) && showToast) uni.showToast({ title: cause instanceof Error ? cause.message : "检查进度刷新失败", icon: "none" });
    } finally {
      if (progressRequest.isCurrent(token)) refreshingProgress.value = false;
    }
  }

  function cancelCurrentRoute() {
    routeRequest.cancel();
    loading.value = false;
    route.value = undefined;
    error.value = "";
  }

  function selectPlace(place: GuidanceLocationPoint) {
    stageRequest.cancel();
    cancelCurrentRoute();
    if (searchTarget.value === "origin") {
      origin.value = place;
      if (importedItinerary.value && selectedStopIndex.value === 0) {
        importedItinerary.value = { ...importedItinerary.value, initialOrigin: place };
        updateGuidanceRouteItinerary({ initialOrigin: place });
      }
    } else if (searchTarget.value === "destination") destination.value = place;
    searchTarget.value = undefined;
  }

  function chooseMode(mode: GuidanceTravelMode) {
    cancelCurrentRoute();
    travelMode.value = mode;
    if (importedItinerary.value) {
      importedItinerary.value = { ...importedItinerary.value, transportMode: mode };
      updateGuidanceRouteItinerary({ transportMode: mode });
    }
  }

  function swapPoints() {
    stageRequest.cancel();
    cancelCurrentRoute();
    const previous = origin.value;
    origin.value = destination.value;
    destination.value = previous;
  }

  async function calculateRoute() {
    if (!origin.value || !destination.value || loading.value) return;
    const token = routeRequest.begin();
    const requestedOrigin = origin.value;
    const requestedDestination = destination.value;
    const requestedMode = effectiveMode.value;
    loading.value = true;
    error.value = "";
    try {
      const result = await guidanceApi.calculateRoute({ origin: requestedOrigin, destination: requestedDestination, mode: requestedMode });
      if (!routeRequest.isCurrent(token)) return;
      route.value = result;
      await nextTick();
      uni.createMapContext("guidance-route-map").includePoints({ points: result.polyline, padding: [56, 40, 56, 40] });
    } catch (cause) {
      if (!routeRequest.isCurrent(token)) return;
      route.value = undefined;
      if (cause instanceof ApiError && cause.code === "SERVICE_UNAVAILABLE") error.value = "路线服务暂不可用，请稍后重试";
      else if (cause instanceof ApiError && cause.code === "NOT_FOUND") error.value = "没有找到可用路线，请重新选择地点或出行方式";
      else error.value = cause instanceof Error ? cause.message : "路线计算失败，请稍后重试";
      uni.showToast({ title: error.value, icon: "none" });
    } finally {
      if (routeRequest.isCurrent(token)) loading.value = false;
    }
  }

  return {
    origin, destination, route, loading, error, searchTarget, importedItinerary, recommendation,
    selectedStopIndex, refreshingProgress, travelMode, travelModes, pageReady,
    canCalculate, activeItemIds, currentStopIndex, completedStopCount, isOutsideStage,
    mapCenter, markers, polylines, mapPoints, distanceText, durationText, providerText, actionLabel, durationLabel,
    choosePoint, selectStage, refreshProgress, selectPlace, chooseMode, swapPoints, calculateRoute, modeLabel,
  };
}

function marker(id: number, point: GuidanceLocationPoint, content: string): MapMarker {
  return {
    id, latitude: point.latitude, longitude: point.longitude, title: point.name, width: 28, height: 28,
    label: { content, color: "#ffffff", fontSize: 12, borderRadius: 14, bgColor: id === 1 ? "#168fe4" : "#16a085", padding: 6 },
  };
}

function modeLabel(mode: GuidanceTravelMode) {
  return ({ walking: "步行", transit: "公交", driving: "驾车" } as Record<GuidanceTravelMode, string>)[mode];
}
