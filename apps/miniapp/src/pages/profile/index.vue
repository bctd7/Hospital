<script setup lang="ts">
import ActionMenu from "@/components/menu/ActionMenu.vue";
import ProfileHero from "@/components/profile/ProfileHero.vue";
import type { MenuEntry } from "@/types/menu";

const menuItems: MenuEntry[] = [
  {
    id: "patients",
    title: "就诊人管理",
    description: "本人及家庭就诊人",
    symbol: "人",
    tone: "blue",
    route: "/pages/profile/patients/index",
  },
  {
    id: "appointments",
    title: "我的预约",
    description: "查看预约记录与状态",
    symbol: "约",
    tone: "blue",
    route: "/pages/profile/appointments/index",
  },
  {
    id: "favorites",
    title: "我的关注",
    description: "已关注的医生与服务",
    symbol: "关",
    tone: "orange",
    route: "/pages/profile/favorites/index",
  },
  {
    id: "settings",
    title: "设置",
    description: "清除缓存、联系客服、隐私协议与注销",
    symbol: "设",
    tone: "gray",
    route: "/pages/profile/settings/index",
  },
];

function navigateTo(item: MenuEntry) {
  uni.navigateTo({
    url: item.route,
  });
}
</script>

<template>
  <view class="profile-page">
    <ProfileHero />
    <view class="profile-page__body">
      <ActionMenu :items="menuItems" @select="navigateTo" />
      <text class="profile-page__hint">当前页面仅展示前端布局，不会读取或保存真实医疗数据。</text>
    </view>
  </view>
</template>

<style scoped>
.profile-page {
  min-height: 100vh;
  background: #f4f7fb;
}

.profile-page__body {
  position: relative;
  z-index: 4;
  padding: 0 24rpx 64rpx;
  margin-top: -28rpx;
}

.profile-page__hint {
  display: block;
  padding: 30rpx 20rpx 0;
  color: #9aa4b5;
  font-size: 22rpx;
  line-height: 1.65;
  text-align: center;
}
</style>
