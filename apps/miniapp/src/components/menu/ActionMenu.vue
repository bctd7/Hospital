<script setup lang="ts">
import type { MenuEntry } from "@/types/menu";

import ActionMenuItem from "./ActionMenuItem.vue";

defineProps<{
  items: MenuEntry[];
}>();

const emit = defineEmits<{
  select: [item: MenuEntry];
}>();
</script>

<template>
  <view class="action-menu">
    <ActionMenuItem
      v-for="(item, index) in items"
      :key="item.id"
      :item="item"
      :last="index === items.length - 1"
      @select="emit('select', $event)"
    />
  </view>
</template>

<style scoped>
.action-menu {
  overflow: hidden;
  background: #ffffff;
  border: 1rpx solid #e8edf3;
  border-radius: 28rpx;
  box-shadow: 0 18rpx 55rpx rgba(35, 70, 112, 0.07);
}
</style>
