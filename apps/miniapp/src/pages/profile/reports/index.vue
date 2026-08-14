<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { ref } from "vue";

import { patientAppointmentApi } from "@/api/appointment";
import { sessionState } from "@/stores/session";
import type { ExaminationReport } from "@/types/appointment";

const reports = ref<ExaminationReport[]>([]);
const loading = ref(false);
const errorMessage = ref("");

onShow(() => {
  if (sessionState.appVariant !== "patient") {
    reports.value = [];
    errorMessage.value = "请切换到患者端查看本人检查报告";
    return;
  }
  void loadReports();
});

async function loadReports() {
  if (loading.value || sessionState.appVariant !== "patient") return;
  loading.value = true;
  errorMessage.value = "";
  try {
    reports.value = (await patientAppointmentApi.listMyReports(1, 50)).items;
  } catch (error) {
    errorMessage.value = messageOf(error, "报告加载失败");
  } finally {
    loading.value = false;
  }
}

function openReport(value: ExaminationReport) {
  uni.navigateTo({ url: `/pages/profile/reports/detail?booking_id=${encodeURIComponent(value.bookingId)}` });
}
function statusLabel(value: ExaminationReport) { return value.currentVersion.versionKind === "correction" ? "已更正" : "已发布"; }
function messageOf(error: unknown, fallback: string) { return error instanceof Error && error.message.trim() ? error.message : fallback; }
</script>

<template>
  <view class="report-list-page">
    <view v-if="loading" class="state">正在加载报告…</view>
    <view v-else-if="errorMessage" class="state error" @tap="loadReports">{{ errorMessage }}</view>
    <view v-else-if="!reports.length" class="state"><text>暂无已发布的检查报告</text><text class="minor">检查完成并发布后会显示在这里</text></view>
    <view v-for="report in reports" v-else :key="report.reportId" class="report-card" @tap="openReport(report)">
      <view class="heading"><text>{{ report.itemName }}</text><text class="status">{{ statusLabel(report) }}</text></view>
      <text class="line">{{ report.departmentName }} · {{ report.campusName }}</text>
      <text class="line">{{ report.building }} · {{ report.floorNumber }}层 · {{ report.roomNumber }}室</text>
      <view class="footer"><text>{{ report.examinationCompletedAt || report.currentVersion.publishedAt }}</text><text>查看报告 ›</text></view>
    </view>
  </view>
</template>

<style scoped>
.report-list-page{min-height:100vh;padding:22rpx;box-sizing:border-box;background:#f3f6f9}.report-card,.state{margin-bottom:18rpx;padding:24rpx;background:#fff;border:1rpx solid #e1e7ed;border-radius:20rpx}.heading,.footer{display:flex;align-items:center;justify-content:space-between}.heading>text:first-child{color:#29374b;font-size:27rpx;font-weight:700}.status{padding:5rpx 12rpx;color:#287b5e;font-size:18rpx;background:#eaf7f1;border-radius:14rpx}.line{display:block;margin-top:12rpx;color:#748195;font-size:21rpx}.footer{margin-top:20rpx;padding-top:16rpx;color:#8793a4;font-size:19rpx;border-top:1rpx solid #e8edf1}.footer text:last-child{color:#197fbd}.state{color:#8490a0;font-size:22rpx;text-align:center}.state text{display:block}.state .minor{margin-top:10rpx;font-size:19rpx}.error{color:#be4e5d}
</style>
