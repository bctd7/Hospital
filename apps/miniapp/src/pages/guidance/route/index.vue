<script setup lang="ts">
import { onLoad, onShow } from "@dcloudio/uni-app";
import { computed, nextTick, ref } from "vue";

import { ApiError } from "@/api/client";
import { guidanceApi } from "@/api/guidance";
import PlaceSearchSheet from "@/components/guidance/PlaceSearchSheet.vue";
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
const travelModes: Array<{ value: GuidanceTravelMode; label: string }> = [
  { value: "walking", label: "步行" },
  { value: "transit", label: "公交" },
  { value: "driving", label: "驾车" },
];

onLoad((query) => {
  if (query?.source === "today") void loadImportedRoute();
});
onShow(() => {
  if (importedItinerary.value) void refreshProgress(false);
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
const markers = computed<MapMarker[]>(() => {
  if (!route.value) return [];
  return [
    marker(1, route.value.origin, "起"),
    marker(2, route.value.destination, "终"),
  ];
});
const polylines = computed<MapPolyline[]>(() => route.value?.polyline.length
  ? [{
      points: route.value.polyline,
      color: "#168fe4",
      width: 6,
      arrowLine: true,
      borderColor: "#ffffff",
      borderWidth: 2,
    }]
  : []);
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

function marker(id: number, point: GuidanceLocationPoint, content: string): MapMarker {
  return {
    id,
    latitude: point.latitude,
    longitude: point.longitude,
    title: point.name,
    width: 28,
    height: 28,
    label: {
      content,
      color: "#ffffff",
      fontSize: 12,
      borderRadius: 14,
      bgColor: id === 1 ? "#168fe4" : "#16a085",
      padding: 6,
    },
  };
}

function choosePoint(target: "origin" | "destination") {
  searchTarget.value = target;
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
  const stop = importedItinerary.value?.stops[index];
  if (!stop) return undefined;
  const result = await guidanceApi.searchPlaces({ keyword: stop.searchKeyword, city: "上海市", limit: 8 });
  const exact = result.places.find((place) => place.name.includes(stop.building)) ?? result.places[0];
  if (exact) resolvedStops.value = { ...resolvedStops.value, [index]: exact };
  return exact;
}

async function selectStage(index: number, followCurrentProgress = false) {
  const itinerary = importedItinerary.value;
  if (!itinerary || index < 0 || index >= itinerary.stops.length) return;
  selectedStopIndex.value = index;
  route.value = undefined;
  error.value = "";
  try {
    destination.value = await resolveStop(index);
    if (followCurrentProgress) {
      const completedIds = completedItemIdsInRecommendedOrder.value;
      const lastCompletedItemId = completedIds[completedIds.length - 1];
      const completedIndex = lastCompletedItemId
        ? itinerary.stops.findIndex((stop) => stop.itemId === lastCompletedItemId)
        : -1;
      if (completedIndex >= 0) {
        origin.value = await resolveStop(completedIndex);
        outsideRouteLeg.value = false;
      } else {
        origin.value = itinerary.initialOrigin;
        outsideRouteLeg.value = true;
        if (!origin.value) searchTarget.value = "origin";
      }
    } else if (index === 0) {
      origin.value = itinerary.initialOrigin;
      outsideRouteLeg.value = true;
      if (!origin.value) searchTarget.value = "origin";
    } else {
      origin.value = await resolveStop(index - 1);
      outsideRouteLeg.value = false;
    }
  } catch {
    error.value = "当前检查楼栋暂时无法定位，请手动选择地点";
  }
}

async function refreshProgress(showToast = true) {
  if (!importedItinerary.value || refreshingProgress.value) return;
  refreshingProgress.value = true;
  try {
    recommendation.value = await guidanceApi.getTodayExaminationRecommendation();
    const nextIndex = currentStopIndex.value;
    if (nextIndex >= 0) await selectStage(nextIndex, true);
    else {
      selectedStopIndex.value = -1;
      route.value = undefined;
      if (showToast) uni.showToast({ title: "今日检查已全部完成", icon: "success" });
    }
  } catch (cause) {
    if (showToast) uni.showToast({ title: cause instanceof Error ? cause.message : "检查进度刷新失败", icon: "none" });
  } finally {
    refreshingProgress.value = false;
  }
}

function selectPlace(place: GuidanceLocationPoint) {
  if (searchTarget.value === "origin") {
    origin.value = place;
    if (importedItinerary.value && selectedStopIndex.value === 0) {
      importedItinerary.value = { ...importedItinerary.value, initialOrigin: place };
      updateGuidanceRouteItinerary({ initialOrigin: place });
    }
  }
  else if (searchTarget.value === "destination") destination.value = place;
  searchTarget.value = undefined;
  route.value = undefined;
  error.value = "";
}

function chooseMode(mode: GuidanceTravelMode) {
  travelMode.value = mode;
  route.value = undefined;
  if (importedItinerary.value) {
    importedItinerary.value = { ...importedItinerary.value, transportMode: mode };
    updateGuidanceRouteItinerary({ transportMode: mode });
  }
}

function swapPoints() {
  const previous = origin.value;
  origin.value = destination.value;
  destination.value = previous;
  route.value = undefined;
  error.value = "";
}

async function calculateRoute() {
  if (!origin.value || !destination.value || loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    route.value = await guidanceApi.calculateRoute({
      origin: origin.value,
      destination: destination.value,
      mode: effectiveMode.value,
    });
    await nextTick();
    uni.createMapContext("guidance-route-map").includePoints({
      points: route.value.polyline,
      padding: [56, 40, 56, 40],
    });
  } catch (cause) {
    route.value = undefined;
    if (cause instanceof ApiError && cause.code === "SERVICE_UNAVAILABLE") {
      error.value = "路线服务暂不可用，请稍后重试";
    } else if (cause instanceof ApiError && cause.code === "NOT_FOUND") {
      error.value = "没有找到可用路线，请重新选择地点或出行方式";
    } else {
      error.value = cause instanceof Error ? cause.message : "路线计算失败，请稍后重试";
    }
    uni.showToast({ title: error.value, icon: "none" });
  } finally {
    loading.value = false;
  }
}

function modeLabel(mode: GuidanceTravelMode) {
  return ({ walking: "步行", transit: "公交", driving: "驾车" } as Record<GuidanceTravelMode, string>)[mode];
}
</script>

<template>
  <view class="route-page">
    <view class="intro">
      <text class="intro__title">检查导航</text>
      <text class="intro__description">院外首段可选择出行方式；进入医院后，系统按检查进度自动切换下一段楼栋步行路线。</text>
    </view>

    <view v-if="importedItinerary" class="itinerary-panel">
      <view class="itinerary-panel__heading"><view><text>已导入今日顺序</text><text>{{ completedStopCount }}/{{ importedItinerary.stops.length }} 已完成</text></view><button :loading="refreshingProgress" @tap="refreshProgress">刷新检查进度</button></view>
      <scroll-view scroll-x class="itinerary-strip">
        <view class="itinerary-strip__inner">
          <button v-for="(stop, index) in importedItinerary.stops" :key="`${stop.itemId}-${index}`" class="itinerary-stop" :class="{ 'itinerary-stop--current': selectedStopIndex === index, 'itinerary-stop--completed': recommendation && !activeItemIds.has(stop.itemId) }" @tap="selectStage(index)">
            <text class="itinerary-stop__index">{{ index + 1 }}</text><view><text class="itinerary-stop__name">{{ stop.itemName }}</text><text class="itinerary-stop__location">{{ stop.building }} · {{ stop.roomNumber }}室</text></view>
          </button>
        </view>
      </scroll-view>
      <text v-if="currentStopIndex >= 0" class="itinerary-panel__hint">当前导航到第 {{ currentStopIndex + 1 }} 站。检查完成后重新进入本页或点击刷新，系统会自动推进；点击其他阶段只预览路线，不改变检查状态。</text>
      <text v-else class="itinerary-panel__hint itinerary-panel__hint--done">今天的检查已经全部完成。</text>
    </view>

    <view class="location-panel">
      <button class="location-row" @tap="choosePoint('origin')">
        <text class="location-row__badge location-row__badge--origin">起</text>
        <view class="location-row__content">
          <text class="location-row__label">起点</text>
          <text class="location-row__name">{{ origin?.name || "在地图中选择起点" }}</text>
          <text v-if="origin?.address" class="location-row__address">{{ origin.address }}</text>
        </view>
        <text class="location-row__action">{{ origin ? "重选" : "选择" }}</text>
      </button>

      <button class="swap-button" :disabled="!origin && !destination" @tap="swapPoints">交换起终点</button>

      <button class="location-row" @tap="choosePoint('destination')">
        <text class="location-row__badge location-row__badge--destination">终</text>
        <view class="location-row__content">
          <text class="location-row__label">终点</text>
          <text class="location-row__name">{{ destination?.name || "在地图中选择终点" }}</text>
          <text v-if="destination?.address" class="location-row__address">{{ destination.address }}</text>
        </view>
        <text class="location-row__action">{{ destination ? "重选" : "选择" }}</text>
      </button>

      <view v-if="isOutsideStage" class="mode-options">
        <text>院外出行方式</text>
        <view><button v-for="option in travelModes" :key="option.value" :class="{ selected: travelMode === option.value }" @tap="chooseMode(option.value)">{{ option.label }}</button></view>
      </view>
      <view v-else class="internal-mode"><text>院内楼栋间按步行路线计算</text></view>

      <button class="calculate-button" :disabled="!canCalculate" :loading="loading" @tap="calculateRoute">
        {{ loading ? "正在计算" : actionLabel }}
      </button>
      <text v-if="error" class="route-error">{{ error }}</text>
    </view>

    <view v-if="route" class="route-result">
      <view class="route-summary">
        <view>
          <text class="route-summary__label">{{ durationLabel }}</text>
          <text class="route-summary__value">{{ durationText }}</text>
        </view>
        <view>
          <text class="route-summary__label">路线距离</text>
          <text class="route-summary__value">{{ distanceText }}</text>
        </view>
        <text class="route-summary__provider">{{ providerText }}</text>
      </view>

      <map
        id="guidance-route-map"
        class="route-map"
        :latitude="mapCenter.latitude"
        :longitude="mapCenter.longitude"
        :markers="markers"
        :polyline="polylines"
        :include-points="mapPoints"
        :show-location="false"
      />

      <view v-if="route.steps.length" class="steps-panel">
        <text class="steps-panel__title">{{ modeLabel(route.mode) }}指引</text>
        <view v-for="(step, index) in route.steps" :key="`${index}-${step.instruction}`" class="route-step">
          <text class="route-step__index">{{ index + 1 }}</text>
          <view class="route-step__content">
            <text class="route-step__instruction">{{ step.instruction }}</text>
            <text class="route-step__meta">{{ step.distance_meters }} 米</text>
          </view>
        </view>
      </view>
    </view>

    <view v-else class="empty-state">
      <text class="empty-state__title">路线将在这里展示</text>
      <text class="empty-state__description">选择起点、终点和出行方式后，即可查看完整路线。</text>
    </view>

    <PlaceSearchSheet
      :visible="Boolean(searchTarget)"
      :title="searchTarget === 'destination' ? '选择终点' : '选择起点'"
      @close="searchTarget = undefined"
      @select="selectPlace"
    />
  </view>
</template>

<style scoped>
button::after { display: none; }
.route-page { min-height: 100vh; padding: 28rpx 24rpx 80rpx; box-sizing: border-box; background: #f3f6fa; }
.intro { padding: 6rpx 8rpx 24rpx; }
.intro__title,.intro__description { display: block; }
.intro__title { color: #172235; font-size: 40rpx; font-weight: 750; }
.intro__description { margin-top: 10rpx; color: #7d899a; font-size: 24rpx; line-height: 1.6; }
.itinerary-panel { margin-bottom: 22rpx; padding: 22rpx; background: #fff; border: 1rpx solid #dce8f1; border-radius: 22rpx; }.itinerary-panel text { display: block; }.itinerary-panel__heading { display: flex; align-items: center; justify-content: space-between; }.itinerary-panel__heading>view text:first-child { color: #2a394d; font-size: 25rpx; font-weight: 700; }.itinerary-panel__heading>view text:last-child { margin-top:4rpx;color: #168fe4; font-size: 19rpx; }.itinerary-panel__heading button{width:auto;margin:0;padding:0 18rpx;color:#168fe4;font-size:19rpx;line-height:52rpx;background:#edf7fd;border-radius:26rpx}.itinerary-strip { width: 100%; margin-top: 18rpx; white-space: nowrap; }.itinerary-strip__inner { display: inline-flex; gap: 12rpx; }.itinerary-stop { display: flex; align-items: center; min-width: 260rpx;margin:0; padding: 16rpx;color:inherit;text-align:left;line-height:normal; background: #f3f7fa;border:2rpx solid transparent; border-radius: 16rpx; }.itinerary-stop--current{background:#eaf6fd;border-color:#76c6f2}.itinerary-stop--completed{opacity:.5}.itinerary-stop__index { display: flex !important; align-items: center; justify-content: center; width: 38rpx; height: 38rpx; margin-right: 12rpx; color: #fff; font-size: 19rpx; font-weight: 700; background: #168fe4; border-radius: 50%; }.itinerary-stop__name { color: #334156; font-size: 22rpx; font-weight: 680; }.itinerary-stop__location { margin-top: 4rpx; color: #8894a4; font-size: 18rpx; }.itinerary-panel__hint { margin-top: 16rpx; color: #738297; font-size: 20rpx; line-height: 1.5; }.itinerary-panel__hint--done{color:#16896f}
.location-panel,.route-result,.empty-state { background: #fff; border: 1rpx solid #e4eaf1; border-radius: 24rpx; }
.location-panel { padding: 10rpx 22rpx 24rpx; }
.location-row { display: flex; align-items: center; width: 100%; min-height: 126rpx; margin: 0; padding: 20rpx 0; color: inherit; text-align: left; background: transparent; border-radius: 0; }
.location-row__badge { display: flex; align-items: center; justify-content: center; width: 52rpx; height: 52rpx; flex: 0 0 auto; color: #fff; font-size: 22rpx; font-weight: 700; border-radius: 50%; }
.location-row__badge--origin { background: #168fe4; }.location-row__badge--destination { background: #16a085; }
.location-row__content { min-width: 0; flex: 1; margin-left: 20rpx; }
.location-row__label,.location-row__name,.location-row__address { display: block; }
.location-row__label { color: #8b97a8; font-size: 21rpx; }.location-row__name { margin-top: 4rpx; overflow: hidden; color: #263449; font-size: 27rpx; font-weight: 680; text-overflow: ellipsis; white-space: nowrap; }.location-row__address { margin-top: 6rpx; overflow: hidden; color: #8b97a8; font-size: 21rpx; text-overflow: ellipsis; white-space: nowrap; }
.location-row__action { margin-left: 16rpx; color: #168fe4; font-size: 23rpx; }
.swap-button { width: 174rpx; margin: -4rpx auto; padding: 0; color: #557088; font-size: 21rpx; line-height: 48rpx; background: #eef5fa; border-radius: 24rpx; }.swap-button[disabled] { color: #aeb7c1; background: #f2f4f6; }
.calculate-button { width: 100%; margin: 20rpx 0 0; color: #fff; font-size: 27rpx; font-weight: 680; line-height: 84rpx; background: #168fe4; border-radius: 18rpx; }.calculate-button[disabled] { color: #a9b3bf; background: #e7ebef; }
.mode-options{margin:12rpx 0 0;padding:18rpx;background:#f6f9fb;border-radius:16rpx}.mode-options>text{display:block;color:#7e8b9c;font-size:20rpx}.mode-options>view{display:grid;grid-template-columns:repeat(3,1fr);gap:10rpx;margin-top:12rpx}.mode-options button{margin:0;color:#68768a;font-size:21rpx;line-height:56rpx;background:#fff;border:1rpx solid #e0e7ed;border-radius:28rpx}.mode-options button.selected{color:#087fc9;background:#e7f5fd;border-color:#76c6f2}.internal-mode{margin:10rpx 0 0;padding:16rpx;color:#58758c;font-size:20rpx;text-align:center;background:#eef6fa;border-radius:14rpx}
.route-error { display: block; margin-top: 18rpx; color: #c94d5e; font-size: 22rpx; text-align: center; }
.route-result { margin-top: 22rpx; overflow: hidden; }
.route-summary { position: relative; display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 22rpx; padding: 24rpx; }
.route-summary__label,.route-summary__value { display: block; }.route-summary__label { color: #8a96a7; font-size: 21rpx; }.route-summary__value { margin-top: 5rpx; color: #233146; font-size: 30rpx; font-weight: 730; }.route-summary__provider { grid-column: 1 / -1; color: #8995a5; font-size: 19rpx; }
.route-map { width: 100%; height: 560rpx; }
.steps-panel { padding: 26rpx 24rpx 10rpx; }.steps-panel__title { color: #233146; font-size: 28rpx; font-weight: 720; }
.route-step { display: flex; padding: 24rpx 0; border-bottom: 1rpx solid #edf0f4; }.route-step:last-child { border-bottom: 0; }.route-step__index { display: flex; align-items: center; justify-content: center; width: 42rpx; height: 42rpx; flex: 0 0 auto; color: #168fe4; font-size: 20rpx; font-weight: 700; background: #e9f5fc; border-radius: 50%; }.route-step__content { min-width: 0; margin-left: 18rpx; }.route-step__instruction,.route-step__meta { display: block; }.route-step__instruction { color: #354256; font-size: 24rpx; line-height: 1.55; }.route-step__meta { margin-top: 6rpx; color: #909baa; font-size: 20rpx; }
.empty-state { margin-top: 22rpx; padding: 58rpx 30rpx; text-align: center; }.empty-state__title,.empty-state__description { display: block; }.empty-state__title { color: #546275; font-size: 27rpx; font-weight: 680; }.empty-state__description { margin-top: 12rpx; color: #98a2b0; font-size: 21rpx; line-height: 1.5; }
</style>
