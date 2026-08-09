<script setup lang="ts">
import { computed } from "vue";

import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
import { sessionState } from "@/stores/session";

const isDoctorMode = computed(() => sessionState.activeMode === "doctor");
</script>

<template>
  <AppPage
    :title="isDoctorMode ? '医生工作台' : '首页'"
    :description="
      isDoctorMode
        ? '本科室预约、待处理事项和医生快捷入口将在这里接入。'
        : '医院服务和预约摘要将在这里逐步接入。'
    "
  >
    <FeaturePlaceholder
      :label="isDoctorMode ? '医生模式' : '患者模式'"
      :title="isDoctorMode ? '医生工作台建设中' : '欢迎使用 Hospital'"
      :description="
        isDoctorMode
          ? '当前只完成安全的模式切换骨架，不展示模拟患者、排班或预约数据。'
          : '当前已完成小程序基础框架，后续将在这里添加公告、预约摘要和快捷入口。'
      "
    />
  </AppPage>
</template>
