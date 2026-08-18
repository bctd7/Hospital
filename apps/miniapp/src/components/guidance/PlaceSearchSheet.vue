<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";

import { ApiError } from "@/api/client";
import { guidanceApi } from "@/api/guidance";
import type { GuidanceLocationPoint } from "@/types/guidance";

const props = defineProps<{
  visible: boolean;
  title: string;
}>();

const emit = defineEmits<{
  close: [];
  select: [place: GuidanceLocationPoint];
}>();

const keyword = ref("");
const places = ref<GuidanceLocationPoint[]>([]);
const searching = ref(false);
const searched = ref(false);
const error = ref("");
const mapMounted = ref(false);
const mapReady = ref(false);
let mapMountTimer: ReturnType<typeof setTimeout> | undefined;

interface PreviewMarker {
  id: number;
  latitude: number;
  longitude: number;
  title: string;
  width: number;
  height: number;
  label: {
    content: string;
    color: string;
    fontSize: number;
    borderRadius: number;
    bgColor: string;
    padding: number;
  };
}

const previewCenter = computed(() => places.value[0] ?? {
  latitude: 31.2304,
  longitude: 121.4737,
});
const previewPoints = computed(() => places.value.map((place) => ({
  latitude: place.latitude,
  longitude: place.longitude,
})));
const previewMarkers = computed<PreviewMarker[]>(() => places.value.map((place, index) => ({
  id: index + 1,
  latitude: place.latitude,
  longitude: place.longitude,
  title: place.name,
  width: 26,
  height: 26,
  label: {
    content: String(index + 1),
    color: "#ffffff",
    fontSize: 11,
    borderRadius: 13,
    bgColor: "#168fe4",
    padding: 6,
  },
})));

watch(() => props.visible, (visible) => {
  if (mapMountTimer) clearTimeout(mapMountTimer);
  mapMounted.value = false;
  mapReady.value = false;
  if (!visible) return;
  keyword.value = "";
  places.value = [];
  searched.value = false;
  error.value = "";
  mapMountTimer = setTimeout(() => {
    mapMounted.value = true;
  }, 180);
}, { immediate: true });

onBeforeUnmount(() => {
  if (mapMountTimer) clearTimeout(mapMountTimer);
});

function markMapReady() {
  mapReady.value = true;
}

async function search() {
  const value = keyword.value.trim();
  if (!value || searching.value) {
    if (!value) uni.showToast({ title: "请输入地点名称或地址", icon: "none" });
    return;
  }
  searching.value = true;
  error.value = "";
  try {
    const result = await guidanceApi.searchPlaces({ keyword: value, city: "上海市", limit: 15 });
    places.value = result.places;
    searched.value = true;
  } catch (cause) {
    places.value = [];
    searched.value = true;
    if (cause instanceof ApiError && cause.code === "SERVICE_UNAVAILABLE") {
      error.value = "高德地点服务暂不可用，请稍后重试";
    } else {
      error.value = cause instanceof Error ? cause.message : "地点搜索失败，请稍后重试";
    }
  } finally {
    searching.value = false;
  }
}
</script>

<template>
  <view v-if="visible" class="place-search" @tap="emit('close')">
    <view class="place-search__panel" @tap.stop>
      <view class="place-search__header">
        <view>
          <text class="place-search__title">{{ title }}</text>
          <text class="place-search__scope">高德地点 · 上海市</text>
        </view>
        <button class="place-search__close" @tap="emit('close')">关闭</button>
      </view>

      <view class="place-search__bar">
        <input
          v-model="keyword"
          class="place-search__input"
          confirm-type="search"
          placeholder="输入地点名称或详细地址"
          @confirm="search"
        />
        <button class="place-search__submit" :loading="searching" @tap="search">搜索</button>
      </view>

      <view class="place-preview">
        <view class="place-preview__heading">
          <text class="place-preview__title">候选位置预览</text>
          <text class="place-preview__hint">地图编号与下方结果对应</text>
        </view>
        <view class="place-preview__stage">
          <view v-if="!mapReady" class="place-preview__placeholder">地图加载中</view>
          <map
            v-if="mapMounted"
            class="place-preview__map"
            :class="{ 'place-preview__map--ready': mapReady }"
            :latitude="previewCenter.latitude"
            :longitude="previewCenter.longitude"
            :markers="previewMarkers"
            :include-points="previewPoints"
            :show-location="false"
            @updated="markMapReady"
          />
        </view>
      </view>

      <scroll-view class="place-search__results" scroll-y>
        <button
          v-for="(place, index) in places"
          :key="`${place.provider_place_id}-${place.longitude}-${place.latitude}`"
          class="place-result"
          @tap="emit('select', place)"
        >
          <text class="place-result__number">{{ index + 1 }}</text>
          <view class="place-result__content">
            <text class="place-result__name">{{ place.name }}</text>
            <text class="place-result__address">{{ place.address || "暂无详细地址" }}</text>
          </view>
        </button>
        <view v-if="error" class="place-search__state place-search__state--error">{{ error }}</view>
        <view v-else-if="searched && !places.length" class="place-search__state">没有找到相关地点，请补充更完整的名称或地址</view>
        <view v-else-if="!searched" class="place-search__state">可搜索小区、道路、医院或具体楼栋</view>
      </scroll-view>
    </view>
  </view>
</template>

<style scoped>
button::after { display: none; }
.place-search { position: fixed; z-index: 50; inset: 0; display: flex; align-items: flex-end; background: rgba(21, 34, 51, .36); }
.place-search__panel { display: flex; width: 100%; height: 82vh; padding: 28rpx 24rpx calc(28rpx + env(safe-area-inset-bottom)); box-sizing: border-box; flex-direction: column; background: #f4f7fa; border-radius: 30rpx 30rpx 0 0; }
.place-search__header { display: flex; align-items: center; justify-content: space-between; padding: 0 4rpx 22rpx; }
.place-search__title,.place-search__scope { display: block; }.place-search__title { color: #1f2d40; font-size: 32rpx; font-weight: 730; }.place-search__scope { margin-top: 6rpx; color: #8b97a8; font-size: 21rpx; }
.place-search__close { margin: 0; padding: 0 12rpx; color: #607187; font-size: 23rpx; line-height: 56rpx; background: transparent; }
.place-search__bar { display: flex; gap: 14rpx; padding: 12rpx; background: #fff; border: 1rpx solid #e2e8ef; border-radius: 20rpx; }
.place-search__input { min-width: 0; height: 68rpx; flex: 1; padding: 0 18rpx; color: #27364a; font-size: 25rpx; background: #f3f6f9; border-radius: 14rpx; }
.place-search__submit { width: 126rpx; margin: 0; padding: 0; color: #fff; font-size: 24rpx; line-height: 68rpx; background: #168fe4; border-radius: 14rpx; }
.place-preview { flex: 0 0 auto; margin-top: 18rpx; overflow: hidden; background: #fff; border: 1rpx solid #e3e9ef; border-radius: 18rpx; }
.place-preview__heading { display: flex; align-items: center; justify-content: space-between; padding: 16rpx 18rpx; }
.place-preview__title { color: #354256; font-size: 23rpx; font-weight: 680; }.place-preview__hint { color: #8d99a8; font-size: 19rpx; }
.place-preview__stage { position: relative; width: 100%; height: 280rpx; overflow: hidden; background: #eef3f7; }
.place-preview__placeholder { position: absolute; z-index: 1; inset: 0; display: flex; align-items: center; justify-content: center; color: #8d99a8; font-size: 22rpx; background: #eef3f7; }
.place-preview__map { position: relative; z-index: 2; width: 100%; height: 280rpx; opacity: 0; }
.place-preview__map--ready { opacity: 1; }
.place-search__results { min-height: 0; height: auto; flex: 1; margin-top: 18rpx; }
.place-result { display: flex; align-items: center; width: 100%; margin: 0 0 12rpx; padding: 24rpx 22rpx; color: inherit; text-align: left; background: #fff; border: 1rpx solid #e3e9ef; border-radius: 18rpx; }
.place-result__number { display: flex; align-items: center; justify-content: center; width: 42rpx; height: 42rpx; flex: 0 0 auto; color: #168fe4; font-size: 20rpx; font-weight: 720; background: #e9f5fc; border-radius: 50%; }
.place-result__content { min-width: 0; flex: 1; margin-left: 18rpx; }
.place-result__name,.place-result__address { display: block; }.place-result__name { color: #263449; font-size: 27rpx; font-weight: 680; }.place-result__address { margin-top: 9rpx; color: #8793a3; font-size: 22rpx; line-height: 1.45; }
.place-search__state { padding: 72rpx 24rpx; color: #96a1af; font-size: 23rpx; line-height: 1.6; text-align: center; }.place-search__state--error { color: #c94d5e; }
</style>
