<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { ref } from "vue";

import { patientAppointmentApi } from "@/api/appointment";
import { sessionState } from "@/stores/session";
import type { ExaminationReport } from "@/types/appointment";

const report = ref<ExaminationReport>();
const loading = ref(false);
const errorMessage = ref("");

onLoad((query) => {
  if (sessionState.appVariant !== "patient") {
    errorMessage.value = "请切换到患者端查看本人检查报告";
    return;
  }
  const bookingId = decode(String(query?.booking_id ?? ""));
  if (bookingId) void loadReport(bookingId);
  else errorMessage.value = "缺少预约编号";
});

async function loadReport(bookingId: string) {
  loading.value = true;
  try {
    report.value = await patientAppointmentApi.getMyReport(bookingId);
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "报告加载失败";
  } finally {
    loading.value = false;
  }
}
function decode(value: string) { try { return decodeURIComponent(value); } catch { return ""; } }

function formatReportDateTime(value?: string) {
  if (!value) return "—";
  const match = value.trim().match(/^(\d{4}-\d{2}-\d{2})[T\s](\d{2}:\d{2})/);
  return match ? `${match[1]} ${match[2]}` : value;
}
</script>

<template>
  <view class="report-page">
    <view v-if="loading" class="state">正在加载报告…</view>
    <view v-else-if="errorMessage" class="state error">{{ errorMessage }}</view>
    <view v-else-if="report" class="report-paper">
      <view class="document-heading">
        <text class="document-title">检查报告</text>
        <text class="document-status">{{ report.currentVersion.versionKind === 'correction' ? '已更正' : '已发布' }}</text>
      </view>
      <view class="document-accent" />

      <view class="information-table">
        <view class="information-row">
          <view class="information-cell">
            <text class="information-label">检查项目</text>
            <text class="information-value">{{ report.itemName }}</text>
          </view>
          <view class="information-cell">
            <text class="information-label">患者</text>
            <text class="information-value">{{ report.patientDisplayName }}</text>
          </view>
        </view>
        <view class="information-row">
          <view class="information-cell">
            <text class="information-label">手机号</text>
            <text class="information-value">{{ report.patientPhoneMasked }}</text>
          </view>
          <view class="information-cell">
            <text class="information-label">检查科室</text>
            <text class="information-value">{{ report.departmentName }}</text>
          </view>
        </view>
        <view class="information-row">
          <view class="information-cell">
            <text class="information-label">院区</text>
            <text class="information-value">{{ report.campusName }}</text>
          </view>
          <view class="information-cell">
            <text class="information-label">执行人员</text>
            <text class="information-value">{{ report.performedByDisplayName || '工作人员' }}</text>
          </view>
        </view>
        <view class="information-row information-row--full">
          <view class="information-cell">
            <text class="information-label">检查地点</text>
            <text class="information-value">{{ report.building }} · {{ report.floorNumber }}层 · {{ report.roomNumber }}室</text>
          </view>
        </view>
        <view class="information-row information-row--full">
          <view class="information-cell">
            <text class="information-label">完成时间</text>
            <text class="information-value">{{ formatReportDateTime(report.examinationCompletedAt) }}</text>
          </view>
        </view>
      </view>

      <view class="report-section">
        <text class="section-title">客观所见</text>
        <text class="section-content">{{ report.currentVersion.objectiveFindings || '—' }}</text>
      </view>

      <view class="report-section report-section--conclusion">
        <text class="section-title section-title--accent">检查结论</text>
        <text class="section-content section-content--important">{{ report.currentVersion.impression || '—' }}</text>
      </view>

      <view class="report-section">
        <text class="section-title">建议</text>
        <text class="section-content">{{ report.currentVersion.recommendation || '—' }}</text>
      </view>

      <view class="report-section">
        <text class="section-title">备注</text>
        <text class="section-content">{{ report.currentVersion.notes || '—' }}</text>
      </view>

      <view v-if="report.currentVersion.correctionReason" class="correction-reason">
        <text class="section-title">更正原因</text>
        <text class="section-content">{{ report.currentVersion.correctionReason }}</text>
      </view>

      <view class="publication-block">
        <view class="publication-information">
          <text>报告版本：第 {{ report.currentVersion.versionNo }} 版</text>
          <text>发布人员：{{ report.currentVersion.publishedByDisplayName || '工作人员' }}</text>
          <text>发布时间：{{ formatReportDateTime(report.currentVersion.publishedAt || report.updatedAt) }}</text>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.report-page {
  min-height: 100vh;
  padding: 24rpx;
  box-sizing: border-box;
  background: #eef1f4;
}

.report-paper {
  display: flex;
  min-height: 1000rpx;
  flex-direction: column;
  padding: 48rpx 46rpx 40rpx;
  box-sizing: border-box;
  color: #172230;
  background: #fff;
  border: 1rpx solid #d1d7de;
  border-radius: 2rpx;
}

.document-heading {
  position: relative;
  min-height: 54rpx;
}

.document-title {
  display: block;
  color: #172230;
  font-size: 36rpx;
  font-weight: 700;
  line-height: 54rpx;
  text-align: center;
}

.document-status {
  position: absolute;
  top: 10rpx;
  right: 0;
  color: #2d835f;
  font-size: 20rpx;
  line-height: 34rpx;
}

.document-accent {
  width: 100%;
  height: 3rpx;
  margin-top: 12rpx;
  background: #246fa8;
}

.information-table {
  margin-top: 30rpx;
  border-top: 1rpx solid #d1d7de;
}

.information-row {
  display: grid;
  min-height: 70rpx;
  grid-template-columns: 1fr 1fr;
  border-bottom: 1rpx solid #d1d7de;
}

.information-row--full {
  grid-template-columns: 1fr;
}

.information-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  padding: 14rpx 12rpx 14rpx 0;
  box-sizing: border-box;
}

.information-row:not(.information-row--full) .information-cell:first-child {
  margin-right: 22rpx;
}

.information-label,
.information-value {
  display: block;
  line-height: 1.45;
}

.information-label {
  flex: 0 0 104rpx;
  color: #667180;
  font-size: 19rpx;
}

.information-value {
  min-width: 0;
  flex: 1;
  color: #172230;
  font-size: 21rpx;
  font-weight: 600;
  word-break: break-all;
}

.report-section {
  padding: 28rpx 0 26rpx;
  border-bottom: 1rpx solid #d1d7de;
}

.section-title,
.section-content {
  display: block;
}

.section-title {
  color: #172230;
  font-size: 24rpx;
  font-weight: 700;
  line-height: 1.5;
}

.section-title--accent {
  color: #246fa8;
}

.section-content {
  margin-top: 12rpx;
  color: #172230;
  font-size: 24rpx;
  line-height: 1.8;
  white-space: pre-wrap;
}

.report-section--conclusion {
  margin-top: 0;
  padding: 24rpx 26rpx;
  background: #eff7fd;
}

.section-content--important {
  font-weight: 600;
}

.correction-reason {
  padding: 24rpx 0;
  border-bottom: 1rpx solid #d1d7de;
}

.publication-block {
  margin-top: auto;
  padding-top: 30rpx;
  border-top: 1rpx solid #d1d7de;
}

.publication-information {
  width: 340rpx;
  margin-left: auto;
}

.publication-information text {
  display: block;
  margin-top: 8rpx;
  color: #667180;
  font-size: 19rpx;
  line-height: 1.5;
}

.publication-information text:first-child {
  margin-top: 0;
}

.state {
  padding: 36rpx 24rpx;
  color: #8490a0;
  font-size: 22rpx;
  text-align: center;
  background: #fff;
  border: 1rpx solid #dfe4ea;
  border-radius: 12rpx;
}

.error {
  color: #be4e5d;
}
</style>
