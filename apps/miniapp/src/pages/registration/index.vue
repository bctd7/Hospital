<script setup lang="ts">
import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
</script>

<template>
  <AppPage title="挂号" description="检查项目、科室和排班将在业务接口确认后接入。">
    <FeaturePlaceholder
      title="挂号功能建设中"
      description="本页面暂不请求后端数据，后续将按照接口契约接入检查项目、医生排班和预约流程。"
    />
  </AppPage>
</template>
