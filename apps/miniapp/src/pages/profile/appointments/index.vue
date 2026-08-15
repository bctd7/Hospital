<script setup lang="ts">
import { onHide, onLoad, onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { patientAppointmentApi } from "@/api/appointment";
import { ApiError } from "@/api/client";
import AppPage from "@/components/layout/AppPage.vue";
import type { AppointmentListView, PatientBooking, PatientBookingStatus } from "@/types/appointment";
import { formatEstimatedDuration } from "@/utils/appointmentManagement";

const bookings = ref<PatientBooking[]>([]);
const loading = ref(false);
const errorMessage = ref("");
const deletingId = ref("");
const checkingInId = ref("");
const focusedBookingId = ref("");
const view = ref<AppointmentListView>("active");
const isCompletedView = computed(() => view.value === "completed");
const nowTick = ref(Date.now());
let clockTimer: ReturnType<typeof setInterval> | undefined;
let lastExpiredRefreshAt = 0;

onLoad((query) => {
  view.value = query?.view === "completed" ? "completed" : "active";
  focusedBookingId.value = decode(query?.booking_id);
  uni.setNavigationBarTitle({ title: isCompletedView.value ? "检查记录" : "我的预约" });
});

onShow(() => {
  nowTick.value = Date.now();
  if (clockTimer) clearInterval(clockTimer);
  clockTimer = setInterval(tickBookingClock, 1000);
  void loadBookings();
});
onHide(() => { if (clockTimer) clearInterval(clockTimer); clockTimer = undefined; });

function tickBookingClock() {
  nowTick.value = Date.now();
  const hasExpiredCall = bookings.value.some((booking) => booking.status === "called"
    && Boolean(booking.callDeadline) && nowTick.value >= Date.parse(booking.callDeadline!));
  if (!hasExpiredCall || nowTick.value - lastExpiredRefreshAt < 2000) return;
  lastExpiredRefreshAt = nowTick.value;
  void loadBookings();
}

async function loadBookings() {
  if (loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    const values = (await patientAppointmentApi.listMyBookings(1, 100, view.value)).items;
    bookings.value = focusedBookingId.value
      ? [...values].sort((left, right) => Number(right.bookingId === focusedBookingId.value) - Number(left.bookingId === focusedBookingId.value))
      : values;
  } catch (error) {
    errorMessage.value = messageOf(error, "预约记录加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

function requestDelete(booking: PatientBooking) {
  if (booking.status !== "confirmed" || deletingId.value) return;
  uni.showModal({
    title: "删除预约",
    content: `确认删除 ${booking.serviceDate} 的“${booking.itemName}”预约？`,
    success: (result) => { if (result.confirm) void deleteBooking(booking); },
  });
}

async function deleteBooking(booking: PatientBooking) {
  deletingId.value = booking.bookingId;
  try {
    await patientAppointmentApi.deleteMyBooking(booking.bookingId, "患者主动删除");
    bookings.value = bookings.value.filter((value) => value.bookingId !== booking.bookingId);
    uni.showToast({ title: "已删除", icon: "success" });
  } catch (error) {
    uni.showModal({ title: "删除失败", content: messageOf(error, "请稍后重试"), showCancel: false });
  } finally {
    deletingId.value = "";
  }
}

async function checkIn(booking: PatientBooking) {
  if (!canCheckIn(booking) || checkingInId.value) return;
  checkingInId.value = booking.bookingId;
  try {
    const updated = await patientAppointmentApi.checkIn(booking);
    bookings.value = bookings.value.map((value) => value.bookingId === updated.bookingId ? updated : value);
    uni.showToast({ title: "报到成功", icon: "success" });
  } catch (error) {
    uni.showModal({ title: "检查报到失败", content: messageOf(error, "请刷新后重试"), showCancel: false });
  } finally {
    checkingInId.value = "";
  }
}

function canCheckIn(booking: PatientBooking) {
  if (booking.status !== "confirmed") return false;
  const now = Date.now();
  return now >= bookingInstant(booking, booking.itemStartTime) && now < bookingInstant(booking, booking.bookingCutoffTime);
}
function bookingInstant(booking: PatientBooking, clock: string) { return Date.parse(`${booking.serviceDate}T${clock}+08:00`); }
function checkInHint(booking: PatientBooking) {
  if (booking.status !== "confirmed") return "";
  const now = Date.now();
  if (now < bookingInstant(booking, booking.itemStartTime)) return `${booking.itemStartTime.slice(0, 5)} 开放检查报到`;
  if (now >= bookingInstant(booking, booking.bookingCutoffTime)) return "已停止检查报到";
  return `请在 ${booking.bookingCutoffTime.slice(0, 5)} 前完成报到`;
}
function calledCountdown(booking: PatientBooking) {
  if (!booking.callDeadline) return "请立即前往检查房间";
  const remaining = Math.max(0, Math.ceil((Date.parse(booking.callDeadline) - nowTick.value) / 1000));
  return remaining ? `请在 ${remaining} 秒内到达房间` : "叫号时间已到，请等待状态更新";
}

function messageOf(error: unknown, fallback: string) {
	if (error instanceof ApiError && error.code === "BOOKING_WINDOW_CLOSED") return "当前检查尚未开放报到，或已经超过停止报到时间";
  return error instanceof Error && error.message.trim() ? error.message : fallback;
}
function decode(value?: string) { try { return value ? decodeURIComponent(value) : ""; } catch { return ""; } }
function sessionLabel(value: string) { return value === "morning" ? "上午" : "下午"; }
function statusLabel(value: PatientBookingStatus) {
  return ({ confirmed: "待报到", queued: "排队中", called: "正在叫号", in_progress: "检查中", report_pending: "报告待完成", completed: "已完成", no_show: "未到场", canceled: "已取消" } as const)[value];
}
</script>

<template>
  <AppPage :title="isCompletedView ? '检查记录' : '我的预约'" :description="isCompletedView ? '查看已完成和未到场的历史记录。' : '查看待检查和检查中的预约；项目窗口开始前可以主动删除。'">
    <view v-if="loading" class="state-card">正在加载预约…</view>
    <view v-else-if="errorMessage" class="state-card state-card--error" @tap="loadBookings">{{ errorMessage }}</view>
    <view v-else-if="!bookings.length" class="state-card">{{ isCompletedView ? "暂无检查记录" : "暂无待检查或检查中的预约" }}</view>
    <view v-else class="booking-list">
      <view v-for="booking in bookings" :key="booking.bookingId" class="booking-card" :class="{ 'booking-card--focused': booking.bookingId === focusedBookingId }">
        <view class="booking-card__heading">
          <text class="booking-card__item">{{ booking.itemName }}</text>
          <text class="booking-card__status" :class="`status--${booking.status}`">{{ statusLabel(booking.status) }}</text>
        </view>
        <text class="booking-card__time">{{ booking.serviceDate }} · {{ sessionLabel(booking.session) }} · {{ booking.itemStartTime }}–{{ booking.itemEndTime }}</text>
        <text class="booking-card__room">{{ booking.roomDisplayName }}</text>
        <text class="booking-card__duration">{{ formatEstimatedDuration(booking.estimatedDurationMinutes) }}</text>
        <view v-if="booking.status === 'queued' || booking.status === 'called'" class="queue-panel" :class="{ 'queue-panel--called': booking.status === 'called' }">
          <text class="queue-panel__number">{{ booking.status === 'called' ? `请 ${booking.queueNumber} 号患者前往检查` : `候检号 ${booking.queueNumber}` }}</text>
          <text v-if="booking.status === 'queued'" class="queue-panel__detail">当前叫到 {{ booking.currentCalledQueueNumber || '—' }} 号 · 前方 {{ booking.peopleAhead }} 人</text>
          <text v-else class="queue-panel__detail">{{ calledCountdown(booking) }}，并留意现场广播</text>
        </view>
        <text v-if="checkInHint(booking)" class="check-in-hint">{{ checkInHint(booking) }}</text>
        <view v-if="booking.status === 'confirmed'" class="booking-actions">
          <button class="check-in-button" :disabled="!canCheckIn(booking) || checkingInId === booking.bookingId" @tap="checkIn(booking)">{{ checkingInId === booking.bookingId ? '报到中…' : '检查报到' }}</button>
          <button class="delete-button" :disabled="deletingId === booking.bookingId" @tap="requestDelete(booking)">{{ deletingId === booking.bookingId ? '删除中…' : '删除预约' }}</button>
        </view>
      </view>
    </view>
  </AppPage>
</template>

<style scoped>
button::after{display:none}.state-card,.booking-card{margin-bottom:20rpx;padding:28rpx;color:#718096;font-size:24rpx;background:#fff;border:1rpx solid #e4eaf1;border-radius:18rpx}.state-card{text-align:center}.state-card--error{color:#c34c4c}.booking-card--focused{border-color:#8fc5ea;background:#f8fcff}.booking-card__heading{display:flex;align-items:center;justify-content:space-between}.booking-card__item{color:#263348;font-size:29rpx;font-weight:700}.booking-card__status{padding:6rpx 13rpx;color:#2379da;font-size:19rpx;background:#eaf4ff;border-radius:16rpx}.booking-card__status.status--called{color:#fff;background:#2188c7}.booking-card__status.status--in_progress,.booking-card__status.status--completed,.booking-card__status.status--report_pending{color:#27855f;background:#e9f8f1}.booking-card__status.status--no_show{color:#8a5b35;background:#f8efe6}.booking-card__time,.booking-card__room{display:block;margin-top:15rpx;color:#66758a;font-size:22rpx}.booking-card__room{color:#8793a3}.booking-actions{display:flex;gap:14rpx;margin-top:24rpx}.booking-actions button{flex:1;margin:0;padding:0;font-size:22rpx;line-height:64rpx;border-radius:32rpx}.delete-button{color:#d14e4e;background:#fff4f4;border:1rpx solid #f1cccc}.check-in-button{color:#fff;background:#2188c7}.check-in-button[disabled]{color:#8b98a8;background:#edf1f5}.check-in-hint{display:block;margin-top:18rpx;color:#708096;font-size:21rpx}.queue-panel{margin-top:22rpx;padding:20rpx;background:#edf7fd;border-left:6rpx solid #2188c7;border-radius:12rpx}.queue-panel--called{color:#fff;background:#2188c7}.queue-panel__number,.queue-panel__detail{display:block}.queue-panel__number{font-size:25rpx;font-weight:700}.queue-panel__detail{margin-top:8rpx;font-size:20rpx;opacity:.82}
.booking-card__duration{display:block;margin-top:15rpx;color:#347ff0;font-size:22rpx}
</style>
