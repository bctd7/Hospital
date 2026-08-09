<script setup lang="ts">
import { computed } from "vue";

import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
import { sessionState } from "@/stores/session";

const isStaffApp = computed(() => sessionState.appVariant === "staff");
</script>

<template>
  <AppPage title="消息" description="预约提醒和系统通知将在消息服务完成后接入。">
    <FeaturePlaceholder
      :label="isStaffApp ? '工作人员端' : '患者端'"
      title="暂无消息"
      :description="isStaffApp ? '工作人员消息入口暂时留空。' : '当前页面只提供稳定的一级入口，不展示模拟消息，也不会请求未读数量。'"
    />
  </AppPage>
</template>
