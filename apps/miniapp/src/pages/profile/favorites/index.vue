<script setup lang="ts">
import AppPage from "@/components/layout/AppPage.vue";
import FeaturePlaceholder from "@/components/states/FeaturePlaceholder.vue";
</script>

<template>
  <AppPage title="我的关注" description="后续在这里集中展示已关注的医生与服务。">
    <FeaturePlaceholder
      label="静态页面"
      title="暂无关注"
      description="当前不请求关注接口，也不展示模拟医生或服务数据。"
    />
  </AppPage>
</template>
