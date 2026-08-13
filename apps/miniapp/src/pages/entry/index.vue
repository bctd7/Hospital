<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { onUnmounted, ref } from "vue";

import { sendPhoneLoginCode } from "@/api/auth";
import ProfileAvatar from "@/components/profile/ProfileAvatar.vue";
import {
  initializeFromPhone,
  sessionState,
} from "@/stores/session";
import { openAppVariant } from "@/utils/appShell";
import {
  getDisplayProfile,
  saveDisplayProfile,
  saveDisplayProfileToServer,
} from "@/utils/displayProfile";
import { saveSelfPatientPreferences } from "@/utils/selfPatientPreferences";

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
const phone = ref("");
const verificationCode = ref("");
const sendingCode = ref(false);
const countdown = ref(0);
let countdownTimer: ReturnType<typeof setInterval> | null = null;

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

function updatePhone(event: Event) {
  const inputEvent = event as unknown as NicknameInputEvent;
  phone.value = (inputEvent.detail.value ?? "").replace(/\D/g, "").slice(0, 11);
}

function updateVerificationCode(event: Event) {
  const inputEvent = event as unknown as NicknameInputEvent;
  verificationCode.value = (inputEvent.detail.value ?? "").replace(/\D/g, "").slice(0, 8);
}

function startCountdown(seconds: number) {
  countdown.value = Math.max(1, Math.floor(seconds));
  if (countdownTimer) {
    clearInterval(countdownTimer);
  }
  countdownTimer = setInterval(() => {
    countdown.value -= 1;
    if (countdown.value <= 0 && countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }, 1000);
}

async function requestVerificationCode() {
  if (sendingCode.value || countdown.value > 0) {
    return;
  }
  if (!/^1[3-9]\d{9}$/.test(phone.value)) {
    uni.showToast({ title: "请输入正确的手机号", icon: "none" });
    return;
  }
  sendingCode.value = true;
  try {
    const result = await sendPhoneLoginCode(phone.value);
    startCountdown(result.retry_after_seconds || 60);
    uni.showToast({ title: "验证码已发送", icon: "success" });
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : "验证码发送失败",
      icon: "none",
    });
  } finally {
    sendingCode.value = false;
  }
}

async function enterMiniapp() {
  if (entering.value) {
    return;
  }

  if (!/^1[3-9]\d{9}$/.test(phone.value)) {
    uni.showToast({ title: "请输入正确的手机号", icon: "none" });
    return;
  }
  if (!/^\d{4,8}$/.test(verificationCode.value)) {
    uni.showToast({ title: "请输入短信验证码", icon: "none" });
    return;
  }

  entering.value = true;
  saveDisplayProfile({
    avatarUrl: avatarUrl.value,
    nickname: nickname.value,
  });

  const authenticated = await initializeFromPhone(phone.value, verificationCode.value);

  if (!authenticated) {
    entering.value = false;
    uni.showToast({ title: "手机号或验证码不正确", icon: "none" });
    return;
  }

  saveSelfPatientPreferences({
    phoneMasked: `${phone.value.slice(0, 3)}****${phone.value.slice(7)}`,
  });

  try {
    await saveDisplayProfileToServer({
      avatarUrl: avatarUrl.value,
      nickname: nickname.value,
    });
  } catch {
    // 登录已经成功，昵称同步失败不阻止进入；用户可在本人资料页再次保存。
  }

  openAppVariant(sessionState.appVariant, () => {
    entering.value = false;
    uni.showToast({
      title: "暂时无法进入，请重试",
      icon: "none",
    });
  });
}

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer);
  }
});
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

      <view class="entry-page__field">
        <text class="entry-page__label">手机号</text>
        <input
          class="entry-page__input"
          type="number"
          :value="phone"
          maxlength="11"
          placeholder="请输入手机号"
          placeholder-class="entry-page__placeholder"
          @input="updatePhone"
        />
      </view>

      <view class="entry-page__field">
        <text class="entry-page__label">验证码</text>
        <input
          class="entry-page__input"
          type="number"
          :value="verificationCode"
          maxlength="8"
          placeholder="请输入验证码"
          placeholder-class="entry-page__placeholder"
          @input="updateVerificationCode"
        />
        <button
          class="entry-page__code-button"
          :disabled="sendingCode || countdown > 0"
          @tap="requestVerificationCode"
        >
          {{ countdown > 0 ? `${countdown}s` : sendingCode ? "发送中" : "获取验证码" }}
        </button>
      </view>
    </view>

    <view class="entry-page__footer">
      <button
        class="entry-page__enter-button"
        :disabled="entering"
        :loading="entering"
        @tap="enterMiniapp"
      >
        {{ entering ? "正在登录" : "手机号登录 / 注册" }}
      </button>
      <text class="entry-page__notice">手机号验证用于账号登录；头像和昵称仅用于页面展示，不代表医疗实名。</text>
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

.entry-page__code-button {
  flex: 0 0 auto;
  padding: 0 18rpx;
  margin: 0;
  color: #1684e6;
  font-size: 24rpx;
  line-height: 64rpx;
  background: #eef7ff;
  border-radius: 12rpx;
}

.entry-page__code-button::after {
  display: none;
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
