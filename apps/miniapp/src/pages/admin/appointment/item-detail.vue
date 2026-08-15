<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { appointmentManagementApi } from "@/api/appointment";
import { ApiError } from "@/api/client";
import {
  invalidateItemAppointment,
  loadItemWeeklyWindows,
} from "@/services/appointment";
import { sessionState } from "@/stores/session";
import type {
  AppointmentSession,
  ExaminationItem,
  ExaminationItemReportTemplate,
  ItemWeeklyWindow,
} from "@/types/appointment";
import {
  canReadAppointmentManagement,
  hasPermission,
  itemWindowTimeValid,
  messageOf,
  SESSION_LABELS,
  WEEKDAY_LABELS,
  windowFitsSession,
} from "@/utils/appointmentManagement";

const departmentId = ref("");
const departmentLabel = ref("所属科室");
const itemId = ref("");
const item = ref<ExaminationItem>();
const name = ref("");
const description = ref("");
const estimatedDurationMinutes = ref(30);
const windows = ref<ItemWeeklyWindow[]>([]);
const reportTemplate = ref<ExaminationItemReportTemplate>();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const editorVisible = ref(false);
const editingWindow = ref<ItemWeeklyWindow>();
const windowWeekday = ref(1);
const windowSession = ref<AppointmentSession>("morning");
const startTime = ref("09:00");
const cutoffTime = ref("11:30");
const endTime = ref("12:00");
const durationOptions = Array.from({ length: 96 }, (_, index) => (index + 1) * 5);
const durationLabels = durationOptions.map((minutes) => {
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  if (!hours) return `${minutes} 分钟`;
  if (!remainder) return `${hours} 小时`;
  return `${hours} 小时 ${remainder} 分钟`;
});

const canCreate = computed(() => hasPermission(sessionState.principal, "appointment.create"));
const canUpdate = computed(() => hasPermission(sessionState.principal, "appointment.update"));
const isNew = computed(() => !itemId.value);
const durationIndex = computed(() => {
  const index = durationOptions.indexOf(estimatedDurationMinutes.value);
  return index >= 0 ? index : durationOptions.indexOf(30);
});
const sortedWindows = computed(() => [...windows.value].sort((a, b) =>
  a.weekday - b.weekday || (a.session === "morning" ? -1 : 1),
));

onLoad((options) => {
  if (!canReadAppointmentManagement(sessionState.principal)) {
    uni.showToast({ title: "无检查项目管理权限", icon: "none" });
    setTimeout(() => uni.navigateBack(), 800);
    return;
  }
  departmentId.value = decodeURIComponent(String(options?.department_id ?? ""));
  departmentLabel.value = decodeURIComponent(String(options?.department_label ?? "所属科室"));
  itemId.value = decodeURIComponent(String(options?.item_id ?? ""));
  uni.setNavigationBarTitle({ title: itemId.value ? "检查项目详情" : "新建检查项目" });
  if (itemId.value) void loadDetail();
});

async function loadDetail(force = false) {
  if (!itemId.value || loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    const [detail, configured, configuredTemplate] = await Promise.all([
      appointmentManagementApi.getItem(itemId.value),
      loadItemWeeklyWindows(itemId.value, force),
      appointmentManagementApi.getItemReportTemplate(itemId.value),
    ]);
    item.value = detail;
    departmentId.value = detail.ownerDepartmentId;
    name.value = detail.name;
    description.value = detail.description;
    estimatedDurationMinutes.value = detail.estimatedDurationMinutes;
    windows.value = configured;
    reportTemplate.value = configuredTemplate;
  } catch (cause) {
    error.value = messageOf(cause, "检查项目加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function saveReportTemplate() {
  if (!reportTemplate.value || saving.value || !canUpdate.value) return;
  saving.value = true;
  try {
    reportTemplate.value = await appointmentManagementApi.saveItemReportTemplate({
      ...reportTemplate.value,
      objectiveFindings: reportTemplate.value.objectiveFindings.trim(),
      impression: reportTemplate.value.impression.trim(),
      recommendation: reportTemplate.value.recommendation.trim(),
      notes: reportTemplate.value.notes.trim(),
    });
    uni.showToast({ title: "报告模板已保存" });
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "报告模板保存失败"), icon: "none" });
  } finally {
    saving.value = false;
  }
}

async function saveItem() {
  const normalizedName = name.value.trim();
  if (!normalizedName) {
    uni.showToast({ title: "请输入检查项目名称", icon: "none" });
    return;
  }
  if (!Number.isInteger(estimatedDurationMinutes.value) || estimatedDurationMinutes.value < 5 || estimatedDurationMinutes.value > 480 || estimatedDurationMinutes.value % 5 !== 0) {
    uni.showToast({ title: "预计时长须为 5–480 分钟且为 5 的倍数", icon: "none" });
    return;
  }
  if (!departmentId.value || saving.value) return;
  saving.value = true;
  try {
    const saved = item.value
      ? await appointmentManagementApi.updateItem(item.value, normalizedName, description.value.trim(), estimatedDurationMinutes.value)
      : await appointmentManagementApi.createItem(departmentId.value, normalizedName, description.value.trim(), estimatedDurationMinutes.value);
    item.value = saved;
    itemId.value = saved.itemId;
    name.value = saved.name;
    description.value = saved.description;
    estimatedDurationMinutes.value = saved.estimatedDurationMinutes;
    if (!reportTemplate.value) {
      reportTemplate.value = await appointmentManagementApi.getItemReportTemplate(saved.itemId);
    }
    invalidateItemAppointment(saved.itemId, saved.ownerDepartmentId);
    uni.setNavigationBarTitle({ title: "检查项目详情" });
    uni.showToast({ title: "项目已保存" });
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "项目保存失败"), icon: "none" });
  } finally {
    saving.value = false;
  }
}

function selectDuration(event: { detail: { value: string | number } }) {
  const selected = durationOptions[Number(event.detail.value)];
  if (selected) estimatedDurationMinutes.value = selected;
}

function changeItemStatus() {
  if (!item.value || saving.value) return;
  const enabled = item.value.status !== "active";
  uni.showModal({
    title: enabled ? "恢复检查项目" : "停用检查项目",
    content: enabled
      ? `恢复“${item.value.name}”后可以重新配置和关联房间。`
      : `停用“${item.value.name}”不会删除历史记录，患者端将不再展示。`,
    confirmText: enabled ? "确认恢复" : "确认停用",
    confirmColor: enabled ? "#1688ce" : "#d34d60",
    success: async ({ confirm }) => {
      if (!confirm || !item.value) return;
      saving.value = true;
      try {
        item.value = await appointmentManagementApi.setItemEnabled(item.value, enabled);
        invalidateItemAppointment(item.value.itemId, item.value.ownerDepartmentId);
        uni.showToast({ title: enabled ? "项目已恢复" : "项目已停用" });
      } catch (cause) {
        uni.showToast({ title: messageOf(cause, "状态修改失败"), icon: "none" });
      } finally {
        saving.value = false;
      }
    },
  });
}

function openWindowEditor(value?: ItemWeeklyWindow) {
  const fallback = firstAvailableSlot();
  editingWindow.value = value;
  windowWeekday.value = value?.weekday ?? fallback.weekday;
  windowSession.value = value?.session ?? fallback.session;
  if (value) {
    startTime.value = value.startTime;
    cutoffTime.value = value.bookingCutoffTime;
    endTime.value = value.endTime;
  } else {
    applySessionTimeDefaults();
  }
  editorVisible.value = true;
}

function firstAvailableSlot(): { weekday: number; session: AppointmentSession } {
  for (let weekday = 1; weekday <= 7; weekday += 1) {
    for (const session of ["morning", "afternoon"] as AppointmentSession[]) {
      if (!windows.value.some((value) => value.weekday === weekday && value.session === session)) {
        return { weekday, session };
      }
    }
  }
  return { weekday: 1, session: "morning" };
}

function selectWeekday(event: { detail: { value: string | number } }) {
  windowWeekday.value = Number(event.detail.value) + 1;
}

function selectSession(event: { detail: { value: string | number } }) {
  windowSession.value = Number(event.detail.value) === 0 ? "morning" : "afternoon";
  applySessionTimeDefaults();
}

function applySessionTimeDefaults() {
  if (windowSession.value === "morning") {
    startTime.value = "09:00";
    cutoffTime.value = "11:30";
    endTime.value = "12:00";
  } else {
    startTime.value = "13:00";
    cutoffTime.value = "17:30";
    endTime.value = "18:00";
  }
}

async function saveWindow() {
  if (!item.value || saving.value) return;
  if (!itemWindowTimeValid(startTime.value, cutoffTime.value, endTime.value)) {
    uni.showToast({ title: "请检查开始、停止新增和结束时间", icon: "none" });
    return;
  }
  if (!windowFitsSession(windowSession.value, startTime.value, endTime.value)) {
    uni.showModal({
      title: "时段与时间不一致",
      content: windowSession.value === "morning"
        ? "上午窗口必须完整设置在 12:00 以前，12:00 可以作为结束时间。"
        : "下午窗口必须从 12:00 或之后开始。",
      showCancel: false,
    });
    return;
  }
  const duplicate = windows.value.find((value) =>
    value.windowId !== editingWindow.value?.windowId &&
    value.weekday === windowWeekday.value && value.session === windowSession.value,
  );
  if (duplicate) {
    uni.showToast({ title: "该星期和时段已经存在配置", icon: "none" });
    return;
  }
  saving.value = true;
  try {
    await appointmentManagementApi.saveItemWindow(item.value.itemId, {
      windowId: editingWindow.value?.windowId,
      weekday: windowWeekday.value,
      session: windowSession.value,
      startTime: startTime.value,
      bookingCutoffTime: cutoffTime.value,
      endTime: endTime.value,
      expectedVersion: editingWindow.value?.version ?? 0,
    });
    invalidateItemAppointment(item.value.itemId, item.value.ownerDepartmentId);
    windows.value = await loadItemWeeklyWindows(item.value.itemId, true);
    editorVisible.value = false;
    uni.showToast({ title: "项目窗口已保存" });
  } catch (cause) {
    if (cause instanceof ApiError && cause.code === "ITEM_ROOM_WINDOW_CONFLICT") {
      uni.showModal({
        title: "项目时间超出房间开放范围",
        content: "项目预约时间必须完整落在所有已关联房间同一天、同时段的开放窗口内。请先扩大房间开放时间，或缩短项目预约时间。",
        showCancel: false,
      });
    } else {
      uni.showToast({ title: messageOf(cause, "窗口保存失败"), icon: "none" });
    }
  } finally {
    saving.value = false;
  }
}

function disableWindow(value: ItemWeeklyWindow) {
  if (saving.value) return;
  uni.showModal({
    title: "关闭项目窗口",
    content: `${WEEKDAY_LABELS[value.weekday - 1]}${SESSION_LABELS[value.session]}将不可预约。关联房间配置不兼容时服务端会拒绝。`,
    confirmText: "确认关闭",
    confirmColor: "#d34d60",
    success: async ({ confirm }) => {
      if (!confirm || !item.value) return;
      saving.value = true;
      try {
        await appointmentManagementApi.disableItemWindow(value);
        invalidateItemAppointment(item.value.itemId, item.value.ownerDepartmentId);
        windows.value = await loadItemWeeklyWindows(item.value.itemId, true);
      } catch (cause) {
        uni.showToast({ title: messageOf(cause, "关闭失败"), icon: "none" });
      } finally {
        saving.value = false;
      }
    },
  });
}
</script>

<template>
  <view class="detail-page">
    <view class="context-card">
      <text class="context-card__label">所属科室</text>
      <text class="context-card__value">{{ departmentLabel }}</text>
    </view>

    <view v-if="error" class="state state--error">
      <text>{{ error }}</text><button @tap="loadDetail(true)">重新加载</button>
    </view>
    <view v-else-if="loading" class="state">正在加载项目…</view>
    <template v-else>
      <view class="section">
        <view class="section__heading">
          <text>项目基本信息</text>
          <text v-if="item" class="status" :class="`status--${item.status}`">{{ item.status === 'active' ? '启用中' : '已停用' }}</text>
        </view>
        <text class="field-label">项目名称</text>
        <input v-model="name" class="field-input" maxlength="128" placeholder="例如：胸部 CT" />
        <text class="field-label">预计检查时长</text>
        <picker
          mode="selector"
          :value="durationIndex"
          :range="durationLabels"
          @change="selectDuration"
        >
          <view class="field-picker">
            <text>{{ durationLabels[durationIndex] }}</text>
            <text class="field-picker__arrow">›</text>
          </view>
        </picker>
        <text class="field-label">患者可见检查说明</text>
        <textarea v-model="description" class="field-textarea" maxlength="4000" placeholder="说明检查用途、准备事项等" />
        <button class="primary-button" :disabled="saving || (isNew ? !canCreate : !canUpdate)" @tap="saveItem">
          {{ saving ? '保存中…' : isNew ? '创建检查项目' : '保存项目信息' }}
        </button>
        <button v-if="item && canUpdate" class="danger-button" :disabled="saving" @tap="changeItemStatus">
          {{ item.status === 'active' ? '停用检查项目' : '恢复检查项目' }}
        </button>
      </view>

      <view v-if="item" class="section">
        <view class="section__heading">
          <text>检查报告模板</text>
          <text class="template-version">版本 {{ reportTemplate?.version ?? 0 }}</text>
        </view>
        <text class="section__hint">开始填写首份报告时可带入以下内容；已生成的报告不会随模板修改。</text>
        <template v-if="reportTemplate">
          <text class="field-label">客观所见</text>
          <textarea v-model="reportTemplate.objectiveFindings" class="field-textarea report-field" maxlength="8000" placeholder="预设客观检查所见，可留空" />
          <text class="field-label">检查结论</text>
          <textarea v-model="reportTemplate.impression" class="field-textarea report-field" maxlength="4000" placeholder="预设检查结论，可留空" />
          <text class="field-label">建议</text>
          <textarea v-model="reportTemplate.recommendation" class="field-textarea report-field" maxlength="4000" placeholder="预设后续建议，可留空" />
          <text class="field-label">备注</text>
          <textarea v-model="reportTemplate.notes" class="field-textarea report-field" maxlength="4000" placeholder="预设备注，可留空" />
          <button v-if="canUpdate" class="primary-button" :disabled="saving" @tap="saveReportTemplate">保存报告模板</button>
        </template>
      </view>

      <view v-if="item" class="section">
        <view class="section__heading"><text>项目周预约窗口</text><button v-if="canUpdate" @tap="openWindowEditor()">＋ 配置</button></view>
        <view v-if="!sortedWindows.length" class="inline-empty">尚未配置预约窗口</view>
        <view v-for="value in sortedWindows" :key="value.windowId" class="window-row">
          <view>
            <text class="window-row__title">{{ WEEKDAY_LABELS[value.weekday - 1] }} · {{ SESSION_LABELS[value.session] }}</text>
            <text class="window-row__time">{{ value.startTime }}—{{ value.endTime }} · {{ value.bookingCutoffTime }} 停止新增</text>
          </view>
          <view class="window-row__actions">
            <text :class="`status status--${value.status}`">{{ value.status === 'active' ? '启用' : '关闭' }}</text>
            <button v-if="canUpdate" @tap="openWindowEditor(value)">编辑</button>
            <button v-if="canUpdate && value.status === 'active'" class="text-danger" @tap="disableWindow(value)">关闭</button>
          </view>
        </view>
      </view>
    </template>

    <view v-if="editorVisible" class="dialog-mask">
      <view class="dialog">
        <text class="dialog__title">{{ editingWindow ? '编辑项目窗口' : '新增项目窗口' }}</text>
        <view class="picker-row">
          <picker :value="windowWeekday - 1" :range="WEEKDAY_LABELS" @change="selectWeekday"><view>{{ WEEKDAY_LABELS[windowWeekday - 1] }} ›</view></picker>
          <picker :value="windowSession === 'morning' ? 0 : 1" :range="['上午', '下午']" @change="selectSession"><view>{{ SESSION_LABELS[windowSession] }} ›</view></picker>
        </view>
        <text class="field-label">预约开始（HH:MM）</text><input v-model="startTime" class="field-input" placeholder="09:00" />
        <text class="field-label">停止新增（HH:MM）</text><input v-model="cutoffTime" class="field-input" placeholder="11:30" />
        <text class="field-label">预约结束（HH:MM）</text><input v-model="endTime" class="field-input" placeholder="12:00" />
        <view class="dialog__buttons"><button @tap="editorVisible = false">取消</button><button class="primary-button" :disabled="saving" @tap="saveWindow">保存</button></view>
      </view>
    </view>
  </view>
</template>

<style scoped>
button::after{display:none}.detail-page{min-height:100vh;padding:24rpx;box-sizing:border-box;background:#f2f6fa}.context-card,.section{padding:26rpx;background:#fff;border:1rpx solid #e6ecf2;border-radius:24rpx}.context-card__label,.context-card__value{display:block}.context-card__label{color:#99a3b2;font-size:20rpx}.context-card__value{margin-top:7rpx;color:#344157;font-size:26rpx;font-weight:680}.section{margin-top:20rpx}.section__heading{display:flex;align-items:center;justify-content:space-between;color:#273449;font-size:28rpx;font-weight:700}.section__heading button,.window-row button{width:auto;margin:0;padding:0 16rpx;color:#1684ca;font-size:21rpx;line-height:50rpx;background:#e9f5fc;border-radius:25rpx}.section__hint{display:block;margin-top:12rpx;color:#8d98a8;font-size:20rpx;line-height:1.6}.field-label{display:block;margin-top:22rpx;color:#657287;font-size:21rpx}.field-input,.field-textarea,.field-picker{width:100%;margin-top:10rpx;padding:0 20rpx;box-sizing:border-box;color:#28364a;font-size:24rpx;background:#f6f8fa;border:1rpx solid #e4eaf0;border-radius:16rpx}.field-input,.field-picker{height:72rpx}.field-picker{display:flex;align-items:center;justify-content:space-between}.field-picker__arrow{color:#8491a4;font-size:32rpx}.field-textarea{height:190rpx;padding-top:18rpx}.report-field{height:150rpx}.template-version{color:#77869a;font-size:20rpx;font-weight:500}.primary-button,.danger-button{width:100%;margin:22rpx 0 0;font-size:24rpx;line-height:72rpx;border-radius:36rpx}.primary-button{color:#fff;background:#168bd7}.danger-button{color:#c94b5e;background:#fbecef}.status{padding:5rpx 11rpx;font-size:18rpx;border-radius:13rpx}.status--active{color:#138766;background:#e1f7ef}.status--disabled{color:#a0616b;background:#f7e8eb}.inline-empty,.state{margin-top:18rpx;padding:48rpx 20rpx;color:#8d98a8;font-size:22rpx;text-align:center;background:#f7f9fb;border-radius:18rpx}.state{display:flex;flex-direction:column;gap:18rpx}.state--error{color:#c44f61}.state button{margin:auto;color:#1684ca;background:#e9f5fc}.window-row{display:flex;align-items:center;justify-content:space-between;gap:18rpx;margin-top:14rpx;padding:18rpx;background:#f7f9fb;border-radius:18rpx}.window-row__title,.window-row__time{display:block}.window-row__title{color:#344157;font-size:23rpx;font-weight:650}.window-row__time{margin-top:6rpx;color:#8390a3;font-size:19rpx}.window-row__actions{display:flex;align-items:center;gap:8rpx}.window-row__actions .text-danger{color:#c94b5e;background:#fbecef}.dialog-mask{position:fixed;inset:0;display:flex;align-items:flex-end;z-index:20;background:rgba(19,29,43,.48)}.dialog{width:100%;padding:30rpx 26rpx calc(30rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#fff;border-radius:30rpx 30rpx 0 0}.dialog__title{color:#263348;font-size:30rpx;font-weight:720}.picker-row{display:grid;grid-template-columns:1fr 1fr;gap:14rpx;margin-top:22rpx}.picker-row view{padding:0 20rpx;color:#425067;font-size:23rpx;line-height:68rpx;background:#f4f7fa;border-radius:16rpx}.dialog__notice{display:block;margin-top:20rpx;color:#bb7838;font-size:20rpx}.dialog__buttons{display:grid;grid-template-columns:1fr 1fr;gap:16rpx;margin-top:20rpx}.dialog__buttons button{margin:0;line-height:68rpx;border-radius:34rpx}.dialog__buttons .primary-button{margin:0}.primary-button[disabled]{opacity:.55}
</style>
