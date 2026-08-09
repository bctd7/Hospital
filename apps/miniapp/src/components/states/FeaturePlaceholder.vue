<script setup lang="ts">
withDefaults(
  defineProps<{
    label?: string;
    title: string;
    description: string;
  }>(),
  {
    label: "基础框架已就绪",
  },
);
</script>

<template>
  <view class="placeholder" role="status">
    <view class="placeholder__mark" aria-hidden="true">
      <view class="placeholder__mark-core" />
    </view>
    <text class="placeholder__label">{{ label }}</text>
    <text class="placeholder__title">{{ title }}</text>
    <text class="placeholder__description">{{ description }}</text>
  </view>
</template>

<style scoped>
.placeholder {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-height: 360rpx;
  padding: 48rpx;
  overflow: hidden;
  background: #ffffff;
  border: 1rpx solid #e7ebf1;
  border-radius: 28rpx;
  box-shadow: 0 20rpx 60rpx rgb(35 70 112 / 8%);
}

.placeholder__mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80rpx;
  height: 80rpx;
  margin-bottom: 36rpx;
  background: #e6f6ff;
  border-radius: 24rpx;
}

.placeholder__mark-core {
  width: 28rpx;
  height: 28rpx;
  background: #1497e3;
  border: 10rpx solid #b8e5fb;
  border-radius: 50%;
}

.placeholder__label {
  margin-bottom: 12rpx;
  color: #1497e3;
  font-size: 24rpx;
  font-weight: 600;
  letter-spacing: 1rpx;
}

.placeholder__title {
  margin-bottom: 16rpx;
  color: #172033;
  font-size: 36rpx;
  font-weight: 650;
  line-height: 1.4;
}

.placeholder__description {
  max-width: 560rpx;
  color: #7f8a9d;
  font-size: 27rpx;
  line-height: 1.8;
}
</style>
