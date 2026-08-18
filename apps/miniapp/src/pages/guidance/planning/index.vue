<script setup lang="ts">
import { useSmartAppointmentPlanning } from "@/features/guidance/useSmartAppointmentPlanning";
import { formatEstimatedDuration } from "@/utils/appointmentManagement";

const {
  departments, activeDepartmentId, selectedItemIds, sessionChoices, selectedAvailability,
  plans, loading, loadingItems, generating, confirmingPlanId, errorMessage,
  currentItems, currentDepartment, selectedItems, candidateDates, selectedDates, canGenerate,
  chooseDepartment, toggleItem, toggleDate, chooseSession, generatePlans, confirmPlan,
  openTodayGuidance, sessionLabel, floorLabel, planKey,
} = useSmartAppointmentPlanning();
</script>

<template>
  <view class="planning-page">
    <view class="guidance-switch"><button class="active">提前规划</button><button @tap="openTodayGuidance">当日导诊</button></view>
    <view class="intro">
      <text class="intro__eyebrow">智能导诊</text>
      <text class="intro__title">智能预约</text>
      <text class="intro__description">选择检查项目和本周可接受的日期，系统会按准备要求与检查先后关系给出少量方案。</text>
    </view>

    <view class="step-card">
      <view class="step-heading"><text class="step-heading__number">1</text><view><text class="step-heading__title">选择检查项目</text><text class="step-heading__meta">已选择 {{ selectedItemIds.length }} 项</text></view></view>
      <scroll-view scroll-x class="department-strip">
        <view class="department-strip__inner">
          <button v-for="entry in departments" :key="entry.department.departmentId" class="department-chip" :class="{ selected: activeDepartmentId === entry.department.departmentId }" @tap="chooseDepartment(entry.department.departmentId)">{{ entry.department.name }}</button>
        </view>
      </scroll-view>
      <text v-if="loading || loadingItems" class="state-text">正在加载检查项目…</text>
      <view v-else class="item-list">
        <button v-for="item in currentItems" :key="item.itemId" class="item-card" :class="{ selected: selectedItemIds.includes(item.itemId) }" @tap="toggleItem(item.itemId)">
          <view class="item-card__check">{{ selectedItemIds.includes(item.itemId) ? "✓" : "" }}</view>
          <view class="item-card__content"><text class="item-card__name">{{ item.name }}</text><text class="item-card__description">{{ item.description || `${currentDepartment?.department.name || "当前科室"}检查项目` }}</text></view>
          <text class="item-card__duration">{{ formatEstimatedDuration(item.estimatedDurationMinutes) }}</text>
        </button>
        <text v-if="!currentItems.length" class="state-text">当前科室暂无可预约项目</text>
      </view>
      <view v-if="selectedItems.length" class="selected-summary"><text>本次规划</text><text>{{ selectedItems.map((item) => item.name).join("、") }}</text></view>
    </view>

    <view class="step-card">
      <view class="step-heading"><text class="step-heading__number">2</text><view><text class="step-heading__title">选择可接受日期</text><text class="step-heading__meta">日期可以不连续</text></view></view>
      <view class="date-grid">
        <button v-for="date in candidateDates" :key="date.value" class="date-card" :class="{ selected: selectedDates.includes(date.value) }" @tap="toggleDate(date.value)"><text>{{ date.weekday }}</text><text>{{ date.label }}</text></button>
      </view>
      <view v-if="selectedDates.length" class="availability-list">
        <view v-for="date in selectedDates" :key="date" class="availability-row">
          <text>{{ date.slice(5).replace('-', '/') }}</text>
          <view class="session-options">
            <button v-for="choice in sessionChoices" :key="choice[0]" :class="{ selected: selectedAvailability[date] === choice[0] }" @tap="chooseSession(date, choice[0])">{{ choice[1] }}</button>
          </view>
        </view>
      </view>
    </view>

    <button class="generate-button" :disabled="!canGenerate" :loading="generating" @tap="generatePlans">{{ generating ? "正在生成" : "生成预约方案" }}</button>
    <view v-if="generating" class="generation-state"><text>正在生成预约方案…</text><text>系统正在核对项目规则、可预约窗口与楼栋安排</text></view>
    <text v-else-if="errorMessage" class="error-message">{{ errorMessage }}</text>

    <view v-else-if="plans.length" class="plans-section">
      <view class="section-heading"><text>推荐方案</text><text>选择一个方案即可一次完成预约</text></view>
      <view v-for="(plan, planIndex) in plans" :key="planKey(plan)" class="plan-card" :class="{ 'plan-card--primary': planIndex === 0 }">
        <view class="plan-card__heading"><view><text class="plan-card__title">{{ plan.title }}</text><text class="plan-card__summary">{{ plan.summary }}</text></view><text v-if="planIndex === 0" class="recommended-tag">推荐</text></view>
        <view v-for="(item, index) in plan.items" :key="`${item.item_id}-${item.service_date}-${item.session}`" class="plan-item">
          <text class="plan-item__index">{{ index + 1 }}</text>
          <view class="plan-item__content"><text class="plan-item__name">{{ item.item_name }}</text><text class="plan-item__meta">{{ item.service_date }} {{ sessionLabel(item.session) }} · {{ item.building }} {{ floorLabel(item.floor_number) }} {{ item.room_number }}室</text><text v-if="item.planned_start_time && item.planned_end_time" class="plan-item__time">规划参考 {{ item.planned_start_time }}–{{ item.planned_end_time }}</text><text v-if="item.travel_minutes" class="plan-item__travel">前往此处约 {{ item.travel_minutes }} 分钟{{ item.travel_time_estimated ? "（估算）" : "" }}</text><text class="plan-item__reason">{{ item.reason }}</text></view>
          <text class="plan-item__duration">约 {{ item.estimated_duration_minutes }} 分钟</text>
        </view>
        <button class="confirm-button" :loading="confirmingPlanId === plan.plan_id" :disabled="Boolean(confirmingPlanId)" @tap="confirmPlan(plan)">采用此方案</button>
      </view>
    </view>
  </view>
</template>

<style scoped>
button::after { display: none; }
.planning-page { min-height: 100vh; padding: 28rpx 24rpx 80rpx; box-sizing: border-box; background: #f3f6fa; }
.guidance-switch{display:grid;grid-template-columns:1fr 1fr;margin-bottom:22rpx;padding:6rpx;background:#e7edf3;border-radius:20rpx}.guidance-switch button{margin:0;color:#748196;font-size:22rpx;line-height:62rpx;background:transparent;border-radius:16rpx}.guidance-switch button.active{color:#087fc9;font-weight:700;background:#fff}
.intro { padding: 6rpx 8rpx 24rpx; }.intro text,.step-heading text,.item-card text,.plan-card text,.selected-summary text,.section-heading text { display: block; }
.intro__eyebrow { color: #168fe4; font-size: 21rpx; font-weight: 700; }.intro__title { margin-top: 6rpx; color: #172235; font-size: 40rpx; font-weight: 750; }.intro__description { margin-top: 10rpx; color: #7d899a; font-size: 24rpx; line-height: 1.6; }
.step-card,.plan-card { margin-bottom: 22rpx; padding: 24rpx; background: #fff; border: 1rpx solid #e3e9f0; border-radius: 24rpx; }
.step-heading { display: flex; align-items: center; }.step-heading__number { display: flex !important; align-items: center; justify-content: center; width: 48rpx; height: 48rpx; margin-right: 16rpx; color: #fff; font-size: 23rpx; font-weight: 750; background: #168fe4; border-radius: 50%; }.step-heading__title { color: #223147; font-size: 29rpx; font-weight: 720; }.step-heading__meta { margin-top: 3rpx; color: #8b97a7; font-size: 20rpx; }
.department-strip { width: 100%; margin: 24rpx 0 18rpx; white-space: nowrap; }.department-strip__inner { display: inline-flex; gap: 12rpx; }.department-chip { margin: 0; padding: 0 24rpx; color: #657287; font-size: 22rpx; line-height: 58rpx; background: #f1f4f7; border-radius: 29rpx; }.department-chip.selected { color: #087fc9; background: #e6f4fd; }
.item-list { display: grid; gap: 14rpx; }.item-card { display: flex; align-items: center; width: 100%; margin: 0; padding: 20rpx; color: inherit; text-align: left; background: #fafbfd; border: 2rpx solid #edf1f5; border-radius: 18rpx; }.item-card.selected { background: #f0f8fe; border-color: #8bcdf3; }.item-card__check { display: flex; align-items: center; justify-content: center; width: 38rpx; height: 38rpx; flex: 0 0 auto; color: #fff; font-size: 20rpx; background: #fff; border: 2rpx solid #b7c2ce; border-radius: 10rpx; }.item-card.selected .item-card__check { background: #168fe4; border-color: #168fe4; }.item-card__content { min-width: 0; flex: 1; margin-left: 16rpx; }.item-card__name { color: #29374a; font-size: 25rpx; font-weight: 690; }.item-card__description { margin-top: 5rpx; overflow: hidden; color: #8b96a6; font-size: 20rpx; text-overflow: ellipsis; white-space: nowrap; }.item-card__duration { margin-left: 12rpx; color: #168fe4; font-size: 20rpx; }
.selected-summary { margin-top: 18rpx; padding: 18rpx; color: #627085; font-size: 21rpx; line-height: 1.55; background: #f4f7fa; border-radius: 16rpx; }.selected-summary text + text { margin-top: 5rpx; color: #344257; }
.date-grid { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); gap: 12rpx; margin-top: 22rpx; }.date-card { margin: 0; padding: 15rpx 4rpx; color: #647187; font-size: 21rpx; line-height: 1.45; background: #f4f6f8; border: 2rpx solid transparent; border-radius: 16rpx; }.date-card text { display: block; }.date-card.selected { color: #087fc9; background: #eaf6fd; border-color: #7cc7f1; }
.availability-list{display:grid;gap:12rpx;margin-top:18rpx}.availability-row{display:flex;align-items:center;justify-content:space-between;gap:16rpx;padding:14rpx 16rpx;background:#f7f9fb;border-radius:16rpx}.availability-row>text{color:#46556a;font-size:21rpx;font-weight:650}.session-options{display:flex;gap:8rpx}.session-options button{width:auto;margin:0;padding:0 18rpx;color:#718095;font-size:19rpx;line-height:50rpx;background:#fff;border:1rpx solid #e1e7ed;border-radius:25rpx}.session-options button.selected{color:#087fc9;background:#e5f4fd;border-color:#78c6f1}
.generate-button,.confirm-button { width: 100%; color: #fff; font-size: 27rpx; font-weight: 700; background: #168fe4; border-radius: 18rpx; }.generate-button { line-height: 88rpx; }.generate-button[disabled],.confirm-button[disabled] { color: #abb5c0; background: #e5eaf0; }.error-message,.state-text { display: block; padding: 24rpx 8rpx; color: #c25464; font-size: 22rpx; text-align: center; }.state-text { color: #8e99a8; }
.generation-state{margin-top:20rpx;padding:34rpx 24rpx;text-align:center;background:#fff;border:1rpx solid #e3e9f0;border-radius:22rpx}.generation-state text{display:block;color:#526176;font-size:24rpx;font-weight:680}.generation-state text+text{margin-top:10rpx;color:#929dac;font-size:20rpx;font-weight:400}
.plans-section { margin-top: 30rpx; }.section-heading { padding: 0 8rpx 18rpx; }.section-heading text:first-child { color: #223147; font-size: 31rpx; font-weight: 730; }.section-heading text:last-child { margin-top: 5rpx; color: #8c97a6; font-size: 21rpx; }.plan-card--primary { border-color: #94d1f4; }.plan-card__heading { display: flex; align-items: flex-start; justify-content: space-between; }.plan-card__title { color: #223147; font-size: 29rpx; font-weight: 730; }.plan-card__summary { margin-top: 5rpx; color: #8b96a5; font-size: 21rpx; }.recommended-tag { padding: 7rpx 14rpx; color: #087fc9; font-size: 19rpx; background: #e6f5fd; border-radius: 18rpx; }
.plan-item { display: flex; align-items: flex-start; padding: 24rpx 0; border-bottom: 1rpx solid #edf1f5; }.plan-item__index { display: flex !important; align-items: center; justify-content: center; width: 42rpx; height: 42rpx; flex: 0 0 auto; color: #168fe4; font-size: 20rpx; font-weight: 720; background: #e9f5fc; border-radius: 50%; }.plan-item__content { min-width: 0; flex: 1; margin-left: 16rpx; }.plan-item__name { color: #2c394c; font-size: 25rpx; font-weight: 690; }.plan-item__meta,.plan-item__time,.plan-item__travel,.plan-item__reason { margin-top: 6rpx; color: #7f8b9c; font-size: 20rpx; line-height: 1.5; }.plan-item__time { color: #246f9f; font-weight: 650; }.plan-item__travel { color: #718196; }.plan-item__reason { color: #4f7794; }.plan-item__duration { margin-left: 10rpx; color: #68758a; font-size: 19rpx; }.confirm-button { margin-top: 22rpx; line-height: 78rpx; }
</style>
