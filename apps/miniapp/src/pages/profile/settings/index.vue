<script setup lang="ts">
import { ref } from "vue";

import InfoList from "@/components/info/InfoList.vue";
import AppPage from "@/components/layout/AppPage.vue";
import { logout, sessionState } from "@/stores/session";
import type { InfoRow } from "@/types/menu";

const loggingOut = ref(false);

const generalRows: InfoRow[] = [
  {
    id: "cache",
    title: "清除缓存",
    description: "后续仅清理可安全重建的本地数据",
    value: "待接入",
  },
  {
    id: "support",
    title: "联系客服",
    description: "后续根据小程序能力和服务时间配置",
    value: "待接入",
  },
];

const privacyRows: InfoRow[] = [
  {
    id: "privacy",
    title: "用户隐私协议",
    description: "后续展示版本化隐私内容",
    value: "待完善",
  },
  {
    id: "close-account",
    title: "账号注销",
    description: "后续按身份确认和数据保留规则实现",
    value: "待接入",
  },
];

async function logoutCurrentSession() {
  if (loggingOut.value) {
    return;
  }

  loggingOut.value = true;
  await logout();

  uni.reLaunch({
    url: "/pages/entry/index",
    fail: () => {
      loggingOut.value = false;
      uni.showToast({
        title: "页面跳转失败，请重试",
        icon: "none",
      });
    },
  });
}
</script>

<template>
  <AppPage title="设置" description="常用工具、服务支持和隐私选项。">
    <view class="settings-page__section">
      <InfoList title="常用设置" :rows="generalRows" />
    </view>
    <view class="settings-page__section">
      <InfoList title="隐私与账号" :rows="privacyRows" />
    </view>
    <button
      v-if="sessionState.status === 'authenticated'"
      class="settings-page__logout"
      :disabled="loggingOut"
      :loading="loggingOut"
      @tap="logoutCurrentSession"
    >
      退出当前登录
    </button>
    <text class="settings-page__hint">除退出当前登录外，其余设置操作暂未启用。</text>
  </AppPage>
</template>

<style scoped>
.settings-page__section + .settings-page__section {
  margin-top: 24rpx;
}

.settings-page__hint {
  display: block;
  padding: 28rpx 20rpx 0;
  color: #9aa4b5;
  font-size: 22rpx;
  line-height: 1.65;
  text-align: center;
}

.settings-page__logout {
  height: 88rpx;
  margin-top: 32rpx;
  color: #d64b4b;
  font-size: 29rpx;
  line-height: 88rpx;
  background: #ffffff;
  border: 1rpx solid #f0dada;
  border-radius: 20rpx;
}

.settings-page__logout::after {
  display: none;
}

.settings-page__logout[disabled] {
  color: #dc8a8a;
  background: #fff8f8;
}
</style>
