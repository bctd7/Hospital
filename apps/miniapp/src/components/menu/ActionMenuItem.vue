<script setup lang="ts">
import type { MenuEntry } from "@/types/menu";

defineProps<{
  item: MenuEntry;
  last?: boolean;
}>();

const emit = defineEmits<{
  select: [item: MenuEntry];
}>();
</script>

<template>
  <view
    class="menu-item"
    :class="{ 'menu-item--last': last }"
    hover-class="menu-item--pressed"
    role="button"
    :aria-label="item.description ? `${item.title}，${item.description}` : item.title"
    @tap="emit('select', item)"
  >
    <view class="menu-item__symbol" :class="`menu-item__symbol--${item.tone}`" aria-hidden="true">
      <text>{{ item.symbol }}</text>
    </view>
    <view class="menu-item__copy">
      <text class="menu-item__title">{{ item.title }}</text>
      <text v-if="item.description" class="menu-item__description">{{ item.description }}</text>
    </view>
    <text class="menu-item__arrow" aria-hidden="true">›</text>
  </view>
</template>

<style scoped>
.menu-item {
  display: flex;
  align-items: center;
  min-height: 128rpx;
  margin-left: 28rpx;
  padding: 24rpx 28rpx 24rpx 0;
  border-bottom: 1rpx solid #e8edf3;
  transition: background-color 120ms ease;
}

.menu-item--last {
  border-bottom: 0;
}

.menu-item--pressed {
  background: #f7f9fc;
}

.menu-item__symbol {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 76rpx;
  height: 76rpx;
  margin-right: 26rpx;
  color: #ffffff;
  font-size: 27rpx;
  font-weight: 700;
  border-radius: 24rpx;
}

.menu-item__symbol--blue {
  background: linear-gradient(145deg, #82a7ff, #5d7ff4);
}

.menu-item__symbol--cyan {
  background: linear-gradient(145deg, #63d1f3, #37aee6);
}

.menu-item__symbol--orange {
  background: linear-gradient(145deg, #ffaf55, #ff812b);
}

.menu-item__symbol--gray {
  background: linear-gradient(145deg, #c5cbd4, #9da6b2);
}

.menu-item__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.menu-item__title {
  color: #283345;
  font-size: 31rpx;
  line-height: 1.45;
}

.menu-item__description {
  max-width: 500rpx;
  margin-top: 6rpx;
  overflow: hidden;
  color: #9099a8;
  font-size: 23rpx;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.menu-item__arrow {
  flex: 0 0 auto;
  margin-left: 18rpx;
  color: #c1c8d2;
  font-family: Arial, sans-serif;
  font-size: 58rpx;
  font-weight: 200;
  line-height: 1;
}
</style>
