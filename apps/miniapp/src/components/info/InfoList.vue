<script setup lang="ts">
import type { InfoRow } from "@/types/menu";

defineProps<{
  title?: string;
  rows: InfoRow[];
}>();
</script>

<template>
  <view class="info-card">
    <text v-if="title" class="info-card__title">{{ title }}</text>
    <view class="info-card__rows">
      <view
        v-for="(row, index) in rows"
        :key="row.id"
        class="info-row"
        :class="{ 'info-row--last': index === rows.length - 1 }"
      >
        <view class="info-row__copy">
          <text class="info-row__title">{{ row.title }}</text>
          <text v-if="row.description" class="info-row__description">{{ row.description }}</text>
        </view>
        <text v-if="row.value" class="info-row__value">{{ row.value }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped>
.info-card {
  overflow: hidden;
  background: #ffffff;
  border: 1rpx solid #e7ebf1;
  border-radius: 28rpx;
  box-shadow: 0 18rpx 55rpx rgba(35, 70, 112, 0.07);
}

.info-card__title {
  display: block;
  padding: 34rpx 32rpx 12rpx;
  color: #172033;
  font-size: 32rpx;
  font-weight: 650;
}

.info-card__rows {
  padding-left: 32rpx;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 116rpx;
  padding: 24rpx 32rpx 24rpx 0;
  border-bottom: 1rpx solid #e8edf3;
}

.info-row--last {
  border-bottom: 0;
}

.info-row__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.info-row__title {
  color: #283345;
  font-size: 29rpx;
  line-height: 1.45;
}

.info-row__description {
  margin-top: 5rpx;
  color: #939dac;
  font-size: 23rpx;
  line-height: 1.5;
}

.info-row__value {
  flex: 0 0 auto;
  margin-left: 28rpx;
  color: #1497e3;
  font-size: 25rpx;
}
</style>
