<script setup lang="ts">
import ProfileAvatar from "./ProfileAvatar.vue";

withDefaults(
  defineProps<{
    avatarUrl?: string;
    nickname?: string;
  }>(),
  {
    avatarUrl: "",
    nickname: "微信用户",
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
      <ProfileAvatar :src="avatarUrl" />
      <text class="profile-hero__nickname">{{ nickname }}</text>
    </view>
  </view>
</template>

<style scoped>
.profile-hero {
  position: relative;
  min-height: 300rpx;
  padding: calc(var(--status-bar-height) + 58rpx) 28rpx 62rpx;
  overflow: hidden;
  background: linear-gradient(135deg, #559cea 0%, #087fe1 62%, #0e71dd 100%);
}

.profile-hero__content {
  position: relative;
  z-index: 3;
  display: flex;
  align-items: center;
  width: 100%;
}

.profile-hero__nickname {
  min-width: 0;
  flex: 1;
  margin-left: 30rpx;
  overflow: hidden;
  color: #ffffff;
  font-size: 38rpx;
  font-weight: 600;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-hero__orb,
.profile-hero__wave {
  position: absolute;
  pointer-events: none;
}

.profile-hero__orb {
  background: rgba(255, 255, 255, 0.06);
  border-radius: 50%;
}

.profile-hero__orb--left {
  top: -116rpx;
  left: -76rpx;
  width: 300rpx;
  height: 300rpx;
}

.profile-hero__orb--right {
  top: 34rpx;
  right: 68rpx;
  width: 118rpx;
  height: 118rpx;
}

.profile-hero__wave {
  bottom: -200rpx;
  width: 850rpx;
  height: 300rpx;
  border-radius: 50%;
}

.profile-hero__wave--back {
  right: -150rpx;
  background: rgba(21, 100, 218, 0.44);
  transform: rotate(-7deg);
}

.profile-hero__wave--front {
  right: 190rpx;
  bottom: -240rpx;
  background: rgba(14, 83, 195, 0.32);
  transform: rotate(8deg);
}
</style>
