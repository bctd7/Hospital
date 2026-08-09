<script setup lang="ts">
import { computed } from "vue";

import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
import { sessionState } from "@/stores/session";

const isStaffApp = computed(() => sessionState.appVariant === "staff");
</script>

<template>
  <AppPage
    :title="isStaffApp ? '工作台' : '首页'"
    :description="isStaffApp ? '工作人员功能将在后续业务开发中接入。' : '医院服务和预约摘要将在这里逐步接入。'"
  >
    <FeaturePlaceholder
      :label="isStaffApp ? '工作人员端' : '患者端'"
      :title="isStaffApp ? '工作台暂未建设' : '欢迎使用 Hospital'"
      :description="isStaffApp ? '当前保留稳定入口，不展示模拟业务数据。' : '当前已完成小程序基础框架，后续将在这里添加公告、预约摘要和快捷入口。'"
    />
  </AppPage>
</template>
