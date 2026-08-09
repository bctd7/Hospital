<script setup lang="ts">
import InfoList from "@/components/info/InfoList.vue";
import AppPage from "@/components/layout/AppPage.vue";
import type { InfoRow } from "@/types/menu";

const accountRows: InfoRow[] = [
  {
    id: "login-status",
    title: "账号状态",
    description: "后续在此接入微信登录或其他认证方式",
    value: "未登录",
  },
  {
    id: "display-name",
    title: "姓名显示",
    description: "登录并完成身份确认后显示",
    value: "待完善",
  },
];

const patientRows: InfoRow[] = [
  {
    id: "default-patient",
    title: "默认就诊人",
    description: "用于预约时快速选择",
    value: "尚未添加",
  },
  {
    id: "family-patients",
    title: "家庭就诊人",
    description: "后续支持管理本人及家庭成员",
    value: "0 人",
  },
];
</script>

<template>
  <AppPage title="就诊人管理" description="账号信息、姓名和家庭就诊人统一在这里维护。">
    <view class="patients-page__section">
      <InfoList title="账号信息" :rows="accountRows" />
    </view>
    <view class="patients-page__section">
      <InfoList title="就诊人" :rows="patientRows" />
    </view>
    <text class="patients-page__hint">当前均为静态占位状态，不会调用登录或就诊人接口。</text>
  </AppPage>
</template>

<style scoped>
.patients-page__section + .patients-page__section {
  margin-top: 24rpx;
}

.patients-page__hint {
  display: block;
  padding: 28rpx 20rpx 0;
  color: #9aa4b5;
  font-size: 22rpx;
  line-height: 1.65;
  text-align: center;
}
</style>
