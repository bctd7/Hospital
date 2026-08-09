<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import ActionMenu from "@/components/menu/ActionMenu.vue";
import ProfileHero from "@/components/profile/ProfileHero.vue";
import {
  availableAppVariants,
  sessionState,
  setAppVariant,
} from "@/stores/session";
import type { MenuEntry } from "@/types/menu";
import { applyAppVariantNavigation } from "@/utils/appShell";
import { getDisplayProfile } from "@/utils/displayProfile";

const avatarUrl = ref("");
const nickname = ref("微信用户");
const navigationPending = ref(false);
const canSwitchApp = computed(() => availableAppVariants().includes("staff"));
const isStaffApp = computed(() => sessionState.appVariant === "staff");

const menuItems: MenuEntry[] = [
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

function toggleAppVariant() {
  if (navigationPending.value) {
    return;
  }

  const target = isStaffApp.value ? "patient" : "staff";
  if (!setAppVariant(target)) {
    uni.showToast({ title: "当前账号没有工作人员权限", icon: "none" });
    return;
  }

  applyAppVariantNavigation(target);
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
    <view v-if="canSwitchApp" class="profile-page__variant">
      <view>
        <text class="profile-page__variant-label">当前界面</text>
        <text class="profile-page__variant-value">
          {{ isStaffApp ? "工作人员端" : "患者端" }}
        </text>
      </view>
      <button class="profile-page__variant-button" @tap="toggleAppVariant">
        {{ isStaffApp ? "切换到患者端" : "切换到工作人员端" }}
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

.profile-page__variant {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 30rpx;
  margin: 20rpx 24rpx;
  background: #ffffff;
  border-radius: 18rpx;
}

.profile-page__variant-label,
.profile-page__variant-value {
  display: block;
}

.profile-page__variant-label {
  color: #9299a5;
  font-size: 23rpx;
}

.profile-page__variant-value {
  margin-top: 8rpx;
  color: #1c2430;
  font-size: 30rpx;
  font-weight: 600;
}

.profile-page__variant-button {
  margin: 0;
  color: #147fd1;
  font-size: 25rpx;
  line-height: 66rpx;
  background: #edf7ff;
  border-radius: 33rpx;
}

.profile-page__variant-button::after {
  display: none;
}

</style>
