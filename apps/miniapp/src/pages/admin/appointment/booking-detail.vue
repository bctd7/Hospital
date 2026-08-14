<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { ApiError } from "@/api/client";
import { appointmentManagementApi, staffBookingApi } from "@/api/appointment";
import { sessionState } from "@/stores/session";
import type { ExaminationReport, ExaminationReportContent, PatientBooking, PatientBookingStatus } from "@/types/appointment";
import { examinationWindowRelation, hasPermission } from "@/utils/appointmentManagement";

const bookingId = ref("");
const booking = ref<PatientBooking>();
const report = ref<ExaminationReport>();
const loading = ref(false);
const saving = ref(false);
const errorMessage = ref("");
const correctionMode = ref(false);
const correctionReason = ref("");
const objectiveFindings = ref("");
const impression = ref("");
const recommendation = ref("");
const notes = ref("");

const canPublish = computed(() => hasPermission(sessionState.principal, "report.publish"));
const canCorrect = computed(() => hasPermission(sessionState.principal, "report.correct"));
const editable = computed(() => booking.value?.status === "in_progress" || correctionMode.value);
const showBottomActions = computed(() => booking.value?.status === "confirmed"
  || booking.value?.status === "in_progress"
  || (booking.value?.status === "completed" && canCorrect.value));
const examinationWindowState = computed(() => booking.value
  ? examinationWindowRelation(booking.value.serviceDate, booking.value.itemStartTime, booking.value.itemEndTime)
  : "invalid");

onLoad((query) => {
  bookingId.value = decode(String(query?.booking_id ?? ""));
  if (!bookingId.value) {
    errorMessage.value = "缺少预约编号";
    return;
  }
  void loadDetail();
});

async function loadDetail() {
  if (loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    booking.value = await staffBookingApi.getBooking(bookingId.value);
    report.value = undefined;
    clearContent();
    if (booking.value.status === "in_progress" || booking.value.status === "completed") {
      try {
        report.value = await staffBookingApi.getReport(booking.value.bookingId);
        fillContent(report.value.currentVersion);
      } catch (error) {
        if (!(error instanceof ApiError) || error.statusCode !== 404 || booking.value.status === "completed") throw error;
        const template = await appointmentManagementApi.getItemReportTemplate(booking.value.itemId);
        fillContent(template);
      }
    }
  } catch (error) {
    errorMessage.value = messageOf(error, "预约详情加载失败");
  } finally {
    loading.value = false;
  }
}

async function startExamination() {
  if (!booking.value || booking.value.status !== "confirmed" || saving.value) return;
  if (examinationWindowState.value !== "open") {
    uni.showModal({
      title: "暂不能开始检查",
      content: examinationWindowDescription(booking.value),
      showCancel: false,
    });
    return;
  }
  await runMutation("检查已开始", async () => {
    booking.value = await staffBookingApi.startExamination(booking.value!);
    const template = await appointmentManagementApi.getItemReportTemplate(booking.value.itemId);
    fillContent(template);
  });
}

async function saveDraft() {
  if (!booking.value || booking.value.status !== "in_progress" || saving.value) return;
  if (!hasAnyContent()) {
    uni.showToast({ title: "请至少填写一项报告内容", icon: "none" });
    return;
  }
  await runMutation("草稿已保存", async () => {
    report.value = await staffBookingApi.saveReportDraft(booking.value!.bookingId, currentContent(), report.value?.version ?? 0);
    fillContent(report.value.currentVersion);
  });
}

function requestPublish() {
  if (!booking.value || booking.value.status !== "in_progress" || !canPublish.value) return;
  if (!objectiveFindings.value.trim() || !impression.value.trim()) {
    uni.showToast({ title: "发布前必须填写客观所见和检查结论", icon: "none" });
    return;
  }
  uni.showModal({
    title: "完成检查并发布报告",
    content: "发布后将完成本次检查并释放容量。原版不可覆盖，后续只能通过更正生成新版本。",
    confirmText: "确认发布",
    success: ({ confirm }) => { if (confirm) void publish(); },
  });
}

async function publish() {
  if (!booking.value) return;
  await runMutation("报告已发布", async () => {
    report.value = await staffBookingApi.completeAndPublishReport(booking.value!, currentContent(), report.value?.version ?? 0);
    booking.value = await staffBookingApi.getBooking(booking.value!.bookingId);
    fillContent(report.value.currentVersion);
  });
}

function beginCorrection() {
  if (!report.value || !canCorrect.value) return;
  correctionMode.value = true;
  correctionReason.value = "";
  fillContent(report.value.currentVersion);
}

async function submitCorrection() {
  if (!report.value || saving.value) return;
  if (!objectiveFindings.value.trim() || !impression.value.trim() || !correctionReason.value.trim()) {
    uni.showToast({ title: "请填写客观所见、检查结论和更正原因", icon: "none" });
    return;
  }
  await runMutation("更正版已发布", async () => {
    report.value = await staffBookingApi.correctReport(report.value!, currentContent(), correctionReason.value.trim());
    correctionMode.value = false;
    fillContent(report.value.currentVersion);
  });
}

function openHistory() {
  if (!report.value) return;
  uni.navigateTo({ url: `/pages/admin/appointment/report-history?report_id=${encodeURIComponent(report.value.reportId)}` });
}

async function runMutation(successTitle: string, action: () => Promise<void>) {
  saving.value = true;
  try {
    await action();
    uni.showToast({ title: successTitle, icon: "success" });
  } catch (error) {
    uni.showModal({ title: "操作失败", content: messageOf(error, "请刷新后重试"), showCancel: false });
  } finally {
    saving.value = false;
  }
}

function currentContent(): ExaminationReportContent {
  return { objectiveFindings: objectiveFindings.value.trim(), impression: impression.value.trim(), recommendation: recommendation.value.trim(), notes: notes.value.trim() };
}
function hasAnyContent() { return Object.values(currentContent()).some(Boolean); }
function fillContent(value: ExaminationReportContent) {
  objectiveFindings.value = value.objectiveFindings;
  impression.value = value.impression;
  recommendation.value = value.recommendation;
  notes.value = value.notes;
}
function clearContent() { fillContent({ objectiveFindings: "", impression: "", recommendation: "", notes: "" }); }
function decode(value: string) { try { return decodeURIComponent(value); } catch { return ""; } }
function messageOf(error: unknown, fallback: string) { return error instanceof Error && error.message.trim() ? error.message : fallback; }
function sessionLabel(value: string) { return value === "morning" ? "上午" : "下午"; }
function statusLabel(value: PatientBookingStatus) { return ({ confirmed: "待检查", in_progress: "检查中", completed: "已完成", no_show: "未到场" } as const)[value]; }
function examinationWindowDescription(value: PatientBooking) { return `只能在 ${value.serviceDate} ${clockLabel(value.itemStartTime)}–${clockLabel(value.itemEndTime)} 内开始检查。`; }
function clockLabel(value: string) { return value.slice(0, 5); }
</script>

<template>
  <view class="detail-page">
    <view v-if="loading" class="state">正在加载预约…</view>
    <view v-else-if="errorMessage" class="state state--error" @tap="loadDetail">{{ errorMessage }}</view>
    <template v-else-if="booking">
      <view class="card patient-card">
        <view><text class="patient-name">{{ booking.patientDisplayName }}</text><text class="phone">{{ booking.patientPhoneMasked }}</text></view>
        <text class="status" :class="`status--${booking.status}`">{{ statusLabel(booking.status) }}</text>
      </view>

      <view class="card info-card">
        <text class="project-name">{{ booking.itemName }}</text>
        <text>{{ booking.serviceDate }} · {{ sessionLabel(booking.session) }} · {{ booking.itemStartTime }}–{{ booking.itemEndTime }}</text>
        <text>{{ booking.departmentName || '当前科室' }} · {{ booking.roomDisplayName }}</text>
        <text>{{ booking.campusName ? `${booking.campusName} · ` : '' }}{{ booking.building }} · {{ booking.floorNumber }}层 · {{ booking.roomNumber }}室</text>
        <text v-if="booking.status === 'confirmed' && examinationWindowState !== 'open'" class="arrival-notice">只能在 {{ booking.serviceDate }} {{ clockLabel(booking.itemStartTime) }}–{{ clockLabel(booking.itemEndTime) }} 内开始检查</text>
        <text v-if="booking.status === 'no_show'" class="arrival-notice">项目结束时仍未开始检查，系统已记录为未到场。</text>
      </view>

      <view v-if="booking.status === 'in_progress' || booking.status === 'completed'" class="card report-card">
        <view class="card-heading">
          <text>{{ correctionMode ? '填写更正版' : '检查报告' }}</text>
          <button v-if="report" @tap="openHistory">版本记录</button>
        </view>
        <text v-if="report" class="report-meta">当前第 {{ report.currentVersion.versionNo }} 版 · {{ report.status === 'draft' ? '草稿' : report.currentVersion.versionKind === 'correction' ? '已更正' : '已发布' }}</text>
        <text class="field-label">客观所见</text><textarea v-model="objectiveFindings" :disabled="!editable" class="field" maxlength="8000" placeholder="记录客观检查所见" />
        <text class="field-label">检查结论</text><textarea v-model="impression" :disabled="!editable" class="field" maxlength="4000" placeholder="填写检查结论" />
        <text class="field-label">建议</text><textarea v-model="recommendation" :disabled="!editable" class="field short" maxlength="4000" placeholder="可选" />
        <text class="field-label">备注</text><textarea v-model="notes" :disabled="!editable" class="field short" maxlength="4000" placeholder="可选" />
        <template v-if="correctionMode">
          <text class="field-label">更正原因</text><textarea v-model="correctionReason" class="field short" maxlength="1000" placeholder="必填，将随新版本永久保留" />
        </template>
        <view v-if="booking.status === 'completed' && report" class="snapshot">
          <text>执行人员：{{ report.performedByDisplayName || '工作人员' }}</text>
          <text>发布人员：{{ report.currentVersion.publishedByDisplayName || '工作人员' }}</text>
          <text>发布于：{{ report.currentVersion.publishedAt || report.updatedAt }}</text>
        </view>
      </view>

      <view v-if="showBottomActions" class="bottom-actions">
        <button v-if="booking.status === 'confirmed'" :disabled="saving" class="primary" @tap="startExamination">{{ examinationWindowState === 'open' ? '开始检查' : '查看可开始时间' }}</button>
        <template v-else-if="booking.status === 'in_progress'">
          <button :disabled="saving" class="secondary" @tap="saveDraft">保存草稿</button>
          <button v-if="canPublish" :disabled="saving" class="primary" @tap="requestPublish">完成并发布</button>
        </template>
        <template v-else-if="booking.status === 'completed' && canCorrect">
          <button v-if="!correctionMode" :disabled="saving" class="primary" @tap="beginCorrection">发起更正</button>
          <template v-else><button class="secondary" @tap="correctionMode = false">取消</button><button :disabled="saving" class="primary" @tap="submitCorrection">发布更正版</button></template>
        </template>
      </view>
    </template>
  </view>
</template>

<style scoped>
button::after{display:none}.detail-page{min-height:100vh;padding:22rpx 22rpx 170rpx;box-sizing:border-box;background:#f3f6f9}.card,.state{margin-bottom:18rpx;padding:24rpx;background:#fff;border:1rpx solid #e2e8ee;border-radius:20rpx}.patient-card,.card-heading{display:flex;align-items:center;justify-content:space-between}.patient-name{color:#263348;font-size:30rpx;font-weight:720}.phone{margin-left:14rpx;color:#7d899a;font-size:21rpx}.status{padding:6rpx 13rpx;color:#247bb4;font-size:19rpx;background:#eaf4fb;border-radius:15rpx}.status--in_progress{color:#147cb1;background:#e6f5fc}.status--completed{color:#287b5e;background:#eaf7f1}.status--no_show{color:#8a5b35;background:#f8efe6}.info-card text{display:block;margin-top:10rpx;color:#718095;font-size:22rpx}.info-card .project-name{margin-top:0;color:#2f3d52;font-size:27rpx;font-weight:680}.card-heading{color:#2c3a4e;font-size:27rpx;font-weight:700}.card-heading button{width:auto;margin:0;padding:0 18rpx;color:#187fbd;font-size:20rpx;line-height:50rpx;background:#eaf5fb;border-radius:25rpx}.report-meta{display:block;margin-top:10rpx;color:#728197;font-size:20rpx}.field-label{display:block;margin-top:22rpx;color:#5f6e82;font-size:21rpx}.field{width:100%;height:180rpx;margin-top:9rpx;padding:18rpx;box-sizing:border-box;color:#2c394d;font-size:23rpx;background:#f5f7f9;border:1rpx solid #e3e8ed;border-radius:15rpx}.field.short{height:130rpx}.field[disabled]{color:#47566a;background:#f8f9fa}.snapshot{margin-top:22rpx;padding:18rpx;background:#f3f7fa;border-radius:15rpx}.snapshot text{display:block;margin-top:7rpx;color:#718095;font-size:20rpx}.snapshot text:first-child{margin-top:0}.state{color:#8490a0;font-size:22rpx;text-align:center}.state--error{color:#be4e5d}.bottom-actions{position:fixed;right:0;bottom:0;left:0;z-index:10;display:flex;gap:16rpx;padding:18rpx 22rpx calc(18rpx + env(safe-area-inset-bottom));background:#fff;border-top:1rpx solid #e2e8ee}.bottom-actions button{flex:1;margin:0;font-size:24rpx;line-height:76rpx;border-radius:38rpx}.bottom-actions .primary{color:#fff;background:#1688ce}.bottom-actions .secondary{color:#217daf;background:#eaf5fb}.bottom-actions button[disabled]{opacity:.55}
</style>
