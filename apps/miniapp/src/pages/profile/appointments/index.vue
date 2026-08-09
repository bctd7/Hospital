<script setup lang="ts">
import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
</script>

<template>
  <AppPage title="我的预约" description="后续在这里查看预约记录与状态。">
    <FeaturePlaceholder
      label="静态页面"
      title="暂无预约"
      description="当前不请求预约接口，业务规则确认后再接入列表、筛选和预约详情。"
    />
  </AppPage>
</template>
