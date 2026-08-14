<script setup lang="ts">
import { onLoad, onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { staffBookingApi } from "@/api/appointment";
import {
  loadDepartmentOptions,
  type DepartmentOption,
} from "@/services/organization";
import type { AppointmentListView, PatientBooking, PatientBookingStatus } from "@/types/appointment";
import { examinationWindowRelation } from "@/utils/appointmentManagement";

const departmentId = ref("");
const departmentLabel = ref("科室预约");
const patientKeyword = ref("");
const status = ref<PatientBookingStatus | "">("");
const bookings = ref<PatientBooking[]>([]);
const loading = ref(false);
const errorMessage = ref("");
const pendingId = ref("");
const departmentLocked = ref(false);
const departmentOptions = ref<DepartmentOption[]>([]);
const loadingDepartments = ref(false);
const view = ref<AppointmentListView>("active");
const isCompletedView = computed(() => view.value === "completed");
const canSwitchDepartment = computed(() => !departmentLocked.value && departmentOptions.value.length > 1);
const statusOptions: Array<{ label: string; value: PatientBookingStatus | "" }> = [
  { label: "全部状态", value: "" },
  { label: "待检查", value: "confirmed" },
  { label: "检查中", value: "in_progress" },
];

onLoad((query) => {
  view.value = query?.view === "completed" ? "completed" : "active";
  departmentId.value = decode(query?.department_id);
  departmentLocked.value = Boolean(departmentId.value);
  departmentLabel.value = decode(query?.department_label) || "科室预约";
  uni.setNavigationBarTitle({ title: departmentLabel.value });
  if (!departmentLocked.value) void prepareDepartmentSelection();
});
onShow(() => void loadBookings());

async function prepareDepartmentSelection() {
  if (loadingDepartments.value) return;
  loadingDepartments.value = true;
  errorMessage.value = "";
  try {
    departmentOptions.value = await loadDepartmentOptions();
    const first = departmentOptions.value[0];
    if (!first) {
      errorMessage.value = "暂无可查看的科室";
      return;
    }
    selectDepartment(first);
    await loadBookings();
  } catch (error) {
    errorMessage.value = messageOf(error, "科室列表加载失败");
  } finally {
    loadingDepartments.value = false;
  }
}

function selectDepartment(option: DepartmentOption) {
  departmentId.value = option.department.departmentId;
  departmentLabel.value = option.label;
}

function switchDepartment() {
  if (!canSwitchDepartment.value || loading.value) return;
  uni.showActionSheet({
    title: "选择要查看的科室",
    itemList: departmentOptions.value.map((option) => option.label),
    success: ({ tapIndex }) => {
      const selected = departmentOptions.value[tapIndex];
      if (!selected || selected.department.departmentId === departmentId.value) return;
      selectDepartment(selected);
      patientKeyword.value = "";
      status.value = "";
      bookings.value = [];
      void loadBookings();
    },
  });
}

async function loadBookings() {
  if (!departmentId.value || loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    bookings.value = (await staffBookingApi.listBookings(departmentId.value, {
      patientKeyword: patientKeyword.value.trim() || undefined,
      status: status.value || undefined,
      view: view.value,
    })).items;
  } catch (error) {
    errorMessage.value = messageOf(error, "科室预约加载失败");
  } finally {
    loading.value = false;
  }
}

function selectStatus(event: { detail: { value: string | number } }) {
  status.value = statusOptions[Number(event.detail.value)]?.value ?? "";
  void loadBookings();
}

function openBooking(value: PatientBooking) {
  uni.navigateTo({ url: `/pages/admin/appointment/booking-detail?booking_id=${encodeURIComponent(value.bookingId)}` });
}

async function startExamination(value: PatientBooking) {
  if (pendingId.value || value.status !== "confirmed") return;
  if (examinationWindowRelation(value.serviceDate, value.itemStartTime, value.itemEndTime) !== "open") {
    uni.showModal({
      title: "暂不能开始检查",
      content: examinationWindowDescription(value),
      showCancel: false,
    });
    return;
  }
  pendingId.value = value.bookingId;
  try {
    const updated = await staffBookingApi.startExamination(value);
    bookings.value = bookings.value.map((booking) => booking.bookingId === updated.bookingId ? updated : booking);
    uni.showToast({ title: "已开始检查", icon: "success" });
  } catch (error) {
    uni.showModal({ title: "开始检查失败", content: messageOf(error, "请刷新后重试"), showCancel: false });
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
function statusLabel(value: PatientBookingStatus) {
  return ({ confirmed: "待检查", in_progress: "检查中", completed: "已完成", no_show: "未到场" } as const)[value];
}
function examinationActionLabel(value: PatientBooking) {
  const relation = examinationWindowRelation(value.serviceDate, value.itemStartTime, value.itemEndTime);
  if (relation === "open") return "开始检查";
  if (relation === "after") return "已超过检查时间";
  if (relation === "before") return `${clockLabel(value.itemStartTime)} 后可开始`;
  return "暂不可开始";
}
function examinationWindowDescription(value: PatientBooking) { return `只能在 ${value.serviceDate} ${clockLabel(value.itemStartTime)}–${clockLabel(value.itemEndTime)} 内开始检查。`; }
function clockLabel(value: string) { return value.slice(0, 5); }
function roomAddress(value: PatientBooking) { return `${value.campusName ? `${value.campusName} · ` : ""}${value.building} · ${value.floorNumber}层 · ${value.roomNumber}室`; }
</script>

<template>
  <view class="staff-bookings">
    <view class="summary">
      <view><text class="summary__title">{{ departmentLabel }}</text><text class="summary__minor">{{ isCompletedView ? "查看并搜索本科室已完成和未到场记录" : "按患者姓名或手机号后四位查找本科室待检查和检查中预约" }}</text></view>
      <view class="summary__actions">
        <button v-if="canSwitchDepartment" @tap="switchDepartment">切换科室</button>
        <button @tap="loadBookings">刷新</button>
      </view>
    </view>
    <view class="filters">
      <input v-model="patientKeyword" confirm-type="search" placeholder="患者姓名 / 手机号后四位" @confirm="loadBookings" />
      <picker v-if="!isCompletedView" :range="statusOptions" range-key="label" @change="selectStatus"><view class="filter-status">{{ statusOptions.find((item) => item.value === status)?.label }} ›</view></picker>
      <button @tap="loadBookings">搜索</button>
    </view>
    <view v-if="loading" class="state">正在加载…</view>
    <view v-else-if="errorMessage" class="state state--error" @tap="loadBookings">{{ errorMessage }}</view>
    <view v-else-if="!bookings.length" class="state">{{ isCompletedView ? "暂无符合条件的诊后记录" : "暂无符合条件的待检查或检查中预约" }}</view>
    <view v-else class="booking-list">
      <view v-for="booking in bookings" :key="booking.bookingId" class="booking-card" @tap="openBooking(booking)">
        <view class="booking-card__heading">
          <view><text class="patient">{{ booking.patientDisplayName }}</text><text class="phone">{{ booking.patientPhoneMasked }}</text></view>
          <text class="status" :class="`status--${booking.status}`">{{ statusLabel(booking.status) }}</text>
        </view>
        <text class="booking-card__project">{{ booking.itemName }}</text>
        <text class="booking-card__line">{{ booking.serviceDate }} · {{ sessionLabel(booking.session) }} · {{ booking.itemStartTime }}–{{ booking.itemEndTime }}</text>
        <text class="booking-card__line">{{ roomAddress(booking) }}</text>
        <view class="actions" @tap.stop>
          <button v-if="booking.status === 'confirmed'" :disabled="pendingId === booking.bookingId" @tap="startExamination(booking)">{{ examinationActionLabel(booking) }}</button>
          <button v-if="booking.status === 'confirmed'" class="danger" :disabled="pendingId === booking.bookingId" @tap="requestDelete(booking)">删除</button>
          <button class="detail" @tap="openBooking(booking)">查看详情</button>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
button::after{display:none}.staff-bookings{min-height:100vh;padding:22rpx 22rpx calc(28rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#f3f5f8}.summary,.filters,.booking-card,.state{padding:24rpx;background:#fff;border:1rpx solid #e1e6ed;border-radius:18rpx}.summary{display:flex;align-items:center;justify-content:space-between}.summary__title,.summary__minor,.booking-card__project,.booking-card__line{display:block}.summary__title{color:#263348;font-size:27rpx;font-weight:700}.summary__minor{margin-top:7rpx;color:#8995a5;font-size:18rpx}.summary button,.filters button,.actions button{width:auto;margin:0;padding:0 22rpx;color:#fff;font-size:20rpx;line-height:58rpx;background:#2188c7;border-radius:29rpx}.filters{display:grid;grid-template-columns:1fr auto auto;gap:12rpx;margin-top:16rpx;padding:14rpx}.filters input,.filter-status{height:62rpx;padding:0 18rpx;color:#344157;font-size:21rpx;line-height:62rpx;background:#f3f6f9;border-radius:14rpx}.filter-status{min-width:120rpx}.state{margin-top:20rpx;color:#8490a0;font-size:22rpx;text-align:center}.state--error{color:#be4e5d}.booking-card{margin-top:18rpx}.booking-card__heading{display:flex;align-items:center;justify-content:space-between}.patient{color:#263348;font-size:27rpx;font-weight:700}.phone{margin-left:14rpx;color:#7f8b9c;font-size:20rpx}.status{padding:5rpx 12rpx;color:#277bc0;font-size:18rpx;background:#edf6fb;border-radius:14rpx}.status--in_progress{color:#147cb1;background:#e6f5fc}.status--completed{color:#287b5e;background:#eaf7f1}.status--no_show{color:#8a5b35;background:#f8efe6}.booking-card__project{margin-top:18rpx;color:#344157;font-size:24rpx;font-weight:650}.booking-card__line{margin-top:10rpx;color:#758195;font-size:21rpx}.actions{display:flex;justify-content:flex-end;gap:12rpx;margin-top:22rpx}.actions .danger{color:#c84f5d;background:#fff0f1}.actions .detail{color:#247bb4;background:#edf6fb}
.summary__actions{display:flex;gap:10rpx;margin-left:14rpx;flex-shrink:0}
</style>
