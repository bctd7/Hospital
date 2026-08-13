<script setup lang="ts">
import { onLoad, onShow } from "@dcloudio/uni-app";
import { ref } from "vue";

import { staffBookingApi } from "@/api/appointment";
import type { PatientBooking } from "@/types/appointment";

const departmentId = ref("");
const departmentLabel = ref("科室预约");
const bookings = ref<PatientBooking[]>([]);
const loading = ref(false);
const errorMessage = ref("");
const pendingId = ref("");

onLoad((query) => {
  departmentId.value = decode(query?.department_id);
  departmentLabel.value = decode(query?.department_label) || "科室预约";
  uni.setNavigationBarTitle({ title: departmentLabel.value });
});
onShow(() => void loadBookings());

async function loadBookings() {
  if (!departmentId.value || loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    bookings.value = (await staffBookingApi.listBookings(departmentId.value)).items;
  } catch (error) {
    errorMessage.value = messageOf(error, "科室预约加载失败");
  } finally {
    loading.value = false;
  }
}

async function checkIn(value: PatientBooking) {
  if (pendingId.value || value.status !== "confirmed") return;
  pendingId.value = value.bookingId;
  try {
    const updated = await staffBookingApi.checkInBooking(value);
    bookings.value = bookings.value.map((booking) => booking.bookingId === updated.bookingId ? updated : booking);
    uni.showToast({ title: "核销成功", icon: "success" });
  } catch (error) {
    uni.showModal({ title: "核销失败", content: messageOf(error, "请刷新后重试"), showCancel: false });
  } finally {
    pendingId.value = "";
  }
}

function requestDelete(value: PatientBooking) {
  if (pendingId.value) return;
  uni.showModal({
    title: "删除预约",
    content: `确认删除“${value.itemName}”预约？`,
    success: (result) => { if (result.confirm) void deleteBooking(value); },
  });
}

async function deleteBooking(value: PatientBooking) {
  pendingId.value = value.bookingId;
  try {
    await staffBookingApi.deleteBooking(value.bookingId);
    bookings.value = bookings.value.filter((booking) => booking.bookingId !== value.bookingId);
    uni.showToast({ title: "已删除", icon: "success" });
  } catch (error) {
    uni.showModal({ title: "删除失败", content: messageOf(error, "请稍后重试"), showCancel: false });
  } finally {
    pendingId.value = "";
  }
}

function decode(value?: string) { try { return value ? decodeURIComponent(value) : ""; } catch { return ""; } }
function messageOf(error: unknown, fallback: string) { return error instanceof Error && error.message.trim() ? error.message : fallback; }
function sessionLabel(value: string) { return value === "morning" ? "上午" : "下午"; }
</script>

<template>
  <view class="staff-bookings">
    <view class="summary"><view><text class="summary__title">{{ departmentLabel }}</text><text class="summary__minor">患者预约由患者选择房间；工作人员只核销或处理异常预约</text></view><button @tap="loadBookings">刷新</button></view>
    <view v-if="loading" class="state">正在加载…</view>
    <view v-else-if="errorMessage" class="state state--error" @tap="loadBookings">{{ errorMessage }}</view>
    <view v-else-if="!bookings.length" class="state">暂无预约</view>
    <view v-else class="booking-list">
      <view v-for="booking in bookings" :key="booking.bookingId" class="booking-card">
        <view class="booking-card__heading"><text>{{ booking.itemName }}</text><text class="status" :class="{ checked: booking.status === 'checked_in' }">{{ booking.status === 'checked_in' ? '已核销' : '待核销' }}</text></view>
        <text class="booking-card__line">{{ booking.serviceDate }} · {{ sessionLabel(booking.session) }} · {{ booking.itemStartTime }}–{{ booking.itemEndTime }}</text>
        <text class="booking-card__line">{{ booking.roomDisplayName }}</text>
        <view class="actions"><button v-if="booking.status === 'confirmed'" :disabled="pendingId === booking.bookingId" @tap="checkIn(booking)">到院核销</button><button class="danger" :disabled="pendingId === booking.bookingId" @tap="requestDelete(booking)">删除</button></view>
      </view>
    </view>
  </view>
</template>

<style scoped>
button::after{display:none}.staff-bookings{min-height:100vh;padding:22rpx 22rpx calc(28rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#f3f5f8}.summary,.booking-card,.state{padding:24rpx;background:#fff;border:1rpx solid #e1e6ed;border-radius:18rpx}.summary{display:flex;align-items:center;justify-content:space-between}.summary__title,.summary__minor,.booking-card__line{display:block}.summary__title{color:#263348;font-size:27rpx;font-weight:700}.summary__minor{margin-top:7rpx;color:#8995a5;font-size:18rpx}.summary button,.actions button{margin:0;padding:0 22rpx;color:#fff;font-size:20rpx;line-height:58rpx;background:#2188c7;border-radius:29rpx}.state{margin-top:20rpx;color:#8490a0;font-size:22rpx;text-align:center}.state--error{color:#be4e5d}.booking-card{margin-top:18rpx}.booking-card__heading{display:flex;align-items:center;justify-content:space-between;color:#263348;font-size:26rpx;font-weight:700}.status{padding:5rpx 12rpx;color:#277bc0;font-size:18rpx;background:#edf6fb;border-radius:14rpx}.status.checked{color:#287b5e;background:#eaf7f1}.booking-card__line{margin-top:13rpx;color:#758195;font-size:21rpx}.actions{display:flex;justify-content:flex-end;gap:14rpx;margin-top:22rpx}.actions .danger{color:#c84f5d;background:#fff0f1}
</style>
