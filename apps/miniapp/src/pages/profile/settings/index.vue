<script setup lang="ts">
import InfoList from "@/components/info/InfoList.vue";
import AppPage from "@/components/layout/AppPage.vue";
import type { InfoRow } from "@/types/menu";

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
</script>

<template>
  <AppPage title="设置" description="常用工具、服务支持和隐私选项。">
    <view class="settings-page__section">
      <InfoList title="常用设置" :rows="generalRows" />
    </view>
    <view class="settings-page__section">
      <InfoList title="隐私与账号" :rows="privacyRows" />
    </view>
    <text class="settings-page__hint">当前仅展示页面结构，所有操作均未启用。</text>
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
</style>
