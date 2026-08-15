<script setup lang="ts">
import { onLoad, onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { patientAppointmentApi } from "@/api/appointment";
import AppPage from "@/components/layout/AppPage.vue";
import type { AppointmentListView, PatientBooking, PatientBookingStatus } from "@/types/appointment";
import { formatEstimatedDuration } from "@/utils/appointmentManagement";

const bookings = ref<PatientBooking[]>([]);
const loading = ref(false);
const errorMessage = ref("");
const deletingId = ref("");
const focusedBookingId = ref("");
const view = ref<AppointmentListView>("active");
const isCompletedView = computed(() => view.value === "completed");

onLoad((query) => {
  view.value = query?.view === "completed" ? "completed" : "active";
  focusedBookingId.value = decode(query?.booking_id);
  uni.setNavigationBarTitle({ title: isCompletedView.value ? "检查记录" : "我的预约" });
});

onShow(() => void loadBookings());

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

function messageOf(error: unknown, fallback: string) {
  return error instanceof Error && error.message.trim() ? error.message : fallback;
}
function decode(value?: string) { try { return value ? decodeURIComponent(value) : ""; } catch { return ""; } }
function sessionLabel(value: string) { return value === "morning" ? "上午" : "下午"; }
function statusLabel(value: PatientBookingStatus) {
  return ({ confirmed: "待检查", in_progress: "检查中", completed: "已完成", no_show: "未到场", canceled: "已取消" } as const)[value];
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
        <button v-if="booking.status === 'confirmed'" class="delete-button" :disabled="deletingId === booking.bookingId" @tap="requestDelete(booking)">{{ deletingId === booking.bookingId ? '删除中…' : '删除预约' }}</button>
      </view>
    </view>
  </AppPage>
</template>

<style scoped>
button::after{display:none}.state-card,.booking-card{margin-bottom:20rpx;padding:28rpx;color:#718096;font-size:24rpx;background:#fff;border:1rpx solid #e4eaf1;border-radius:18rpx}.state-card{text-align:center}.state-card--error{color:#c34c4c}.booking-card--focused{border-color:#8fc5ea;background:#f8fcff}.booking-card__heading{display:flex;align-items:center;justify-content:space-between}.booking-card__item{color:#263348;font-size:29rpx;font-weight:700}.booking-card__status{padding:6rpx 13rpx;color:#2379da;font-size:19rpx;background:#eaf4ff;border-radius:16rpx}.booking-card__status.status--in_progress,.booking-card__status.status--completed{color:#27855f;background:#e9f8f1}.booking-card__status.status--no_show{color:#8a5b35;background:#f8efe6}.booking-card__time,.booking-card__room{display:block;margin-top:15rpx;color:#66758a;font-size:22rpx}.booking-card__room{color:#8793a3}.delete-button{margin:24rpx 0 0;padding:0;color:#d14e4e;font-size:22rpx;line-height:64rpx;background:#fff4f4;border:1rpx solid #f1cccc;border-radius:32rpx}
.booking-card__duration{display:block;margin-top:15rpx;color:#347ff0;font-size:22rpx}
</style>
