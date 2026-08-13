<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import {
  patientAppointmentMockAdapter,
  type PatientMockDepartment,
} from "@/mocks/appointmentBooking";

const departments = ref<PatientMockDepartment[]>([]);
const selectedDepartmentId = ref("");
const selectedItemId = ref("");
const selectedRoomId = ref("");
const selectedWindowId = ref("");

const selectedDepartment = computed(() => departments.value.find((value) => value.id === selectedDepartmentId.value));
const items = computed(() => selectedDepartment.value?.items ?? []);
const selectedItem = computed(() => items.value.find((value) => value.id === selectedItemId.value));
const rooms = computed(() => selectedItem.value?.rooms ?? []);
const selectedRoom = computed(() => rooms.value.find((value) => value.id === selectedRoomId.value));
const windows = computed(() => selectedRoom.value?.windows ?? []);
const selectedWindow = computed(() => windows.value.find((value) => value.id === selectedWindowId.value));
const canConfirm = computed(() => Boolean(selectedDepartment.value && selectedItem.value && selectedRoom.value && selectedWindow.value));

onMounted(async () => {
  departments.value = await patientAppointmentMockAdapter.listDepartments();
});

function chooseDepartment(id: string) {
  if (id === selectedDepartmentId.value) return;
  selectedDepartmentId.value = id;
  selectedItemId.value = "";
  selectedRoomId.value = "";
  selectedWindowId.value = "";
}

function chooseItem(id: string) {
  if (id === selectedItemId.value) return;
  selectedItemId.value = id;
  selectedRoomId.value = "";
  selectedWindowId.value = "";
}

function chooseRoom(id: string) {
  if (id === selectedRoomId.value) return;
  selectedRoomId.value = id;
  selectedWindowId.value = "";
}

function confirmMockAppointment() {
  if (!canConfirm.value) return;
  uni.showModal({
    title: "Mock 预约预览",
    content: `${selectedDepartment.value?.campus} / ${selectedDepartment.value?.name}\n${selectedItem.value?.name} · ${selectedRoom.value?.name}\n${selectedWindow.value?.weekday} ${selectedWindow.value?.session} ${selectedWindow.value?.time}\n\n当前只验证患者选择流程，不会创建真实预约。`,
    showCancel: false,
    confirmText: "我知道了",
  });
}
</script>

<template>
  <view class="booking-page">
    <view class="booking-hero">
      <text class="booking-hero__eyebrow">检查项目 · MOCK 预览</text>
      <text class="booking-hero__title">先了解项目，再选择具体检查房间</text>
      <text class="booking-hero__description">目前不会占用容量，也不会向后端创建预约。</text>
    </view>

    <view class="step-card">
      <view class="step-heading"><text class="step-number">1</text><text class="step-title">选择院区和科室</text></view>
      <view class="department-grid">
        <button v-for="value in departments" :key="value.id" class="department-card" :class="{ selected: selectedDepartmentId === value.id }" @tap="chooseDepartment(value.id)">
          <text class="department-card__campus">{{ value.campus }}</text><text class="department-card__name">{{ value.name }}</text><text class="department-card__count">{{ value.items.length }} 个检查项目</text>
        </button>
      </view>
    </view>

    <view v-if="selectedDepartment" class="step-card">
      <view class="step-heading"><text class="step-number">2</text><text class="step-title">选择检查项目</text></view>
      <button v-for="value in items" :key="value.id" class="choice-card" :class="{ selected: selectedItemId === value.id }" @tap="chooseItem(value.id)">
        <view><text class="choice-card__title">{{ value.name }}</text><text class="choice-card__description">{{ value.description }}</text></view><text class="choice-card__check">{{ selectedItemId === value.id ? '✓' : '›' }}</text>
      </button>
    </view>

    <view v-if="selectedItem" class="step-card">
      <view class="preparation"><text>检查准备</text><text>{{ selectedItem.preparation }}</text></view>
      <view class="step-heading"><text class="step-number">3</text><text class="step-title">选择检查房间</text></view>
      <button v-for="value in rooms" :key="value.id" class="choice-card" :class="{ selected: selectedRoomId === value.id }" @tap="chooseRoom(value.id)">
        <view><text class="choice-card__title">{{ value.name }}</text><text class="choice-card__description">由患者主动选择，不绑定医生</text></view><text class="choice-card__check">{{ selectedRoomId === value.id ? '✓' : '›' }}</text>
      </button>
    </view>

    <view v-if="selectedRoom" class="step-card">
      <view class="step-heading"><text class="step-number">4</text><text class="step-title">选择本周时间</text></view>
      <button v-for="value in windows" :key="value.id" class="window-card" :class="{ selected: selectedWindowId === value.id }" @tap="selectedWindowId = value.id">
        <view><text class="window-card__date">{{ value.weekday }} · {{ value.date }} · {{ value.session }}</text><text class="window-card__time">{{ value.time }}，{{ value.cutoff }} 停止新增</text></view><text class="choice-card__check">{{ selectedWindowId === value.id ? '✓' : '›' }}</text>
      </button>
    </view>

    <view class="booking-footer">
      <text class="booking-footer__hint">演示数据不会写入后端，也不代表真实剩余容量</text>
      <button class="confirm-button" :disabled="!canConfirm" @tap="confirmMockAppointment">预览选择</button>
    </view>
  </view>
</template>

<style scoped>
button::after{display:none}.booking-page{min-height:100vh;padding:24rpx 24rpx 180rpx;box-sizing:border-box;background:#f3f6fa}.booking-hero{padding:34rpx 32rpx;color:#fff;background:linear-gradient(135deg,#1487d7,#19b7bd);border-radius:28rpx}.booking-hero__eyebrow,.booking-hero__title,.booking-hero__description{display:block}.booking-hero__eyebrow{font-size:20rpx;opacity:.78;letter-spacing:2rpx}.booking-hero__title{margin-top:14rpx;font-size:34rpx;font-weight:730;line-height:1.4}.booking-hero__description{margin-top:10rpx;font-size:22rpx;opacity:.86}.step-card{margin-top:22rpx;padding:26rpx;background:#fff;border:1rpx solid #e6ebf1;border-radius:24rpx}.step-heading{display:flex;align-items:center;gap:14rpx;margin-bottom:20rpx}.step-number{display:flex;align-items:center;justify-content:center;width:42rpx;height:42rpx;color:#fff;font-size:21rpx;font-weight:700;background:#178dce;border-radius:14rpx}.step-title{color:#2a3548;font-size:27rpx;font-weight:700}.department-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14rpx}.department-card{display:flex;flex-direction:column;align-items:flex-start;margin:0;padding:22rpx;color:inherit;text-align:left;background:#f8fafc;border:1rpx solid #e7ecf2;border-radius:18rpx}.department-card__campus{color:#718096;font-size:20rpx}.department-card__name{margin-top:7rpx;color:#2e3a4e;font-size:25rpx;font-weight:680}.department-card__count{margin-top:12rpx;color:#1590ce;font-size:19rpx}.choice-card,.window-card{display:flex;align-items:center;justify-content:space-between;width:100%;margin:12rpx 0 0;padding:20rpx;color:inherit;text-align:left;background:#f8fafc;border:1rpx solid #e8edf3;border-radius:18rpx}.selected{background:#edf8ff;border-color:#48a8de}.choice-card__title,.choice-card__description,.window-card__date,.window-card__time{display:block}.choice-card__title,.window-card__date{color:#334055;font-size:24rpx;font-weight:650}.choice-card__description,.window-card__time{max-width:570rpx;margin-top:7rpx;color:#8c97a8;font-size:20rpx;line-height:1.5}.choice-card__check{color:#168dce;font-size:28rpx}.preparation{margin-bottom:22rpx;padding:18rpx;color:#7b633c;font-size:20rpx;line-height:1.6;background:#fff8e9;border-radius:16rpx}.preparation text{display:block}.preparation text:first-child{font-weight:700}.booking-footer{position:fixed;left:0;right:0;bottom:0;display:flex;align-items:center;gap:20rpx;padding:20rpx 24rpx calc(20rpx + env(safe-area-inset-bottom));background:rgba(255,255,255,.96);border-top:1rpx solid #e5eaf0}.booking-footer__hint{flex:1;color:#8c97a7;font-size:19rpx;line-height:1.4}.confirm-button{flex:0 0 auto;margin:0;padding:0 34rpx;color:#fff;font-size:24rpx;line-height:72rpx;background:#168dce;border-radius:36rpx}.confirm-button[disabled]{color:#fff;background:#bcc8d3}
</style>
