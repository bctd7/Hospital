<script setup lang="ts">
import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
</script>

<template>
  <AppPage title="消息管理" description="后续在这里管理消息偏好和通知状态。">
    <FeaturePlaceholder
      label="静态页面"
      title="消息管理待接入"
      description="当前不请求消息配置接口，也不会修改订阅通知权限。"
    />
  </AppPage>
</template>
