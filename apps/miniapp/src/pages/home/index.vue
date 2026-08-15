<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { ref } from "vue";

import PatientServiceHome from "@/components/home/PatientServiceHome.vue";
import { homeWorkbenchAdapter } from "@/services/homeWorkbench";
import { refreshMessageBadge } from "@/services/messageBadge";
import { sessionState } from "@/stores/session";
import type { HomeAction, HomeWorkbenchView } from "@/types/homeWorkbench";

const view = ref<HomeWorkbenchView>();
const loading = ref(false);
const error = ref("");
const navigationPending = ref(false);
let loadGeneration = 0;

onShow(() => {
  navigationPending.value = false;
  uni.setNavigationBarTitle({ title: "首页" });
  void refreshMessageBadge();
  void loadWorkbench();
});

async function loadWorkbench() {
  const generation = ++loadGeneration;
  loading.value = true;
  error.value = "";
  try {
    const result = await homeWorkbenchAdapter.load(
      sessionState.appVariant,
      sessionState.principal,
    );
    if (generation === loadGeneration) view.value = result;
  } catch (cause) {
    if (generation === loadGeneration) {
      error.value = cause instanceof Error ? cause.message : "首页加载失败，请重试";
    }
  } finally {
    if (generation === loadGeneration) loading.value = false;
  }
}

function openAction(action: HomeAction) {
  if (navigationPending.value) return;
  if (action.target.type === "unavailable") {
    uni.showToast({ title: action.target.message, icon: "none" });
    return;
  }
  navigationPending.value = true;
  const options = {
    url: action.target.url,
    fail: () => uni.showToast({ title: `${action.title}打开失败`, icon: "none" }),
    complete: () => { navigationPending.value = false; },
  };
  if (action.target.type === "tab") uni.switchTab(options);
  else uni.navigateTo(options);
}
</script>

<template>
  <view v-if="loading && !view" class="home-state">正在准备工作台…</view>
  <view v-else-if="error && !view" class="home-state home-state--error">
    <text>{{ error }}</text>
    <button @tap="loadWorkbench">重新加载</button>
  </view>
  <PatientServiceHome v-else-if="view" :view="view" @action="openAction" />
</template>

<style scoped>
button::after { display:none; }
.home-state { display:flex; align-items:center; justify-content:center; min-height:100vh; color:#7e899a; font-size:25rpx; background:#f3f6fa; }
.home-state--error { flex-direction:column; gap:24rpx; color:#c34f61; }
.home-state button { margin:0; padding:0 26rpx; color:#1683d0; font-size:23rpx; line-height:60rpx; background:#e7f4fc; border-radius:30rpx; }
</style>
