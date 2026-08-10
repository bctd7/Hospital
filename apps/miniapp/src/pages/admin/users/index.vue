<script setup lang="ts">
import { onLoad, onReachBottom, onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { staffManagementApi } from "@/api/staffManagement";
import AdminUserListItem from "@/components/admin/AdminUserListItem.vue";
import { sessionState } from "@/stores/session";
import type { AccountIdentityType, AccountStatus, AdminAccountSummary } from "@/types/staffManagement";
import { hasIdentityPermission } from "@/utils/appShell";

const PAGE_SIZE = 20;
const filters: Array<{ label: string; identity: AccountIdentityType | "all"; status: AccountStatus | "all" }> = [
  { label: "全部", identity: "all", status: "all" },
  { label: "普通用户", identity: "patient", status: "active" },
  { label: "医生", identity: "doctor", status: "active" },
  { label: "已禁用", identity: "all", status: "disabled" },
];

const keyword = ref("");
const activeFilter = ref(0);
const accounts = ref<AdminAccountSummary[]>([]);
const page = ref(1);
const total = ref(0);
const loading = ref(false);
const loadingMore = ref(false);
const loaded = ref(false);
const error = ref("");
const navigationPending = ref(false);

const canManageUsers = computed(
  () =>
    sessionState.principal?.roles.includes("super_admin") === true &&
    (hasIdentityPermission(sessionState.principal, "identity.authorization.manage") ||
      hasIdentityPermission(sessionState.principal, "identity.account.manage")),
);
const hasMore = computed(() => accounts.value.length < total.value);

onLoad(() => {
  if (!canManageUsers.value) {
    uni.showToast({ title: "仅超级管理员可访问", icon: "none" });
    setTimeout(() => uni.navigateBack(), 600);
  }
});

onShow(() => {
  navigationPending.value = false;
  if (canManageUsers.value) void search(true);
});

onReachBottom(() => {
  if (hasMore.value && !loading.value && !loadingMore.value) void loadMore();
});

function normalizePhone(value: string): string {
  return value.replace(/[\s-]/g, "");
}

function isFullPhone(value: string): boolean {
  return /^1\d{10}$/.test(normalizePhone(value));
}

async function search(reset = true) {
  if (!canManageUsers.value || loading.value) return;
  loading.value = true;
  error.value = "";
  if (reset) {
    page.value = 1;
    accounts.value = [];
  }
  try {
    const trimmed = keyword.value.trim();
    if (trimmed && isFullPhone(trimmed)) {
      const account = await staffManagementApi.searchAccountByPhone(normalizePhone(trimmed));
      accounts.value = account ? [account] : [];
      total.value = accounts.value.length;
    } else {
      const filter = filters[activeFilter.value];
      const result = await staffManagementApi.listAccounts({
        page: 1,
        pageSize: PAGE_SIZE,
        nickname: trimmed || undefined,
        identityType: filter.identity,
        status: filter.status,
      });
      accounts.value = result.items;
      total.value = result.total;
      page.value = result.page;
    }
  } catch (caught) {
    error.value = messageOf(caught, "用户列表加载失败，请重试");
  } finally {
    loading.value = false;
    loaded.value = true;
  }
}

async function loadMore() {
  const trimmed = keyword.value.trim();
  if (trimmed && isFullPhone(trimmed)) return;
  loadingMore.value = true;
  try {
    const filter = filters[activeFilter.value];
    const nextPage = page.value + 1;
    const result = await staffManagementApi.listAccounts({
      page: nextPage,
      pageSize: PAGE_SIZE,
      nickname: trimmed || undefined,
      identityType: filter.identity,
      status: filter.status,
    });
    const existing = new Set(accounts.value.map((account) => account.accountId));
    accounts.value.push(...result.items.filter((account) => !existing.has(account.accountId)));
    page.value = result.page;
    total.value = result.total;
  } catch (caught) {
    uni.showToast({ title: messageOf(caught, "加载失败"), icon: "none" });
  } finally {
    loadingMore.value = false;
  }
}

function selectFilter(index: number) {
  if (activeFilter.value === index) return;
  activeFilter.value = index;
  void search(true);
}

function openAccount(account: AdminAccountSummary) {
  if (navigationPending.value) return;
  navigationPending.value = true;
  uni.navigateTo({
    url: `/pages/admin/users/detail?account_id=${encodeURIComponent(account.accountId)}`,
    fail: () => {
      navigationPending.value = false;
      uni.showToast({ title: "用户详情打开失败", icon: "none" });
    },
  });
}

function handleInput(event: Event) {
  keyword.value = (event as unknown as { detail: { value: string } }).detail.value;
}

function messageOf(value: unknown, fallback: string): string {
  return value instanceof Error && value.message ? value.message : fallback;
}
</script>

<template>
  <view class="users-page">
    <view class="search-bar">
      <text class="search-icon">⌕</text>
      <input
        :value="keyword"
        class="search-input"
        confirm-type="search"
        placeholder="输入完整手机号，或昵称 / 医生姓名"
        @input="handleInput"
        @confirm="search(true)"
      />
      <button class="search-button" @tap="search(true)">搜索</button>
    </view>

    <scroll-view class="filters" scroll-x :show-scrollbar="false">
      <view class="filters__inner">
        <button
          v-for="(filter, index) in filters"
          :key="filter.label"
          class="filter-button"
          :class="{ 'filter-button--active': activeFilter === index }"
          @tap="selectFilter(index)"
        >{{ filter.label }}</button>
      </view>
    </scroll-view>

    <view class="result-summary">共 {{ total }} 个账号</view>
    <view class="user-list">
      <view v-if="loading" class="page-state">加载中...</view>
      <view v-else-if="error" class="page-state page-state--error">
        <text>{{ error }}</text><button @tap="search(true)">重新加载</button>
      </view>
      <view v-else-if="loaded && accounts.length === 0" class="page-state">没有找到匹配账号</view>
      <template v-else>
        <AdminUserListItem v-for="account in accounts" :key="account.accountId" :account="account" @open="openAccount" />
        <view v-if="loadingMore" class="load-more">加载更多...</view>
        <view v-else-if="accounts.length && !hasMore" class="load-more">已加载全部</view>
      </template>
    </view>
  </view>
</template>

<style scoped>
button::after { display:none; }
.users-page { min-height:100vh; padding:24rpx; box-sizing:border-box; background:#f3f6fa; }
.search-bar { display:flex; align-items:center; padding:10rpx 12rpx 10rpx 22rpx; background:#fff; border:1rpx solid #e5ebf2; border-radius:22rpx; box-shadow:0 8rpx 24rpx rgba(42,66,96,.05); }
.search-icon { color:#94a0b1; font-size:34rpx; }
.search-input { flex:1; height:66rpx; margin:0 12rpx; color:#263248; font-size:24rpx; }
.search-button { width:112rpx; margin:0; padding:0; color:#fff; font-size:24rpx; line-height:60rpx; background:#168bd8; border-radius:30rpx; }
.filters { margin:22rpx 0 14rpx; white-space:nowrap; }
.filters__inner { display:flex; gap:14rpx; }
.filter-button { flex:0 0 auto; width:auto; margin:0; padding:0 26rpx; color:#697588; font-size:23rpx; line-height:58rpx; background:#fff; border-radius:29rpx; }
.filter-button--active { color:#fff; background:#248bd1; }
.result-summary { padding:4rpx 8rpx 14rpx; color:#8f99aa; font-size:21rpx; }
.user-list { overflow:hidden; background:#fff; border-radius:22rpx; box-shadow:0 10rpx 30rpx rgba(41,61,90,.05); }
.page-state { display:flex; flex-direction:column; align-items:center; justify-content:center; gap:20rpx; min-height:320rpx; padding:30rpx; color:#96a0af; font-size:24rpx; text-align:center; }
.page-state--error { color:#c44f61; }
.page-state button { margin:0; padding:0 24rpx; color:#1683d0; font-size:23rpx; line-height:58rpx; background:#edf7ff; border-radius:29rpx; }
.load-more { padding:22rpx; color:#a0a9b7; font-size:21rpx; text-align:center; }
</style>
