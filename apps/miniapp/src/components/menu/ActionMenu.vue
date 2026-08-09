<script setup lang="ts">
import type { MenuEntry } from "@/types/menu";

import ActionMenuItem from "./ActionMenuItem.vue";

defineProps<{
  items: MenuEntry[];
  flat?: boolean;
  compact?: boolean;
}>();

const emit = defineEmits<{
  select: [item: MenuEntry];
}>();
</script>

<template>
  <view class="action-menu" :class="{ 'action-menu--flat': flat }">
    <ActionMenuItem
      v-for="(item, index) in items"
      :key="item.id"
      :item="item"
      :last="index === items.length - 1"
      :compact="compact"
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

.action-menu--flat {
  border: 0;
  border-radius: 0;
  box-shadow: none;
}
</style>
