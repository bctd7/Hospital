<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { ref } from "vue";

import ProfileAvatar from "@/components/profile/ProfileAvatar.vue";
import { initializeFromWechat } from "@/stores/session";
import { getDisplayProfile, saveDisplayProfile } from "@/utils/displayProfile";

interface ChooseAvatarEvent {
  detail: {
    avatarUrl?: string;
  };
}

interface NicknameInputEvent {
  detail: {
    value?: string;
  };
}

const avatarUrl = ref("");
const nickname = ref("");
const entering = ref(false);

onLoad(() => {
  const profile = getDisplayProfile();
  avatarUrl.value = profile.avatarUrl;
  nickname.value = profile.nickname === "微信用户" ? "" : profile.nickname;
});

function chooseAvatar(event: ChooseAvatarEvent) {
  avatarUrl.value = event.detail.avatarUrl ?? "";
}

function updateNickname(event: Event) {
  const inputEvent = event as unknown as NicknameInputEvent;
  nickname.value = inputEvent.detail.value ?? "";
}

async function enterMiniapp() {
  if (entering.value) {
    return;
  }

  entering.value = true;
  saveDisplayProfile({
    avatarUrl: avatarUrl.value,
    nickname: nickname.value,
  });

  const authenticated = await initializeFromWechat();

  uni.switchTab({
    url: "/pages/home/index",
    success: () => {
      if (!authenticated) {
        uni.showToast({
          title: "暂以访客身份进入",
          icon: "none",
        });
      }
    },
    fail: () => {
      entering.value = false;
      uni.showToast({
        title: "暂时无法进入，请重试",
        icon: "none",
      });
    },
  });
}
</script>

<template>
  <view class="entry-page">
    <view class="entry-page__safe-top" />
    <text class="entry-page__brand">Hospital</text>

    <view class="entry-page__form">
      <button
        class="entry-page__avatar-button"
        open-type="chooseAvatar"
        aria-label="选择微信头像"
        @chooseavatar="chooseAvatar"
      >
        <ProfileAvatar :src="avatarUrl" size="large" />
        <text class="entry-page__avatar-tip">点击选择头像</text>
      </button>

      <view class="entry-page__field">
        <text class="entry-page__label">昵称</text>
        <input
          class="entry-page__input"
          type="nickname"
          :value="nickname"
          maxlength="32"
          placeholder="请输入或选择昵称"
          placeholder-class="entry-page__placeholder"
          @input="updateNickname"
        />
      </view>
    </view>

    <view class="entry-page__footer">
      <button
        class="entry-page__enter-button"
        :disabled="entering"
        :loading="entering"
        @tap="enterMiniapp"
      >
        {{ entering ? "正在进入" : "进入小程序" }}
      </button>
      <text class="entry-page__notice">头像和昵称仅用于页面展示，暂不代表就诊人身份。</text>
    </view>
  </view>
</template>

<style scoped>
.entry-page {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  padding: 0 72rpx calc(env(safe-area-inset-bottom) + 34rpx);
  background: #ffffff;
}

.entry-page__safe-top {
  height: calc(var(--status-bar-height) + 44rpx);
}

.entry-page__brand {
  color: #17191d;
  font-size: 42rpx;
  font-weight: 700;
  line-height: 1.4;
  text-align: center;
}

.entry-page__form {
  display: flex;
  flex-direction: column;
  margin-top: 260rpx;
}

.entry-page__avatar-button {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0;
  margin: 0 auto 90rpx;
  overflow: visible;
  color: inherit;
  line-height: normal;
  background: transparent;
  border: 0;
  border-radius: 0;
}

.entry-page__avatar-button::after {
  display: none;
}

.entry-page__avatar-tip {
  margin-top: 18rpx;
  color: #a2a7af;
  font-size: 23rpx;
  line-height: 1.4;
}

.entry-page__field {
  display: flex;
  align-items: center;
  min-height: 104rpx;
  border-top: 1rpx solid #e5e7eb;
  border-bottom: 1rpx solid #e5e7eb;
}

.entry-page__label {
  flex: 0 0 auto;
  width: 150rpx;
  color: #26292f;
  font-size: 31rpx;
}

.entry-page__input {
  height: 104rpx;
  flex: 1;
  color: #26292f;
  font-size: 30rpx;
  line-height: 104rpx;
}

.entry-page__placeholder {
  color: #a4a8af;
}

.entry-page__footer {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-top: auto;
}

.entry-page__enter-button {
  width: 100%;
  height: 92rpx;
  color: #ffffff;
  font-size: 31rpx;
  font-weight: 600;
  line-height: 92rpx;
  background: #1684e6;
  border-radius: 18rpx;
  box-shadow: 0 14rpx 32rpx rgba(22, 132, 230, 0.2);
}

.entry-page__enter-button::after {
  display: none;
}

.entry-page__enter-button[disabled] {
  color: rgba(255, 255, 255, 0.85);
  background: #72afe7;
}

.entry-page__notice {
  margin-top: 22rpx;
  color: #a0a6af;
  font-size: 21rpx;
  line-height: 1.6;
  text-align: center;
}
</style>
