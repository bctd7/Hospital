<script setup lang="ts">
import { computed, ref } from "vue";

import { logout, sessionState } from "@/stores/session";

const leaving = ref(false);

const identityLabel = computed(() => {
  const roles = sessionState.principal?.roles ?? [];
  if (roles.includes("super_admin")) {
    return "超级管理员";
  }
  if (roles.includes("department_doctor")) {
    return "医生";
  }
  return "工作人员";
});

async function exitStaffApp() {
  if (leaving.value) {
    return;
  }

  leaving.value = true;
  await logout();
  uni.reLaunch({
    url: "/pages/entry/index",
    fail: () => {
      leaving.value = false;
      uni.showToast({ title: "页面跳转失败，请重试", icon: "none" });
    },
  });
}
</script>

<template>
  <view class="staff-page">
    <view class="staff-page__identity">
      <text class="staff-page__eyebrow">当前身份</text>
      <text class="staff-page__title">{{ identityLabel }}</text>
      <text class="staff-page__hint">已进入工作人员后台，功能暂未开始建设</text>
    </view>
    <button
      class="staff-page__exit"
      :disabled="leaving"
      :loading="leaving"
      @tap="exitStaffApp"
    >
      退出登录
    </button>
  </view>
</template>

<style scoped>
.staff-page {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: calc(var(--status-bar-height) + 48rpx) 48rpx 64rpx;
  background: #ffffff;
}

.staff-page__identity {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.staff-page__eyebrow {
  color: #9299a5;
  font-size: 24rpx;
}

.staff-page__title {
  margin-top: 18rpx;
  color: #1c2430;
  font-size: 48rpx;
  font-weight: 700;
}

.staff-page__hint {
  margin-top: 20rpx;
  color: #a1a7b0;
  font-size: 25rpx;
  text-align: center;
}

.staff-page__exit {
  width: 280rpx;
  margin-top: 96rpx;
  color: #687386;
  font-size: 27rpx;
  line-height: 76rpx;
  background: #f4f6f8;
  border-radius: 38rpx;
}

.staff-page__exit::after {
  display: none;
}
</style>
