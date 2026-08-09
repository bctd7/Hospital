<script setup lang="ts">
import { computed } from "vue";

import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
import { sessionState } from "@/stores/session";

const isDoctorMode = computed(() => sessionState.activeMode === "doctor");
</script>

<template>
  <AppPage
    :title="isDoctorMode ? '预约管理' : '挂号'"
    :description="
      isDoctorMode
        ? '本科室预约查看和处理能力将在预约业务完成后接入。'
        : '检查项目、科室和排班将在业务接口确认后接入。'
    "
  >
    <FeaturePlaceholder
      :label="isDoctorMode ? '医生模式' : '患者模式'"
      :title="isDoctorMode ? '预约管理建设中' : '挂号功能建设中'"
      :description="
        isDoctorMode
          ? '医生模式只会请求当前角色有权访问的预约数据，当前不展示模拟内容。'
          : '本页面暂不请求后端数据，后续将按照接口契约接入检查项目、医生排班和预约流程。'
      "
    />
  </AppPage>
</template>
