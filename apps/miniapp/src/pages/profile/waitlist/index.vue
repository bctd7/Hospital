<script setup lang="ts">
import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
</script>

<template>
  <AppPage title="候补订单" description="后续在这里查看候补申请和处理状态。">
    <FeaturePlaceholder
      label="静态页面"
      title="暂无候补订单"
      description="当前不请求候补订单接口，也不展示模拟排队或订单数据。"
    />
  </AppPage>
</template>
