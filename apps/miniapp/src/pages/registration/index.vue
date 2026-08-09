<script setup lang="ts">
import { computed } from "vue";

import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
import { sessionState } from "@/stores/session";

const isStaffApp = computed(() => sessionState.appVariant === "staff");
</script>

<template>
  <AppPage
    :title="isStaffApp ? '业务管理' : '挂号'"
    :description="isStaffApp ? '工作人员业务入口将在后续逐项开放。' : '检查项目、科室和排班将在业务接口确认后接入。'"
  >
    <FeaturePlaceholder
      :label="isStaffApp ? '工作人员端' : '患者端'"
      :title="isStaffApp ? '业务管理暂未建设' : '挂号功能建设中'"
      :description="isStaffApp ? '当前页面暂时留空，后续根据权限接入账号、预约或检查管理。' : '本页面暂不请求后端数据，后续将按照接口契约接入检查项目、医生排班和预约流程。'"
    />
  </AppPage>
</template>
