<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { appointmentMessageApi } from "@/api/appointment";
import AppPage from "@/components/layout/AppPage.vue";
import { loadDepartmentOptions, type DepartmentOption } from "@/services/organization";
import { sessionState } from "@/stores/session";
import type { AppointmentMessage } from "@/types/appointment";
import { hasRole } from "@/utils/appointmentManagement";
import {
  appointmentMessageDetail,
  appointmentMessageSchedule,
  appointmentMessageTarget,
  appointmentMessageTime,
  appointmentMessageTitle,
} from "@/utils/appointmentMessages";
import { currentStaffDepartmentId, rememberStaffDepartmentId } from "@/utils/staffDepartmentContext";

const messages = ref<AppointmentMessage[]>([]);
const departments = ref<DepartmentOption[]>([]);
const departmentId = ref("");
const loading = ref(false);
const errorMessage = ref("");
const unreadCount = ref(0);
const departmentUnreadCounts = ref<Record<string, number>>({});

const isStaff = computed(() => sessionState.appVariant === "staff");
const isDoctor = computed(() => hasRole(sessionState.principal, "department_doctor"));
const canSwitchDepartment = computed(() => isStaff.value && !isDoctor.value && departments.value.length > 0);
const selectedDepartment = computed(() => departments.value.find((value) => value.department.departmentId === departmentId.value));
const description = computed(() => isStaff.value
  ? "查看当前科室的新预约、取消与检查报告待办提醒。"
  : "查看预约、到院和检查报告提醒。",
);

onShow(() => void initialize());

async function initialize() {
  if (loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    if (isStaff.value) {
      departments.value = await loadDepartmentOptions();
      if (isDoctor.value) {
        departmentId.value = sessionState.principal?.department_id?.trim() ?? "";
      } else {
        const remembered = currentStaffDepartmentId();
        departmentId.value = departments.value.some((value) => value.department.departmentId === remembered) ? remembered : "";
      }
      if (!departmentId.value) {
        messages.value = [];
        const summary = await appointmentMessageApi.listDepartment("", 1, 1);
        unreadCount.value = summary.unreadCount;
        departmentUnreadCounts.value = Object.fromEntries(
          summary.departmentUnreadCounts.map((current) => [current.departmentId, current.unreadCount]),
        );
        updateBadge(summary.unreadCount);
        return;
      }
    }
    await loadMessages();
  } catch (error) {
    errorMessage.value = messageOf(error, "消息加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function loadMessages() {
  const value = isStaff.value
    ? await appointmentMessageApi.listDepartment(departmentId.value, 1, 100)
    : await appointmentMessageApi.listMine(1, 100);
  messages.value = value.items;
  unreadCount.value = value.unreadCount;
  departmentUnreadCounts.value = Object.fromEntries(
    value.departmentUnreadCounts.map((current) => [current.departmentId, current.unreadCount]),
  );
  updateBadge(value.unreadCount);
}

function chooseDepartment() {
  if (!canSwitchDepartment.value || loading.value) return;
  uni.showActionSheet({
    title: "选择消息所属科室",
    itemList: departments.value.map((value) => {
      const count = departmentUnreadCounts.value[value.department.departmentId] ?? 0;
      return count > 0 ? `${value.label}（${count}）` : value.label;
    }),
    success: ({ tapIndex }) => {
      const selected = departments.value[tapIndex];
      if (!selected) return;
      departmentId.value = selected.department.departmentId;
      rememberStaffDepartmentId(departmentId.value);
      messages.value = [];
      void reload();
    },
  });
}

async function reload() {
  if (loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    await loadMessages();
  } catch (error) {
    errorMessage.value = messageOf(error, "消息加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function openMessage(value: AppointmentMessage) {
  if (!value.readAt) {
    try {
      const updated = isStaff.value
        ? await appointmentMessageApi.markDepartmentRead(departmentId.value, value.messageKey)
        : await appointmentMessageApi.markMineRead(value.messageKey);
      value.readAt = updated.readAt;
      unreadCount.value = Math.max(0, unreadCount.value - 1);
      if (isStaff.value) {
        departmentUnreadCounts.value[departmentId.value] = Math.max(
          0,
          (departmentUnreadCounts.value[departmentId.value] ?? 0) - 1,
        );
      }
      updateBadge(unreadCount.value);
    } catch (error) {
      uni.showToast({ title: messageOf(error, "标记已读失败"), icon: "none" });
      return;
    }
  }
  uni.navigateTo({ url: appointmentMessageTarget(value, isStaff.value) });
}

function titleOf(value: AppointmentMessage) {
  return appointmentMessageTitle(value.messageType, isStaff.value);
}

function detailOf(value: AppointmentMessage) {
  return appointmentMessageDetail(value, isStaff.value);
}

function appointmentOf(value: AppointmentMessage) {
  return appointmentMessageSchedule(value);
}

function formatTime(value: string) {
  return appointmentMessageTime(value);
}

function updateBadge(count: number) {
  if (count <= 0) {
    clearBadge();
    return;
  }
  uni.setTabBarBadge({ index: 2, text: count > 99 ? "99+" : String(count) });
}

function clearBadge() {
  uni.removeTabBarBadge({ index: 2, fail: () => undefined });
}

function messageOf(error: unknown, fallback: string) {
  return error instanceof Error && error.message.trim() ? error.message : fallback;
}
</script>

<template>
  <AppPage title="消息" :description="description">
    <view v-if="isStaff" class="scope-card" @tap="chooseDepartment">
      <view>
        <text class="scope-card__label">当前科室</text>
        <text class="scope-card__value">{{ selectedDepartment?.label || (isDoctor ? "本人所属科室" : "请选择科室") }}</text>
      </view>
      <text v-if="canSwitchDepartment" class="scope-card__action">切换</text>
    </view>

    <view v-if="loading" class="state-card">正在加载消息…</view>
    <view v-else-if="errorMessage" class="state-card state-card--error" @tap="reload">{{ errorMessage }}</view>
    <view v-else-if="isStaff && !departmentId" class="state-card state-card--select" @tap="chooseDepartment">请选择科室后查看消息</view>
    <view v-else-if="!messages.length" class="state-card">暂无消息</view>
    <view v-else class="message-list">
      <view v-for="message in messages" :key="message.messageKey" class="message-card" :class="{ 'message-card--unread': !message.readAt }" @tap="openMessage(message)">
        <view class="message-card__topline">
          <view class="message-card__title-row">
            <view v-if="!message.readAt" class="unread-dot" />
            <text class="message-card__title">{{ titleOf(message) }}</text>
          </view>
          <text class="message-card__time">{{ formatTime(message.occurredAt) }}</text>
        </view>
        <text class="message-card__detail">{{ detailOf(message) }}</text>
        <text class="message-card__appointment">{{ appointmentOf(message) }}</text>
      </view>
    </view>
  </AppPage>
</template>

<style scoped>
.scope-card,.state-card,.message-card{background:#fff;border:1rpx solid #e2e9f1;border-radius:20rpx}.scope-card{display:flex;align-items:center;justify-content:space-between;margin-bottom:24rpx;padding:24rpx 28rpx}.scope-card__label,.scope-card__value{display:block}.scope-card__label{margin-bottom:8rpx;color:#8b96a8;font-size:21rpx}.scope-card__value{color:#25334a;font-size:28rpx;font-weight:650}.scope-card__action{padding:10rpx 18rpx;color:#1677c8;font-size:22rpx;background:#edf7ff;border-radius:22rpx}.state-card{padding:44rpx 28rpx;color:#78869a;font-size:24rpx;text-align:center}.state-card--error{color:#c34c4c}.state-card--select{color:#1677c8}.message-list{display:flex;flex-direction:column;gap:18rpx}.message-card{padding:28rpx}.message-card--unread{border-color:#beddf5;background:#fafdff}.message-card__topline,.message-card__title-row{display:flex;align-items:center}.message-card__topline{justify-content:space-between;gap:20rpx}.message-card__title-row{min-width:0;gap:13rpx}.unread-dot{width:13rpx;height:13rpx;flex:none;background:#168bd1;border-radius:50%}.message-card__title{color:#24334a;font-size:28rpx;font-weight:650}.message-card__time{flex:none;color:#9aa4b3;font-size:19rpx}.message-card__detail,.message-card__appointment{display:block;margin-top:16rpx;color:#506077;font-size:23rpx;line-height:1.55}.message-card__appointment{margin-top:9rpx;color:#8290a3;font-size:21rpx}
</style>
