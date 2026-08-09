<script setup lang="ts">
import { ref, watch } from "vue";

const props = withDefaults(
  defineProps<{
    src?: string;
    size?: "small" | "large";
  }>(),
  {
    src: "",
    size: "small",
  },
);

const imageFailed = ref(false);

watch(
  () => props.src,
  () => {
    imageFailed.value = false;
  },
);
</script>

<template>
  <view class="profile-avatar" :class="`profile-avatar--${size}`">
    <image
      v-if="src && !imageFailed"
      class="profile-avatar__image"
      :src="src"
      mode="aspectFill"
      @error="imageFailed = true"
    />
    <view v-else class="profile-avatar__fallback" aria-label="默认头像">
      <view class="profile-avatar__head" />
      <view class="profile-avatar__body" />
    </view>
  </view>
</template>

<style scoped>
.profile-avatar {
  position: relative;
  flex: 0 0 auto;
  overflow: hidden;
  background: #f0f2f5;
  border-radius: 50%;
}

.profile-avatar--small {
  width: 104rpx;
  height: 104rpx;
}

.profile-avatar--large {
  width: 152rpx;
  height: 152rpx;
  border-radius: 24rpx;
}

.profile-avatar__image,
.profile-avatar__fallback {
  width: 100%;
  height: 100%;
}

.profile-avatar__fallback {
  position: relative;
  overflow: hidden;
  background: linear-gradient(155deg, #f5f6f8, #e7e9ed);
}

.profile-avatar__head,
.profile-avatar__body {
  position: absolute;
  left: 50%;
  background: #b9bdc4;
  transform: translateX(-50%);
}

.profile-avatar__head {
  top: 24%;
  width: 31%;
  height: 31%;
  border-radius: 50%;
}

.profile-avatar__body {
  bottom: 17%;
  width: 59%;
  height: 34%;
  border-radius: 50% 50% 18% 18%;
}
</style>
