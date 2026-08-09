<script setup lang="ts">
import type { AdminAccountSummary } from "@/types/staffManagement";

defineProps<{ account: AdminAccountSummary }>();
defineEmits<{ (event: "open", account: AdminAccountSummary): void }>();

const identityLabels = {
  patient: "普通用户",
  doctor: "医生",
  super_admin: "超级管理员",
} as const;
</script>

<template>
  <button class="user-card" @tap="$emit('open', account)">
    <view class="user-avatar">{{ (account.displayName || account.nickname || "用").slice(0, 1) }}</view>
    <view class="user-card__body">
      <view class="user-card__title-row">
        <text class="user-card__name">{{ account.displayName || account.nickname || "未设置昵称" }}</text>
        <text class="identity-badge" :class="`identity-badge--${account.identityType}`">
          {{ identityLabels[account.identityType] }}
        </text>
        <text v-if="account.accountStatus === 'disabled'" class="disabled-badge">已禁用</text>
      </view>
      <text class="user-card__meta">
        {{ account.maskedPhone || "未绑定手机号" }}{{ account.departmentName ? ` · ${account.departmentName}` : "" }}
      </text>
    </view>
    <text class="user-card__arrow">›</text>
  </button>
</template>

<style scoped>
button::after { display: none; }
.user-card { display:flex; align-items:center; width:100%; margin:0; padding:26rpx 24rpx; text-align:left; background:#fff; border-radius:0; border-bottom:1rpx solid #edf1f5; }
.user-card:active { background:#f7fbff; }
.user-avatar { display:flex; flex:0 0 auto; align-items:center; justify-content:center; width:76rpx; height:76rpx; color:#fff; font-size:28rpx; font-weight:700; background:linear-gradient(145deg,#62b9ef,#587ee8); border-radius:50%; }
.user-card__body { flex:1; min-width:0; margin-left:20rpx; }
.user-card__title-row { display:flex; align-items:center; gap:10rpx; min-width:0; }
.user-card__name { max-width:220rpx; overflow:hidden; color:#243149; font-size:28rpx; font-weight:650; text-overflow:ellipsis; white-space:nowrap; }
.identity-badge,.disabled-badge { flex:0 0 auto; padding:4rpx 10rpx; font-size:19rpx; border-radius:9rpx; }
.identity-badge { color:#596579; background:#eef2f7; }
.identity-badge--doctor { color:#1479bd; background:#e7f4fd; }
.identity-badge--super_admin { color:#7654bd; background:#f0eafd; }
.disabled-badge { color:#bd4c5d; background:#fdecef; }
.user-card__meta { display:block; margin-top:10rpx; overflow:hidden; color:#929cac; font-size:22rpx; text-overflow:ellipsis; white-space:nowrap; }
.user-card__arrow { margin-left:14rpx; color:#c1c9d5; font-size:40rpx; }
</style>
