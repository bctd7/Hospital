<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { ref } from "vue";

import ActionMenu from "@/components/menu/ActionMenu.vue";
import ProfileHero from "@/components/profile/ProfileHero.vue";
import type { MenuEntry } from "@/types/menu";
import { getDisplayProfile } from "@/utils/displayProfile";

const avatarUrl = ref("");
const nickname = ref("微信用户");

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
    id: "waitlist",
    title: "候补订单",
    symbol: "候",
    tone: "cyan",
    route: "/pages/profile/waitlist/index",
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
  const profile = getDisplayProfile();
  avatarUrl.value = profile.avatarUrl;
  nickname.value = profile.nickname;
});

function navigateTo(item: MenuEntry) {
  uni.navigateTo({
    url: item.route,
  });
}

function openAccountSettings() {
  uni.navigateTo({
    url: "/pages/profile/account/index",
  });
}
</script>

<template>
  <view class="profile-page">
    <ProfileHero
      :avatar-url="avatarUrl"
      :nickname="nickname"
      @settings="openAccountSettings"
    />
    <ActionMenu :items="menuItems" flat compact @select="navigateTo" />
  </view>
</template>

<style scoped>
.profile-page {
  min-height: 100vh;
  background: #f4f5f7;
}
</style>
