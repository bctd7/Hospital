<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { loadDoctors } from "@/services/organization";
import type { DoctorSummary } from "@/types/staffManagement";

const accountId = ref("");
const departmentId = ref("");
const departmentName = ref("");
const campusName = ref("");
const hospitalName = ref("");
const doctor = ref<DoctorSummary>();
const loading = ref(true);
const error = ref("");

const avatarText = computed(() => doctor.value?.displayName.trim().slice(0, 1) || "医");
const locationText = computed(() =>
  [hospitalName.value, campusName.value, departmentName.value].filter(Boolean).join(" · "),
);

onLoad((query) => {
  accountId.value = readQuery(query?.account_id);
  departmentId.value = readQuery(query?.department_id);
  departmentName.value = readQuery(query?.department_name);
  campusName.value = readQuery(query?.campus_name);
  hospitalName.value = readQuery(query?.hospital_name);

  if (!accountId.value || !departmentId.value) {
    loading.value = false;
    error.value = "医生信息不完整，请返回后重新选择";
    return;
  }
  void loadDetail(false);
});

async function loadDetail(force: boolean) {
  loading.value = true;
  error.value = "";
  try {
    const doctors = await loadDoctors(departmentId.value, force);
    doctor.value = doctors.find((item) => item.accountId === accountId.value);
    if (!doctor.value) {
      error.value = "未找到该医生，医生信息可能已更新";
    }
  } catch (caught) {
    error.value = caught instanceof Error && caught.message
      ? caught.message
      : "医生详情加载失败，请重试";
  } finally {
    loading.value = false;
  }
}

function readQuery(value: unknown): string {
  if (typeof value !== "string") return "";
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}
</script>

<template>
  <view class="doctor-detail-page">
    <view v-if="loading" class="page-state">医生信息加载中...</view>
    <view v-else-if="error" class="page-state page-state--error">
      <text>{{ error }}</text>
      <button v-if="accountId && departmentId" class="retry-button" @tap="loadDetail(true)">
        重新加载
      </button>
    </view>

    <template v-else-if="doctor">
      <view class="profile-card">
        <image
          v-if="doctor.avatarUrl"
          class="profile-avatar"
          :src="doctor.avatarUrl"
          mode="aspectFill"
        />
        <view v-else class="profile-avatar profile-avatar--fallback">{{ avatarText }}</view>
        <view class="profile-main">
          <text class="profile-name">{{ doctor.displayName }}</text>
          <text class="profile-role">医生</text>
        </view>
      </view>

      <view v-if="locationText" class="location-card">
        <text class="section-label">所在科室</text>
        <text class="location-text">{{ locationText }}</text>
      </view>

      <view class="introduction-card">
        <text class="section-title">医生介绍</text>
        <text class="introduction-text">
          {{ doctor.description || "该医生暂未填写详细介绍。" }}
        </text>
      </view>
    </template>
  </view>
</template>

<style scoped>
.doctor-detail-page {
  min-height: 100vh;
  padding: 28rpx 24rpx 48rpx;
  box-sizing: border-box;
  background: #f3f7fd;
}

.profile-card,
.location-card,
.introduction-card,
.page-state {
  background: #ffffff;
  border: 1rpx solid #e8edf4;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(43, 62, 89, 0.06);
}

.profile-card {
  display: flex;
  align-items: center;
  padding: 32rpx;
}

.profile-avatar {
  width: 112rpx;
  height: 112rpx;
  flex: 0 0 auto;
  border-radius: 50%;
}

.profile-avatar--fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 42rpx;
  font-weight: 700;
  background: linear-gradient(145deg, #55bce9, #557ee8);
}

.profile-main {
  min-width: 0;
  margin-left: 26rpx;
}

.profile-name,
.profile-role,
.section-label,
.location-text,
.section-title,
.introduction-text {
  display: block;
}

.profile-name {
  overflow: hidden;
  color: #253148;
  font-size: 36rpx;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-role {
  margin-top: 10rpx;
  color: #1688d3;
  font-size: 23rpx;
}

.location-card,
.introduction-card {
  margin-top: 20rpx;
  padding: 28rpx 30rpx;
}

.section-label {
  color: #98a2b2;
  font-size: 22rpx;
}

.location-text {
  margin-top: 12rpx;
  color: #354055;
  font-size: 27rpx;
  line-height: 1.6;
}

.section-title {
  color: #253148;
  font-size: 29rpx;
  font-weight: 700;
}

.introduction-text {
  margin-top: 20rpx;
  color: #69758a;
  font-size: 25rpx;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
}

.page-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24rpx;
  min-height: 360rpx;
  padding: 40rpx;
  box-sizing: border-box;
  color: #8d98a9;
  font-size: 25rpx;
  text-align: center;
}

.page-state--error {
  color: #c44f61;
}

.retry-button {
  margin: 0;
  padding: 0 30rpx;
  color: #1683d0;
  font-size: 23rpx;
  line-height: 62rpx;
  background: #edf7ff;
  border-radius: 31rpx;
}

.retry-button::after {
  display: none;
}
</style>
