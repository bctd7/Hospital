<script setup lang="ts">
defineProps<{
  title: string;
  description: string;
}>();
</script>

<template>
  <view class="app-page">
    <view class="app-page__heading">
      <text class="app-page__title">{{ title }}</text>
      <text class="app-page__description">{{ description }}</text>
    </view>

    <view class="app-page__content">
      <slot />
    </view>
  </view>
</template>

<style scoped>
.app-page {
  min-height: 100vh;
  padding: 48rpx 32rpx 64rpx;
  background: #f4f7fb;
}

.app-page__heading {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  margin-bottom: 32rpx;
}

.app-page__title {
  color: #172033;
  font-size: 44rpx;
  font-weight: 650;
  line-height: 1.3;
}

.app-page__description {
  color: #7f8a9d;
  font-size: 26rpx;
  line-height: 1.6;
}

.app-page__content {
  width: 100%;
}
</style>
