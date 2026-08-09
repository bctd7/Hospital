<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string;
    description?: string;
  }>(),
  {
    title: "个人中心",
    description: "管理就诊人、预约与常用设置",
  },
);
</script>

<template>
  <view class="profile-hero">
    <view class="profile-hero__orb profile-hero__orb--left" />
    <view class="profile-hero__orb profile-hero__orb--right" />
    <view class="profile-hero__wave profile-hero__wave--back" />
    <view class="profile-hero__wave profile-hero__wave--front" />

    <view class="profile-hero__content">
      <view class="profile-hero__mark" aria-hidden="true">
        <view class="profile-hero__mark-ring" />
      </view>
      <view class="profile-hero__copy">
        <text class="profile-hero__title">{{ title }}</text>
        <text class="profile-hero__description">{{ description }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped>
.profile-hero {
  position: relative;
  min-height: 300rpx;
  padding: calc(var(--status-bar-height) + 52rpx) 40rpx 70rpx;
  overflow: hidden;
  background: linear-gradient(135deg, #4d99ea 0%, #087fe1 58%, #0f70db 100%);
}

.profile-hero__content {
  position: relative;
  z-index: 3;
  display: flex;
  align-items: center;
  max-width: 660rpx;
}

.profile-hero__mark {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 104rpx;
  height: 104rpx;
  margin-right: 28rpx;
  background: rgba(255, 255, 255, 0.18);
  border: 2rpx solid rgba(255, 255, 255, 0.56);
  border-radius: 34rpx;
  box-shadow: 0 16rpx 42rpx rgba(7, 73, 151, 0.2);
}

.profile-hero__mark-ring {
  width: 38rpx;
  height: 38rpx;
  border: 12rpx solid #ffffff;
  border-radius: 50%;
}

.profile-hero__copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.profile-hero__title {
  color: #ffffff;
  font-size: 44rpx;
  font-weight: 700;
  line-height: 1.35;
}

.profile-hero__description {
  margin-top: 10rpx;
  color: rgba(255, 255, 255, 0.82);
  font-size: 25rpx;
  line-height: 1.5;
}

.profile-hero__orb,
.profile-hero__wave {
  position: absolute;
  pointer-events: none;
}

.profile-hero__orb {
  background: rgba(255, 255, 255, 0.07);
  border-radius: 50%;
}

.profile-hero__orb--left {
  top: -100rpx;
  left: -70rpx;
  width: 280rpx;
  height: 280rpx;
}

.profile-hero__orb--right {
  top: 20rpx;
  right: 38rpx;
  width: 126rpx;
  height: 126rpx;
}

.profile-hero__wave {
  right: -160rpx;
  bottom: -170rpx;
  width: 850rpx;
  height: 290rpx;
  border-radius: 50%;
  transform: rotate(-7deg);
}

.profile-hero__wave--back {
  background: rgba(22, 103, 222, 0.42);
}

.profile-hero__wave--front {
  right: 180rpx;
  bottom: -225rpx;
  background: rgba(17, 91, 205, 0.34);
  transform: rotate(8deg);
}
</style>
