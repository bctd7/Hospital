<script setup lang="ts">
import { onShow, onUnload } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import ProfileAvatar from "@/components/profile/ProfileAvatar.vue";
import { logout, sessionState } from "@/stores/session";
import {
  getDisplayProfile,
  loadDisplayProfileFromServer,
  saveDisplayProfile,
  saveDisplayProfileToServer,
} from "@/utils/displayProfile";
import {
  getSelfPatientPreferences,
  saveSelfPatientPreferences,
} from "@/utils/selfPatientPreferences";

interface ChooseAvatarEvent {
  detail: {
    avatarUrl?: string;
  };
}

interface InputEvent {
  detail: {
    value?: string;
  };
}

const avatarUrl = ref("");
const nickname = ref("微信用户");
const phoneMasked = ref("");
const editingNickname = ref(false);
const loggingOut = ref(false);
const avatarChoosing = ref(false);
let avatarChoosingTimer: ReturnType<typeof setTimeout> | null = null;

const phoneDisplay = computed(() => {
  if (sessionState.status !== "authenticated") {
    return "登录后查看";
  }

  if (phoneMasked.value) {
    return phoneMasked.value;
  }

  return "已验证登录手机号";
});

onShow(() => {
  const profile = getDisplayProfile();
  const preferences = getSelfPatientPreferences();

  avatarUrl.value = profile.avatarUrl;
  nickname.value = profile.nickname;
  phoneMasked.value = preferences.phoneMasked;
  if (sessionState.status === "authenticated") {
    void loadDisplayProfileFromServer()
      .then((serverProfile) => {
        avatarUrl.value = serverProfile.avatarUrl;
        nickname.value = serverProfile.nickname;
      })
      .catch(() => undefined);
  }
});

onUnload(() => {
  if (avatarChoosingTimer) clearTimeout(avatarChoosingTimer);
});

function beginAvatarChoice() {
  avatarChoosing.value = true;
  if (avatarChoosingTimer) clearTimeout(avatarChoosingTimer);
  avatarChoosingTimer = setTimeout(() => {
    avatarChoosing.value = false;
    avatarChoosingTimer = null;
  }, 1200);
}

function chooseAvatar(event: ChooseAvatarEvent) {
  const nextAvatarUrl = event.detail.avatarUrl?.trim() ?? "";
  avatarUrl.value = nextAvatarUrl;
  avatarChoosing.value = false;
  if (avatarChoosingTimer) {
    clearTimeout(avatarChoosingTimer);
    avatarChoosingTimer = null;
  }
  saveDisplayProfile({
    avatarUrl: nextAvatarUrl,
    nickname: nickname.value,
  });
  uni.showToast({ title: "头像已更新", icon: "success" });
}

function updateNickname(event: Event) {
  const inputEvent = event as unknown as InputEvent;
  nickname.value = inputEvent.detail.value ?? "";
}

function beginNicknameEdit() {
  editingNickname.value = true;
}

async function finishNicknameEdit() {
  const input = {
    avatarUrl: avatarUrl.value,
    nickname: nickname.value,
  };
  const local = saveDisplayProfile(input);
  nickname.value = local.nickname;
  editingNickname.value = false;
  if (sessionState.status !== "authenticated") return;
  try {
    const profile = await saveDisplayProfileToServer(input);
    nickname.value = profile.nickname;
    uni.showToast({ title: "用户名已更新", icon: "success" });
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : "用户名同步失败",
      icon: "none",
    });
  }
}

async function logoutCurrentSession() {
  if (loggingOut.value) {
    return;
  }

  loggingOut.value = true;
  await logout();

  uni.reLaunch({
    url: "/pages/entry/index",
    fail: () => {
      loggingOut.value = false;
      uni.showToast({
        title: "页面跳转失败，请重试",
        icon: "none",
      });
    },
  });
}

</script>

<template>
  <view class="patients-page">
    <view class="patients-card">
      <button
        class="patients-row patients-row--avatar"
        open-type="chooseAvatar"
        :disabled="avatarChoosing"
        aria-label="修改本人头像"
        @tap="beginAvatarChoice"
        @chooseavatar="chooseAvatar"
      >
        <text class="patients-row__label">头像</text>
        <view class="patients-row__value patients-row__value--avatar">
          <ProfileAvatar :src="avatarUrl" />
          <text class="patients-row__arrow">›</text>
        </view>
      </button>

      <view class="patients-row" @tap="beginNicknameEdit">
        <text class="patients-row__label">用户名</text>
        <view class="patients-row__value">
          <input
            v-if="editingNickname"
            class="patients-row__input patients-row__input--nickname"
            type="nickname"
            :value="nickname"
            maxlength="32"
            :focus="editingNickname"
            placeholder="请输入用户名"
            @input="updateNickname"
            @blur="finishNicknameEdit"
            @confirm="finishNicknameEdit"
          />
          <text v-else class="patients-row__text">{{ nickname }}</text>
          <text class="patients-row__arrow">›</text>
        </view>
      </view>

    </view>

    <view class="patients-card patients-card--phone">
      <view class="patients-row">
        <text class="patients-row__label">手机号</text>
        <view class="patients-row__value">
          <text class="patients-row__text" :class="{ 'patients-row__text--muted': !phoneMasked }">
            {{ phoneDisplay }}
          </text>
        </view>
      </view>
    </view>

    <text class="patients-page__hint">
      手机号是当前账号的已验证登录标识，首版不支持在本页直接更换，也不会用于自动查询其他人的病历或报告。
    </text>

    <button
      v-if="sessionState.status === 'authenticated'"
      class="patients-page__logout"
      :disabled="loggingOut"
      :loading="loggingOut"
      @tap="logoutCurrentSession"
    >
      退出登录
    </button>
  </view>
</template>

<style scoped>
.patients-page {
  min-height: 100vh;
  padding: 24rpx 0 calc(env(safe-area-inset-bottom) + 48rpx);
  background: #f5f6f8;
}

.patients-card {
  overflow: hidden;
  background: #ffffff;
  border-top: 1rpx solid #eceff2;
  border-bottom: 1rpx solid #eceff2;
}

.patients-card--phone {
  margin-top: 28rpx;
}

.patients-row {
  display: flex;
  min-height: 112rpx;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 30rpx;
  margin: 0;
  color: inherit;
  line-height: normal;
  background: #ffffff;
  border: 0;
  border-radius: 0;
}

.patients-row + .patients-row {
  border-top: 1rpx solid #edf0f3;
}

.patients-row::after {
  display: none;
}

.patients-row--avatar {
  width: 100%;
  min-height: 150rpx;
}

.patients-row__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.patients-row__label {
  flex: 0 0 auto;
  color: #505762;
  font-size: 31rpx;
  line-height: 1.4;
}

.patients-row__description {
  margin-top: 6rpx;
  color: #a0a7b1;
  font-size: 22rpx;
  line-height: 1.4;
}

.patients-row__value {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  margin-left: 28rpx;
}

.patients-row__value--avatar {
  gap: 16rpx;
}

.patients-row__text,
.patients-row__input {
  max-width: 390rpx;
  overflow: hidden;
  color: #2b3038;
  font-size: 30rpx;
  line-height: 1.4;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.patients-row__text--muted {
  color: #a4abb5;
}

.patients-row__input--nickname {
  width: 330rpx;
  height: 68rpx;
}

.patients-row__arrow {
  margin-left: 16rpx;
  color: #c3c7cd;
  font-size: 50rpx;
  font-weight: 300;
  line-height: 1;
}

.patients-page__hint {
  display: block;
  padding: 28rpx 40rpx 0;
  color: #9aa2ad;
  font-size: 22rpx;
  line-height: 1.65;
  text-align: center;
}

.patients-page__logout {
  height: 88rpx;
  margin: 34rpx 30rpx 0;
  color: #d64b4b;
  font-size: 29rpx;
  line-height: 88rpx;
  background: #ffffff;
  border: 1rpx solid #f0dada;
  border-radius: 18rpx;
}

.patients-page__logout::after {
  display: none;
}

.patients-page__logout[disabled] {
  color: #dc8a8a;
  background: #fff8f8;
}
</style>
