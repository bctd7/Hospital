<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import ActionMenu from "@/components/menu/ActionMenu.vue";
import ProfileHero from "@/components/profile/ProfileHero.vue";
import {
  availableAppModes,
  sessionState,
  setActiveMode,
} from "@/stores/session";
import type { MenuEntry } from "@/types/menu";
import { applyAppModeNavigation } from "@/utils/appMode";
import { getDisplayProfile } from "@/utils/displayProfile";

const avatarUrl = ref("");
const nickname = ref("微信用户");
const navigationPending = ref(false);

const patientMenuItems: MenuEntry[] = [
  {
    id: "patients",
    title: "就诊人管理",
    symbol: "人",
    tone: "blue",
    route: "/pages/profile/patients/index",
  },
  {
    id: "appointments",
    title: "我的预约",
    symbol: "约",
    tone: "blue",
    route: "/pages/profile/appointments/index",
  },
  {
    id: "favorites",
    title: "我的关注",
    symbol: "关",
    tone: "orange",
    route: "/pages/profile/favorites/index",
  },
  {
    id: "settings",
    title: "设置",
    description: "清除缓存、联系客服、用户隐私协议、账号注销",
    symbol: "设",
    tone: "gray",
    route: "/pages/profile/settings/index",
  },
  {
    id: "message-management",
    title: "消息管理",
    symbol: "讯",
    tone: "cyan",
    route: "/pages/profile/message-management/index",
  },
];

const doctorMenuItems: MenuEntry[] = [
  {
    id: "doctor-appointments",
    title: "预约管理",
    description: "查看和处理本科室预约",
    symbol: "约",
    tone: "blue",
    route: "/pages/registration/index",
  },
  {
    id: "doctor-messages",
    title: "消息",
    description: "查看工作提醒和系统通知",
    symbol: "讯",
    tone: "cyan",
    route: "/pages/messages/index",
  },
  {
    id: "doctor-settings",
    title: "设置",
    symbol: "设",
    tone: "gray",
    route: "/pages/profile/settings/index",
  },
];

const canSwitchMode = computed(() => availableAppModes().includes("doctor"));
const isDoctorMode = computed(() => sessionState.activeMode === "doctor");
const menuItems = computed(() =>
  isDoctorMode.value ? doctorMenuItems : patientMenuItems,
);

onShow(() => {
  navigationPending.value = false;
  const profile = getDisplayProfile();
  avatarUrl.value = profile.avatarUrl;
  nickname.value = profile.nickname;
});

function navigateTo(item: MenuEntry) {
  if (navigationPending.value) {
    return;
  }

  navigationPending.value = true;

  const handleFailure = () => {
    navigationPending.value = false;
    uni.showToast({ title: "页面打开失败，请重试", icon: "none" });
  };

  if (
    item.route === "/pages/registration/index" ||
    item.route === "/pages/messages/index"
  ) {
    uni.switchTab({
      url: item.route,
      fail: handleFailure,
    });
    return;
  }

  uni.navigateTo({
    url: item.route,
    fail: handleFailure,
  });
}

function toggleAppMode() {
  if (navigationPending.value) {
    return;
  }

  const targetMode = isDoctorMode.value ? "patient" : "doctor";
  if (!setActiveMode(targetMode)) {
    uni.showToast({ title: "当前账号没有医生权限", icon: "none" });
    return;
  }

  applyAppModeNavigation(targetMode);
  navigationPending.value = true;
  uni.switchTab({
    url: "/pages/home/index",
    fail: () => {
      navigationPending.value = false;
      uni.showToast({ title: "页面切换失败，请重试", icon: "none" });
    },
  });
}
</script>

<template>
  <view class="profile-page">
    <ProfileHero :avatar-url="avatarUrl" :nickname="nickname" />
    <view v-if="canSwitchMode" class="profile-page__mode">
      <view class="profile-page__mode-copy">
        <text class="profile-page__mode-label">当前模式</text>
        <text class="profile-page__mode-value">
          {{ isDoctorMode ? "医生工作台" : "患者端" }}
        </text>
      </view>
      <button
        class="profile-page__mode-button"
        :disabled="navigationPending"
        @tap="toggleAppMode"
      >
        {{ isDoctorMode ? "切换到患者端" : "切换到医生工作台" }}
      </button>
    </view>
    <ActionMenu :items="menuItems" flat compact @select="navigateTo" />
  </view>
</template>

<style scoped>
.profile-page {
  min-height: 100vh;
  background: #f4f5f7;
}

.profile-page__mode {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 28rpx;
  margin: 20rpx 22rpx;
  background: #ffffff;
  border: 1rpx solid #e6ebf2;
  border-radius: 22rpx;
}

.profile-page__mode-copy {
  display: flex;
  flex-direction: column;
}

.profile-page__mode-label {
  color: #8c96a7;
  font-size: 22rpx;
}

.profile-page__mode-value {
  margin-top: 6rpx;
  color: #202938;
  font-size: 29rpx;
  font-weight: 600;
}

.profile-page__mode-button {
  padding: 0 24rpx;
  margin: 0;
  color: #087fe1;
  font-size: 24rpx;
  line-height: 64rpx;
  background: #edf7ff;
  border-radius: 32rpx;
}

.profile-page__mode-button::after {
  display: none;
}
</style>
