<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { ref } from "vue";

import { staffBookingApi } from "@/api/appointment";
import type { ExaminationReportVersion } from "@/types/appointment";

const versions = ref<ExaminationReportVersion[]>([]);
const loading = ref(false);
const errorMessage = ref("");

onLoad((query) => {
  const reportId = decode(String(query?.report_id ?? ""));
  if (reportId) void load(reportId);
  else errorMessage.value = "缺少报告编号";
});

async function load(reportId: string) {
  loading.value = true;
  try {
    versions.value = await staffBookingApi.listReportVersions(reportId);
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "版本记录加载失败";
  } finally {
    loading.value = false;
  }
}
function decode(value: string) { try { return decodeURIComponent(value); } catch { return ""; } }
</script>

<template>
  <view class="history-page">
    <view v-if="loading" class="state">正在加载…</view>
    <view v-else-if="errorMessage" class="state error">{{ errorMessage }}</view>
    <view v-for="version in versions" :key="version.versionId" class="version-card">
      <view class="heading"><text>第 {{ version.versionNo }} 版</text><text>{{ version.versionKind === 'correction' ? '更正版' : '原始版' }}</text></view>
      <text v-if="version.correctionReason" class="reason">更正原因：{{ version.correctionReason }}</text>
      <text class="label">客观所见</text><text class="content">{{ version.objectiveFindings || '—' }}</text>
      <text class="label">检查结论</text><text class="content">{{ version.impression || '—' }}</text>
      <text class="label">建议</text><text class="content">{{ version.recommendation || '—' }}</text>
      <text class="label">备注</text><text class="content">{{ version.notes || '—' }}</text>
      <view class="meta"><text>记录人：{{ version.authoredByDisplayName }}</text><text>发布人：{{ version.publishedByDisplayName || '—' }}</text><text>{{ version.publishedAt || version.updatedAt }}</text></view>
    </view>
  </view>
</template>

<style scoped>
.history-page{min-height:100vh;padding:22rpx;box-sizing:border-box;background:#f3f6f9}.version-card,.state{margin-bottom:18rpx;padding:24rpx;background:#fff;border:1rpx solid #e2e8ee;border-radius:20rpx}.heading{display:flex;justify-content:space-between;color:#2d3b4f;font-size:26rpx;font-weight:700}.heading text:last-child{color:#247eb7;font-size:20rpx}.reason{display:block;margin-top:16rpx;padding:14rpx;color:#9b6925;font-size:21rpx;background:#fff7e7;border-radius:12rpx}.label{display:block;margin-top:20rpx;color:#7c899a;font-size:19rpx}.content{display:block;margin-top:7rpx;color:#344257;font-size:23rpx;line-height:1.65;white-space:pre-wrap}.meta{margin-top:22rpx;padding-top:18rpx;border-top:1rpx solid #e8edf1}.meta text{display:block;margin-top:6rpx;color:#758296;font-size:19rpx}.state{color:#8490a0;font-size:22rpx;text-align:center}.error{color:#be4e5d}
</style>
