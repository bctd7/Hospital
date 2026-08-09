<script setup lang="ts">
import InfoList from "@/components/info/InfoList.vue";
import AppPage from "@/components/layout/AppPage.vue";
import type { InfoRow } from "@/types/menu";

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
  <AppPage title="就诊人管理" description="本人及家庭成员的就诊信息将在这里统一维护。">
    <view class="patients-page__section">
      <InfoList title="就诊人" :rows="patientRows" />
    </view>
    <text class="patients-page__hint">当前仅展示静态布局，不会读取或保存真实就诊人数据。</text>
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
