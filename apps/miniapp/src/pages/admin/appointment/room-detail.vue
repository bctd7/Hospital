<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { appointmentManagementApi } from "@/api/appointment";
import {
  invalidateRoomAppointment,
  loadRoomExaminationItems,
  loadRoomWeeklyWindows,
} from "@/services/appointment";
import { sessionState } from "@/stores/session";
import type {
  AppointmentRoom,
  AppointmentSession,
  ExaminationItem,
  RoomExaminationItem,
  RoomWeeklyWindow,
} from "@/types/appointment";
import {
  canReadAppointmentManagement,
  hasPermission,
  messageOf,
  roomWindowTimeValid,
  SESSION_LABELS,
  WEEKDAY_LABELS,
} from "@/utils/appointmentManagement";

const departmentId = ref("");
const campusId = ref("");
const departmentLabel = ref("所属科室");
const roomId = ref("");
const room = ref<AppointmentRoom>();
const building = ref("");
const floorNumber = ref("");
const roomNumber = ref("");
const relations = ref<RoomExaminationItem[]>([]);
const windows = ref<RoomWeeklyWindow[]>([]);
const relationStatus = ref<"active" | "disabled">("active");
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const editorVisible = ref(false);
const editingWindow = ref<RoomWeeklyWindow>();
const windowWeekday = ref(1);
const windowSession = ref<AppointmentSession>("morning");
const openTime = ref("08:00");
const closeTime = ref("12:00");
const activeCapacity = ref("20");

const canCreate = computed(() => hasPermission(sessionState.principal, "appointment.create"));
const canUpdate = computed(() => hasPermission(sessionState.principal, "appointment.update"));
const isNew = computed(() => !roomId.value);
const sortedWindows = computed(() => [...windows.value].sort((a, b) =>
  a.weekday - b.weekday || (a.session === "morning" ? -1 : 1),
));

onLoad((options) => {
  if (!canReadAppointmentManagement(sessionState.principal)) {
    uni.showToast({ title: "无预约资源管理权限", icon: "none" });
    setTimeout(() => uni.navigateBack(), 800);
    return;
  }
  departmentId.value = decodeURIComponent(String(options?.department_id ?? ""));
  campusId.value = decodeURIComponent(String(options?.campus_id ?? ""));
  departmentLabel.value = decodeURIComponent(String(options?.department_label ?? "所属科室"));
  roomId.value = decodeURIComponent(String(options?.room_id ?? ""));
  uni.setNavigationBarTitle({ title: roomId.value ? "房间资源详情" : "新建房间" });
  if (roomId.value) void loadDetail();
});

async function loadDetail(force = false) {
  if (!roomId.value || loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    const [detail, linked, configured] = await Promise.all([
      appointmentManagementApi.getRoom(roomId.value),
      loadRoomExaminationItems(roomId.value, relationStatus.value, force),
      loadRoomWeeklyWindows(roomId.value, force),
    ]);
    room.value = detail;
    departmentId.value = detail.departmentId;
    campusId.value = detail.campusId;
    building.value = detail.building;
    floorNumber.value = String(detail.floorNumber);
    roomNumber.value = detail.roomNumber;
    relations.value = linked.items;
    windows.value = configured;
  } catch (cause) {
    error.value = messageOf(cause, "房间资源加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function saveRoom() {
  const normalizedBuilding = building.value.trim();
  const normalizedRoomNumber = roomNumber.value.trim();
  const floor = Number(floorNumber.value);
  if (!campusId.value || !normalizedBuilding || !normalizedRoomNumber || !Number.isInteger(floor) || floor === 0 || floor < -9 || floor > 99 || !/^[\p{L}\p{N}_-]+$/u.test(normalizedRoomNumber)) {
    uni.showToast({ title: "请完整填写楼栋、楼层和房间号", icon: "none" });
    return;
  }
  if (!departmentId.value || saving.value) return;
  saving.value = true;
  try {
    const saved = room.value
      ? await appointmentManagementApi.updateRoom(room.value, { campusId: campusId.value, building: normalizedBuilding, floorNumber: floor, roomNumber: normalizedRoomNumber })
      : await appointmentManagementApi.createRoom(departmentId.value, { campusId: campusId.value, building: normalizedBuilding, floorNumber: floor, roomNumber: normalizedRoomNumber });
    room.value = saved;
    roomId.value = saved.roomId;
    building.value = saved.building;
    floorNumber.value = String(saved.floorNumber);
    roomNumber.value = saved.roomNumber;
    invalidateRoomAppointment(saved.roomId, saved.departmentId);
    uni.setNavigationBarTitle({ title: "房间资源详情" });
    uni.showToast({ title: "房间已保存" });
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "房间保存失败"), icon: "none" });
  } finally {
    saving.value = false;
  }
}

function retireRoom() {
  if (!room.value || saving.value) return;
  uni.showModal({
    title: "不再使用此房间",
    content: `“${room.value.displayName}”将从正常管理和患者选择中永久隐藏。`,
    confirmText: "确认不再使用",
    confirmColor: "#d34d60",
    success: async ({ confirm }) => {
      if (!confirm || !room.value) return;
      saving.value = true;
      try {
        await appointmentManagementApi.retireRoom(room.value);
        invalidateRoomAppointment(room.value.roomId, room.value.departmentId);
        uni.navigateBack();
      } catch (cause) {
        uni.showToast({ title: messageOf(cause, "房间处理失败"), icon: "none" });
      } finally {
        saving.value = false;
      }
    },
  });
}

async function addItem() {
  if (!room.value || saving.value) return;
  saving.value = true;
  try {
    const [available, activeRelations, disabledRelations] = await Promise.all([
      appointmentManagementApi.listItems(room.value.departmentId, "active", 1, 100),
      appointmentManagementApi.listRoomItems(room.value.roomId, "active", 1, 100),
      appointmentManagementApi.listRoomItems(room.value.roomId, "disabled", 1, 100),
    ]);
    const existingIds = new Set(activeRelations.items.map((value) => value.itemId));
    const candidates = available.items.filter((value) => !existingIds.has(value.itemId));
    if (!candidates.length) {
      uni.showToast({ title: "当前科室没有可加入的新项目", icon: "none" });
      return;
    }
    chooseItemCandidate(candidates, disabledRelations.items);
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "项目列表加载失败"), icon: "none" });
  } finally {
    saving.value = false;
  }
}

function chooseItemCandidate(candidates: ExaminationItem[], disabledRelations: RoomExaminationItem[]) {
  uni.showActionSheet({
    title: "选择当前科室检查项目",
    itemList: candidates.map((value) => value.name),
    success: async ({ tapIndex }) => {
      const candidate = candidates[tapIndex];
      if (!candidate || !room.value) return;
      saving.value = true;
      try {
        const disabledRelation = disabledRelations.find((value) => value.itemId === candidate.itemId);
        if (disabledRelation) await appointmentManagementApi.setRoomItemEnabled(disabledRelation, true);
        else await appointmentManagementApi.addRoomItem(room.value.roomId, candidate.itemId);
        invalidateRoomAppointment(room.value.roomId, room.value.departmentId);
        relationStatus.value = "active";
        relations.value = (await loadRoomExaminationItems(room.value.roomId, "active", true)).items;
        uni.showToast({ title: "检查项目已加入" });
      } catch (cause) {
        uni.showToast({ title: messageOf(cause, "加入项目失败"), icon: "none" });
      } finally {
        saving.value = false;
      }
    },
  });
}

function changeRelation(value: RoomExaminationItem) {
  if (!room.value || saving.value) return;
  uni.showModal({
    title: "移出检查项目",
    content: `患者将不能再为“${value.itemName}”选择此房间。`,
    confirmText: "确认移出",
    success: async ({ confirm }) => {
      if (!confirm || !room.value) return;
      saving.value = true;
      try {
        await appointmentManagementApi.setRoomItemEnabled(value, false);
        invalidateRoomAppointment(room.value.roomId, room.value.departmentId);
        relations.value = (await loadRoomExaminationItems(room.value.roomId, relationStatus.value, true)).items;
      } catch (cause) {
        uni.showToast({ title: messageOf(cause, "关系状态修改失败"), icon: "none" });
      } finally {
        saving.value = false;
      }
    },
  });
}

function openWindowEditor(value?: RoomWeeklyWindow) {
  const fallback = firstAvailableSlot();
  editingWindow.value = value;
  windowWeekday.value = value?.weekday ?? fallback.weekday;
  windowSession.value = value?.session ?? fallback.session;
  openTime.value = value?.openTime ?? (windowSession.value === "morning" ? "08:00" : "14:00");
  closeTime.value = value?.closeTime ?? (windowSession.value === "morning" ? "12:00" : "17:00");
  activeCapacity.value = String(value?.activeCapacity ?? 20);
  editorVisible.value = true;
}

function firstAvailableSlot(): { weekday: number; session: AppointmentSession } {
  for (let weekday = 1; weekday <= 7; weekday += 1) {
    for (const session of ["morning", "afternoon"] as AppointmentSession[]) {
      if (!windows.value.some((value) => value.weekday === weekday && value.session === session)) return { weekday, session };
    }
  }
  return { weekday: 1, session: "morning" };
}

function selectWeekday(event: { detail: { value: string | number } }) { windowWeekday.value = Number(event.detail.value) + 1; }
function selectSession(event: { detail: { value: string | number } }) { windowSession.value = Number(event.detail.value) === 0 ? "morning" : "afternoon"; }

async function saveWindow() {
  if (!room.value || saving.value) return;
  const capacity = Number(activeCapacity.value);
  if (!roomWindowTimeValid(openTime.value, closeTime.value) || !Number.isInteger(capacity) || capacity < 1) {
    uni.showToast({ title: "请检查开放时间和共享容量", icon: "none" });
    return;
  }
  const duplicate = windows.value.find((value) => value.windowId !== editingWindow.value?.windowId && value.weekday === windowWeekday.value && value.session === windowSession.value);
  if (duplicate) {
    uni.showToast({ title: "该星期和时段已经存在配置", icon: "none" });
    return;
  }
  saving.value = true;
  try {
    await appointmentManagementApi.saveRoomWindow(room.value.roomId, {
      windowId: editingWindow.value?.windowId,
      weekday: windowWeekday.value,
      session: windowSession.value,
      openTime: openTime.value,
      closeTime: closeTime.value,
      activeCapacity: capacity,
      expectedVersion: editingWindow.value?.version ?? 0,
    });
    invalidateRoomAppointment(room.value.roomId, room.value.departmentId);
    windows.value = await loadRoomWeeklyWindows(room.value.roomId, true);
    editorVisible.value = false;
    uni.showToast({ title: "房间窗口已保存" });
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "窗口保存失败"), icon: "none" });
  } finally {
    saving.value = false;
  }
}

function disableWindow(value: RoomWeeklyWindow) {
  if (saving.value) return;
  uni.showModal({
    title: "关闭房间窗口",
    content: `${WEEKDAY_LABELS[value.weekday - 1]}${SESSION_LABELS[value.session]}将停止开放。关联项目配置不兼容时服务端会拒绝。`,
    confirmText: "确认关闭",
    confirmColor: "#d34d60",
    success: async ({ confirm }) => {
      if (!confirm || !room.value) return;
      saving.value = true;
      try {
        await appointmentManagementApi.disableRoomWindow(value);
        invalidateRoomAppointment(room.value.roomId, room.value.departmentId);
        windows.value = await loadRoomWeeklyWindows(room.value.roomId, true);
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
    <view class="context-card"><text class="context-card__label">所属科室</text><text class="context-card__value">{{ departmentLabel }}</text></view>
    <view v-if="error" class="state state--error"><text>{{ error }}</text><button @tap="loadDetail(true)">重新加载</button></view>
    <view v-else-if="loading" class="state">正在加载房间资源…</view>
    <template v-else>
      <view class="section">
        <view class="section__heading"><text>房间基本信息</text></view>
        <text class="field-label">院区</text><view class="readonly-field">{{ departmentLabel.split(' / ')[0] }}</view>
        <text class="field-label">楼栋</text><input v-model="building" class="field-input" maxlength="64" placeholder="例如：门诊楼" />
        <text class="field-label">楼层</text><input v-model="floorNumber" class="field-input" type="number" placeholder="例如：3；地下 1 层填 -1" />
        <text class="field-label">房间号</text><input v-model="roomNumber" class="field-input" maxlength="32" placeholder="例如：301 或 A301" />
        <button class="primary-button" :disabled="saving || (isNew ? !canCreate : !canUpdate)" @tap="saveRoom">{{ saving ? '保存中…' : isNew ? '创建房间' : '保存房间信息' }}</button>
        <button v-if="room && canUpdate" class="danger-button" :disabled="saving" @tap="retireRoom">不再使用此房间</button>
      </view>

      <view v-if="room" class="section">
        <view class="section__heading"><text>可执行检查项目</text><button v-if="canUpdate" @tap="addItem">＋ 加入</button></view>
        <view v-if="!relations.length" class="inline-empty">尚未加入检查项目</view>
        <view v-for="value in relations" :key="value.relationId" class="relation-row">
          <text class="relation-row__title">{{ value.itemName }}</text>
          <button v-if="canUpdate" class="text-danger" @tap="changeRelation(value)">移出</button>
        </view>
      </view>

      <view v-if="room" class="section">
        <view class="section__heading"><text>房间周开放窗口</text><button v-if="canUpdate" @tap="openWindowEditor()">＋ 配置</button></view>
        <view v-if="!sortedWindows.length" class="inline-empty">尚未配置开放窗口</view>
        <view v-for="value in sortedWindows" :key="value.windowId" class="window-row">
          <view><text class="window-row__title">{{ WEEKDAY_LABELS[value.weekday - 1] }} · {{ SESSION_LABELS[value.session] }}</text><text class="window-row__time">{{ value.openTime }}—{{ value.closeTime }} · 共享容量 {{ value.activeCapacity }}</text></view>
          <view class="window-row__actions"><text :class="`status status--${value.status}`">{{ value.status === 'active' ? '开放' : '关闭' }}</text><button v-if="canUpdate" @tap="openWindowEditor(value)">编辑</button><button v-if="canUpdate && value.status === 'active'" class="text-danger" @tap="disableWindow(value)">关闭</button></view>
        </view>
      </view>
    </template>

    <view v-if="editorVisible" class="dialog-mask" @tap.self="editorVisible = false">
      <view class="dialog">
        <text class="dialog__title">{{ editingWindow ? '编辑房间窗口' : '新增房间窗口' }}</text>
        <view class="picker-row"><picker :value="windowWeekday - 1" :range="WEEKDAY_LABELS" @change="selectWeekday"><view>{{ WEEKDAY_LABELS[windowWeekday - 1] }} ›</view></picker><picker :value="windowSession === 'morning' ? 0 : 1" :range="['上午', '下午']" @change="selectSession"><view>{{ SESSION_LABELS[windowSession] }} ›</view></picker></view>
        <text class="field-label">开放开始（HH:MM）</text><input v-model="openTime" class="field-input" placeholder="08:00" />
        <text class="field-label">开放结束（HH:MM）</text><input v-model="closeTime" class="field-input" placeholder="12:00" />
        <text class="field-label">共享活动容量</text><input v-model="activeCapacity" class="field-input" type="number" placeholder="20" />
        <view class="dialog__buttons"><button @tap="editorVisible = false">取消</button><button class="primary-button" :disabled="saving" @tap="saveWindow">保存</button></view>
      </view>
    </view>
  </view>
</template>

<style scoped>
.readonly-field{width:100%;height:72rpx;margin-top:10rpx;padding:0 20rpx;box-sizing:border-box;color:#536177;font-size:24rpx;line-height:72rpx;background:#eef2f6;border:1rpx solid #e1e7ed;border-radius:16rpx}
button::after{display:none}.detail-page{min-height:100vh;padding:24rpx;box-sizing:border-box;background:#f2f6fa}.context-card,.section{padding:26rpx;background:#fff;border:1rpx solid #e6ecf2;border-radius:24rpx}.context-card__label,.context-card__value{display:block}.context-card__label{color:#99a3b2;font-size:20rpx}.context-card__value{margin-top:7rpx;color:#344157;font-size:26rpx;font-weight:680}.section{margin-top:20rpx}.section__heading{display:flex;align-items:center;justify-content:space-between;color:#273449;font-size:28rpx;font-weight:700}.section__heading button,.relation-row button,.window-row button,.sub-toolbar button{width:auto;margin:0;padding:0 16rpx;color:#1684ca;font-size:20rpx;line-height:50rpx;background:#e9f5fc;border-radius:25rpx}.section__hint{display:block;margin-top:12rpx;color:#8d98a8;font-size:20rpx;line-height:1.6}.field-label{display:block;margin-top:22rpx;color:#657287;font-size:21rpx}.field-input{width:100%;height:72rpx;margin-top:10rpx;padding:0 20rpx;box-sizing:border-box;color:#28364a;font-size:24rpx;background:#f6f8fa;border:1rpx solid #e4eaf0;border-radius:16rpx}.primary-button,.danger-button{width:100%;margin:22rpx 0 0;font-size:24rpx;line-height:72rpx;border-radius:36rpx}.primary-button{color:#fff;background:linear-gradient(135deg,#168bd7,#1db4b2)}.danger-button{color:#c94b5e;background:#fbecef}.status{padding:5rpx 11rpx;font-size:18rpx;border-radius:13rpx}.status--active{color:#138766;background:#e1f7ef}.status--disabled{color:#a0616b;background:#f7e8eb}.sub-toolbar{display:flex;align-items:center;justify-content:space-between;margin-top:14rpx;color:#8d98a8;font-size:19rpx}.inline-empty,.state{margin-top:18rpx;padding:48rpx 20rpx;color:#8d98a8;font-size:22rpx;text-align:center;background:#f7f9fb;border-radius:18rpx}.state{display:flex;flex-direction:column;gap:18rpx}.state--error{color:#c44f61}.state button{margin:auto;color:#1684ca;background:#e9f5fc}.relation-row,.window-row{display:flex;align-items:center;justify-content:space-between;gap:18rpx;margin-top:14rpx;padding:18rpx;background:#f7f9fb;border-radius:18rpx}.relation-row__title,.relation-row__hint,.window-row__title,.window-row__time{display:block}.relation-row__title,.window-row__title{color:#344157;font-size:23rpx;font-weight:650}.relation-row__hint,.window-row__time{margin-top:6rpx;color:#8390a3;font-size:19rpx}.text-danger,.relation-row button.text-danger,.window-row__actions button.text-danger{color:#c94b5e;background:#fbecef}.window-row__actions{display:flex;align-items:center;gap:8rpx}.dialog-mask{position:fixed;inset:0;display:flex;align-items:flex-end;z-index:20;background:rgba(19,29,43,.48)}.dialog{width:100%;padding:30rpx 26rpx calc(30rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#fff;border-radius:30rpx 30rpx 0 0}.dialog__title{color:#263348;font-size:30rpx;font-weight:720}.picker-row{display:grid;grid-template-columns:1fr 1fr;gap:14rpx;margin-top:22rpx}.picker-row view{padding:0 20rpx;color:#425067;font-size:23rpx;line-height:68rpx;background:#f4f7fa;border-radius:16rpx}.dialog__notice{display:block;margin-top:20rpx;color:#bb7838;font-size:20rpx}.dialog__buttons{display:grid;grid-template-columns:1fr 1fr;gap:16rpx;margin-top:20rpx}.dialog__buttons button{margin:0;line-height:68rpx;border-radius:34rpx}.dialog__buttons .primary-button{margin:0}.primary-button[disabled]{opacity:.55}
</style>
