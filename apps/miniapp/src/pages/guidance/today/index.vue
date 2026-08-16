<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { guidanceApi } from "@/api/guidance";
import type { SmartAppointmentPlanItem, TodayExaminationRecommendation } from "@/types/guidance";
import { saveGuidanceRouteItinerary } from "@/utils/guidanceRouteImport";

const recommendation = ref<TodayExaminationRecommendation>();
const loading = ref(false);
const errorMessage = ref("");

const routeItems = computed(() => recommendation.value?.stages
  .filter((stage) => stage.status !== "completed")
  .flatMap((stage) => stage.items) ?? []);

onMounted(loadRecommendation);

async function loadRecommendation() {
  loading.value = true;
  errorMessage.value = "";
  try {
    recommendation.value = await guidanceApi.getTodayExaminationRecommendation();
  } catch (error) {
    errorMessage.value = error instanceof Error && error.message.trim() ? error.message : "今日检查顺序加载失败";
  } finally {
    loading.value = false;
  }
}

function adoptRecommendation() {
  const value = recommendation.value;
  if (!value || !routeItems.value.length) return;
  uni.showModal({
    title: "采用今日推荐",
    content: "是否将该方案导入检查导航？导入后请先选择院外出发地址。",
    confirmText: "导入地图",
    cancelText: "暂不导入",
    success: (result) => {
      if (!result.confirm) return;
      saveGuidanceRouteItinerary(value.service_date, routeItems.value);
      void uni.navigateTo({ url: "/pages/guidance/route/index?source=today" });
    },
  });
}

function stageTone(status: string) {
  if (status === "in_progress") return "active";
  if (status === "called") return "called";
  if (status === "completed") return "completed";
  return "waiting";
}
function statusLabel(status: string) {
  return ({ in_progress: "检查中", called: "叫号中", waiting: "待检查", completed: "已结束" } as Record<string, string>)[status] ?? status;
}
function locationText(item: SmartAppointmentPlanItem) { return `${item.building} · ${item.floor_number}层 · ${item.room_number}室`; }
</script>

<template>
  <view class="today-page">
    <view class="intro"><text class="intro__eyebrow">当日动态建议</text><text class="intro__title">今天怎么检查</text><text class="intro__description">系统按已预约、已完成和当前检查状态静默更新顺序；这是建议，现场安排优先。</text></view>
    <view v-if="loading" class="state-card"><text>正在整理今天的检查顺序…</text></view>
    <view v-else-if="errorMessage" class="state-card state-card--error"><text>{{ errorMessage }}</text><button @tap="loadRecommendation">重新加载</button></view>
    <view v-else-if="!recommendation?.stages.length" class="state-card"><text class="state-card__title">今天没有检查安排</text><text>预约当天再来查看，系统会结合当前状态生成建议。</text></view>
    <template v-else>
      <view class="date-card"><text>检查日期</text><text>{{ recommendation.service_date }}</text><text>打开页面时自动刷新</text></view>
      <view class="timeline">
        <view v-for="stage in recommendation.stages" :key="stage.stage_no" class="stage-card" :class="`stage-card--${stageTone(stage.status)}`">
          <view class="stage-card__rail"><text>{{ stage.stage_no }}</text></view>
          <view class="stage-card__body">
            <view class="stage-card__heading"><view><text class="stage-card__status">{{ statusLabel(stage.status) }}</text><text class="stage-card__title">{{ stage.title }}</text></view><text class="stage-card__count">{{ stage.items.length }} 项</text></view>
            <text class="stage-card__focus">{{ stage.focus }}</text>
            <view v-for="item in stage.items" :key="`${item.item_id}-${item.room_id}`" class="stage-item">
              <view><text class="stage-item__name">{{ item.item_name }}</text><text class="stage-item__location">{{ locationText(item) }}</text></view>
              <text class="stage-item__duration">约 {{ item.estimated_duration_minutes }} 分钟</text>
            </view>
          </view>
        </view>
      </view>
      <view v-if="routeItems.length" class="bottom-action"><view><text>采用这份顺序建议</text><text>可继续导入地图并选择院外起点</text></view><button @tap="adoptRecommendation">采用方案</button></view>
    </template>
  </view>
</template>

<style scoped>
button::after { display: none; }
.today-page { min-height: 100vh; padding: 28rpx 24rpx 180rpx; box-sizing: border-box; background: #f3f6fa; }.today-page text { display: block; }
.intro { padding: 6rpx 8rpx 24rpx; }.intro__eyebrow { color: #168fe4; font-size: 21rpx; font-weight: 700; }.intro__title { margin-top: 6rpx; color: #172235; font-size: 40rpx; font-weight: 750; }.intro__description { margin-top: 10rpx; color: #7d899a; font-size: 24rpx; line-height: 1.6; }
.state-card,.date-card,.stage-card { background: #fff; border: 1rpx solid #e3e9f0; border-radius: 22rpx; }.state-card { padding: 56rpx 28rpx; color: #8792a1; font-size: 23rpx; line-height: 1.6; text-align: center; }.state-card__title { margin-bottom: 8rpx; color: #344257; font-size: 28rpx; font-weight: 700; }.state-card--error { color: #c25464; }.state-card button { width: 220rpx; margin-top: 22rpx; color: #168fe4; font-size: 22rpx; background: #ebf6fd; border-radius: 30rpx; }
.date-card { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 18rpx; padding: 22rpx 24rpx; color: #8591a2; font-size: 20rpx; }.date-card text:nth-child(2) { color: #27364a; font-size: 27rpx; font-weight: 720; }.timeline { display: grid; gap: 18rpx; margin-top: 20rpx; }.stage-card { display: flex; overflow: hidden; }.stage-card__rail { display: flex; align-items: flex-start; justify-content: center; width: 68rpx; flex: 0 0 auto; padding-top: 24rpx; background: #eef6fb; }.stage-card__rail text { display: flex; align-items: center; justify-content: center; width: 38rpx; height: 38rpx; color: #168fe4; font-size: 19rpx; font-weight: 730; background: #fff; border-radius: 50%; }.stage-card--active .stage-card__rail { background: #e6f7f3; }.stage-card--active .stage-card__rail text { color: #159174; }.stage-card--called .stage-card__rail { background: #fff4df; }.stage-card--called .stage-card__rail text { color: #b9780d; }.stage-card--completed { opacity: .72; }.stage-card__body { min-width: 0; flex: 1; padding: 24rpx; }.stage-card__heading { display: flex; align-items: flex-start; justify-content: space-between; }.stage-card__status { color: #168fe4; font-size: 19rpx; font-weight: 720; }.stage-card__title { margin-top: 5rpx; color: #27364a; font-size: 28rpx; font-weight: 730; }.stage-card__count { color: #8995a5; font-size: 20rpx; }.stage-card__focus { margin-top: 12rpx; color: #718094; font-size: 21rpx; line-height: 1.55; }.stage-item { display: flex; align-items: flex-start; justify-content: space-between; gap: 14rpx; margin-top: 18rpx; padding-top: 18rpx; border-top: 1rpx solid #edf1f5; }.stage-item__name { color: #344257; font-size: 24rpx; font-weight: 680; }.stage-item__location { margin-top: 6rpx; color: #8b96a5; font-size: 20rpx; }.stage-item__duration { flex: 0 0 auto; color: #627287; font-size: 19rpx; }
.bottom-action { position: fixed; right: 0; bottom: 0; left: 0; display: flex; align-items: center; gap: 20rpx; padding: 20rpx 24rpx calc(20rpx + env(safe-area-inset-bottom)); background: #fff; border-top: 1rpx solid #e6ebf0; }.bottom-action view { min-width: 0; flex: 1; }.bottom-action view text:first-child { color: #2c3a4d; font-size: 25rpx; font-weight: 700; }.bottom-action view text:last-child { margin-top: 4rpx; color: #8995a5; font-size: 19rpx; }.bottom-action button { width: 190rpx; margin: 0; color: #fff; font-size: 24rpx; font-weight: 700; line-height: 72rpx; background: #168fe4; border-radius: 36rpx; }
</style>
