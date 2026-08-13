<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { ref } from "vue";

import { patientAppointmentApi } from "@/api/appointment";
import AppPage from "@/components/layout/AppPage.vue";
import type { PatientBooking } from "@/types/appointment";

const bookings = ref<PatientBooking[]>([]);
const loading = ref(false);
const errorMessage = ref("");
const deletingId = ref("");

onShow(() => void loadBookings());

async function loadBookings() {
  if (loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    bookings.value = (await patientAppointmentApi.listMyBookings(1, 100)).items;
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
function sessionLabel(value: string) { return value === "morning" ? "上午" : "下午"; }
</script>

<template>
  <AppPage title="我的预约" description="查看本周检查预约；项目窗口开始前可以主动删除。">
    <view v-if="loading" class="state-card">正在加载预约…</view>
    <view v-else-if="errorMessage" class="state-card state-card--error" @tap="loadBookings">{{ errorMessage }}</view>
    <view v-else-if="!bookings.length" class="state-card">暂无预约</view>
    <view v-else class="booking-list">
      <view v-for="booking in bookings" :key="booking.bookingId" class="booking-card">
        <view class="booking-card__heading">
          <text class="booking-card__item">{{ booking.itemName }}</text>
          <text class="booking-card__status" :class="{ checked: booking.status === 'checked_in' }">{{ booking.status === 'checked_in' ? '已核销' : '已预约' }}</text>
        </view>
        <text class="booking-card__time">{{ booking.serviceDate }} · {{ sessionLabel(booking.session) }} · {{ booking.itemStartTime }}–{{ booking.itemEndTime }}</text>
        <text class="booking-card__room">{{ booking.roomDisplayName }}</text>
        <button v-if="booking.status === 'confirmed'" class="delete-button" :disabled="deletingId === booking.bookingId" @tap="requestDelete(booking)">{{ deletingId === booking.bookingId ? '删除中…' : '删除预约' }}</button>
      </view>
    </view>
  </AppPage>
</template>

<style scoped>
button::after{display:none}.state-card,.booking-card{margin-bottom:20rpx;padding:28rpx;color:#718096;font-size:24rpx;background:#fff;border:1rpx solid #e4eaf1;border-radius:18rpx}.state-card{text-align:center}.state-card--error{color:#c34c4c}.booking-card__heading{display:flex;align-items:center;justify-content:space-between}.booking-card__item{color:#263348;font-size:29rpx;font-weight:700}.booking-card__status{padding:6rpx 13rpx;color:#2379da;font-size:19rpx;background:#eaf4ff;border-radius:16rpx}.booking-card__status.checked{color:#27855f;background:#e9f8f1}.booking-card__time,.booking-card__room{display:block;margin-top:15rpx;color:#66758a;font-size:22rpx}.booking-card__room{color:#8793a3}.delete-button{margin:24rpx 0 0;padding:0;color:#d14e4e;font-size:22rpx;line-height:64rpx;background:#fff4f4;border:1rpx solid #f1cccc;border-radius:32rpx}
</style>
