<script setup lang="ts">
import { onHide, onLoad, onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { appointmentManagementApi, staffBookingApi } from "@/api/appointment";
import { ApiError } from "@/api/client";
import { loadDepartmentOptions, type DepartmentOption } from "@/services/organization";
import type { AppointmentListView, AppointmentRoom, PatientBooking, PatientBookingStatus } from "@/types/appointment";
import { currentStaffDepartmentId, rememberStaffDepartmentId } from "@/utils/staffDepartmentContext";

const departmentId = ref("");
const departmentLabel = ref("科室预约");
const departmentLocked = ref(false);
const departmentOptions = ref<DepartmentOption[]>([]);
const rooms = ref<AppointmentRoom[]>([]);
const selectedRoomId = ref("");
const serviceDate = ref(today());
const patientKeyword = ref("");
const bookings = ref<PatientBooking[]>([]);
const noShowBookings = ref<PatientBooking[]>([]);
const loading = ref(false);
const pendingId = ref("");
const errorMessage = ref("");
const view = ref<AppointmentListView>("active");
const nowTick = ref(Date.now());
let clockTimer: ReturnType<typeof setInterval> | undefined;
let lastExpiredRefreshAt = 0;

const isCompletedView = computed(() => view.value === "completed");
const isToday = computed(() => serviceDate.value === today());
const canSwitchDepartment = computed(() => !departmentLocked.value && departmentOptions.value.length > 1);
const currentExamination = computed(() => bookings.value.find((value) => value.status === "in_progress"));
const currentCalled = computed(() => bookings.value.find((value) => value.status === "called"));
const currentBooking = computed(() => currentExamination.value ?? currentCalled.value);
const currentCallExpired = computed(() => Boolean(currentCalled.value?.callDeadline)
  && nowTick.value >= Date.parse(currentCalled.value!.callDeadline!));
const currentCallHint = computed(() => {
  const deadline = currentCalled.value?.callDeadline;
  if (!deadline) return "请患者立即前往检查房间，并留意现场广播";
  const remaining = Math.max(0, Math.ceil((Date.parse(deadline) - nowTick.value) / 1000));
  return remaining > 0 ? `叫号剩余 ${remaining} 秒，超时后自动返回候检队列` : "叫号已超时，正在更新候检队列…";
});
const queuedBookings = computed(() => bookings.value.filter((value) => value.status === "queued").sort((a, b) => a.queueNumber - b.queueNumber));
const confirmedBookings = computed(() => bookings.value.filter((value) => value.status === "confirmed"));

onLoad((query) => {
  view.value = query?.view === "completed" ? "completed" : "active";
  const queryDepartmentId = decode(query?.department_id);
  departmentId.value = queryDepartmentId || currentStaffDepartmentId();
  departmentLocked.value = Boolean(queryDepartmentId);
  departmentLabel.value = decode(query?.department_label) || (isCompletedView.value ? "检查记录" : "科室预约");
  uni.setNavigationBarTitle({ title: isCompletedView.value ? "检查记录" : "科室预约" });
  if (!departmentLocked.value) void prepareDepartmentSelection();
});
onShow(() => {
  nowTick.value = Date.now();
  if (clockTimer) clearInterval(clockTimer);
  clockTimer = setInterval(tickQueueClock, 1000);
  void loadPage();
});
onHide(() => { if (clockTimer) clearInterval(clockTimer); clockTimer = undefined; });

function tickQueueClock() {
  nowTick.value = Date.now();
  const deadline = currentCalled.value?.callDeadline;
  if (!deadline || nowTick.value < Date.parse(deadline) || nowTick.value - lastExpiredRefreshAt < 2000) return;
  lastExpiredRefreshAt = nowTick.value;
  void loadPage();
}

async function prepareDepartmentSelection() {
  try {
    departmentOptions.value = await loadDepartmentOptions();
    const selected = departmentOptions.value.find((value) => value.department.departmentId === departmentId.value);
    if (selected) selectDepartment(selected);
  } catch (error) { errorMessage.value = messageOf(error, "科室列表加载失败"); }
}

function selectDepartment(option: DepartmentOption) {
  departmentId.value = option.department.departmentId;
  departmentLabel.value = option.label;
  rememberStaffDepartmentId(departmentId.value);
}

function switchDepartment() {
  if (!canSwitchDepartment.value || loading.value) return;
  uni.showActionSheet({
    title: "选择科室",
    itemList: departmentOptions.value.map((value) => value.label),
    success: ({ tapIndex }) => {
      const selected = departmentOptions.value[tapIndex];
      if (!selected) return;
      selectDepartment(selected);
      rooms.value = [];
      selectedRoomId.value = "";
      void loadPage();
    },
  });
}

async function loadPage() {
  if (!departmentId.value || loading.value) return;
  loading.value = true;
  errorMessage.value = "";
  try {
    if (!isCompletedView.value && !rooms.value.length) {
      rooms.value = (await appointmentManagementApi.listRooms(departmentId.value, 1, 100)).items;
      selectedRoomId.value ||= rooms.value[0]?.roomId ?? "";
    }
    if (isCompletedView.value) {
      bookings.value = (await staffBookingApi.listBookings(departmentId.value, {
        patientKeyword: patientKeyword.value.trim() || undefined,
        view: view.value,
      })).items;
      noShowBookings.value = [];
    } else {
      const filters = { serviceDate: serviceDate.value, roomId: selectedRoomId.value || undefined };
      const [activePage, noShowPage] = await Promise.all([
        staffBookingApi.listBookings(departmentId.value, { ...filters, view: "active" }),
        staffBookingApi.listBookings(departmentId.value, { ...filters, status: "no_show", view: "completed" }),
      ]);
      bookings.value = activePage.items;
      noShowBookings.value = noShowPage.items;
    }
  } catch (error) { errorMessage.value = messageOf(error, "科室预约加载失败"); }
  finally { loading.value = false; }
}

function selectRoom(roomId: string) { if (selectedRoomId.value !== roomId) { selectedRoomId.value = roomId; void loadPage(); } }
function selectDate(event: { detail: { value: string } }) { serviceDate.value = event.detail.value; void loadPage(); }
function openBooking(value: PatientBooking) { uni.navigateTo({ url: `/pages/admin/appointment/booking-detail?booking_id=${encodeURIComponent(value.bookingId)}` }); }

async function callNext() {
  if (!isToday.value || !selectedRoomId.value || currentBooking.value || pendingId.value) return;
  pendingId.value = "call-next";
  try { await staffBookingApi.callNext(departmentId.value, selectedRoomId.value, serviceDate.value); uni.showToast({ title: "已叫下一位", icon: "success" }); await loadAfterMutation(); }
  catch (error) { showOperationError("叫号失败", error); }
  finally { pendingId.value = ""; }
}

async function startExamination(value: PatientBooking) {
  if (value.status !== "called" || pendingId.value) return;
  pendingId.value = value.bookingId;
  try { await staffBookingApi.startExamination(value); uni.showToast({ title: "已开始检查", icon: "success" }); await loadAfterMutation(); }
  catch (error) { showOperationError("开始检查失败", error); }
  finally { pendingId.value = ""; }
}

async function endExamination(value: PatientBooking) {
  if (value.status !== "in_progress" || pendingId.value) return;
  pendingId.value = value.bookingId;
  try {
    const endedBooking = await staffBookingApi.endExamination(value);
    uni.showToast({ title: "检查已结束，请填写报告", icon: "success" });
    openBooking(endedBooking);
  }
  catch (error) { showOperationError("结束检查失败", error); }
  finally { pendingId.value = ""; }
}

async function loadAfterMutation() { loading.value = false; await loadPage(); }
function showOperationError(title: string, error: unknown) { uni.showModal({ title, content: messageOf(error, "请刷新后重试"), showCancel: false }); }
function today() { return new Date(Date.now() + 8 * 60 * 60 * 1000).toISOString().slice(0, 10); }
function decode(value?: string) { try { return value ? decodeURIComponent(value) : ""; } catch { return ""; } }
function messageOf(error: unknown, fallback: string) {
  if (error instanceof ApiError) {
    const messages: Record<string, string> = { QUEUE_EMPTY: "当前房间暂无可以叫号的已报到患者", ROOM_QUEUE_BUSY: "请先处理当前叫号或正在检查的患者", CALL_EXPIRED: "本次叫号已经超时，患者已返回候检队列" };
    if (messages[error.code]) return messages[error.code];
  }
  return error instanceof Error && error.message.trim() ? error.message : fallback;
}
function statusLabel(value: PatientBookingStatus) { return ({ confirmed: "待报到", queued: "排队中", called: "正在叫号", in_progress: "检查中", report_pending: "报告待完成", completed: "已完成", no_show: "未到场", canceled: "已取消" } as const)[value]; }
function roomAddress(value: AppointmentRoom) { return `${value.building} · ${value.floorNumber}层 · ${value.roomNumber}室`; }
function bookingAddress(value: PatientBooking) { return `${value.building} · ${value.floorNumber}层 · ${value.roomNumber}室`; }
</script>

<template>
  <view class="page">
    <view class="summary">
      <view><text class="summary__title">{{ departmentLabel }}</text><text class="summary__minor">{{ isCompletedView ? '查询本科室诊后检查记录' : '按房间处理今日检查队列' }}</text></view>
      <view class="summary__actions"><button v-if="canSwitchDepartment" @tap="switchDepartment">切换科室</button><button @tap="loadPage">刷新</button></view>
    </view>
    <view v-if="isCompletedView" class="search"><input v-model="patientKeyword" confirm-type="search" placeholder="患者姓名 / 手机号后四位" @confirm="loadPage" /><button @tap="loadPage">搜索</button></view>
    <template v-else>
      <view class="date-bar"><text>检查日期</text><picker mode="date" :value="serviceDate" @change="selectDate"><view class="date-value">{{ serviceDate }} ›</view></picker><text v-if="!isToday" class="readonly">历史日期只读</text></view>
      <scroll-view scroll-x class="rooms"><view class="room-row"><view v-for="room in rooms" :key="room.roomId" class="room-chip" :class="{ 'room-chip--active': selectedRoomId === room.roomId }" @tap="selectRoom(room.roomId)"><text>{{ roomAddress(room) }}</text></view></view></scroll-view>
    </template>
    <view v-if="loading" class="state">正在加载…</view>
    <view v-else-if="errorMessage" class="state state--error" @tap="loadPage">{{ errorMessage }}</view>
    <view v-else-if="isCompletedView" class="record-list">
      <view v-if="!bookings.length" class="state">暂无符合条件的检查记录</view>
      <view v-for="booking in bookings" :key="booking.bookingId" class="record" @tap="openBooking(booking)"><view class="card-heading"><text class="patient">{{ booking.patientDisplayName }}</text><text class="status">{{ statusLabel(booking.status) }}</text></view><text class="project">{{ booking.itemName }}</text><text class="line">{{ booking.serviceDate }} · {{ booking.itemStartTime.slice(0,5) }}–{{ booking.itemEndTime.slice(0,5) }}</text><text class="line">{{ bookingAddress(booking) }}</text></view>
    </view>
    <template v-else-if="selectedRoomId">
      <view class="current-panel" :class="{ 'current-panel--called': currentCalled && !currentExamination }">
        <text class="section-label">{{ currentExamination ? '当前检查' : currentCalled ? '当前叫号' : '房间状态' }}</text>
        <template v-if="currentBooking"><view class="card-heading"><text class="current-name">{{ currentBooking.patientDisplayName }}</text><text class="current-number">{{ currentBooking.queueNumber }} 号</text></view><text class="project">{{ currentBooking.itemName }} · {{ currentBooking.patientPhoneMasked }}</text><text v-if="currentCalled" class="call-hint">{{ currentCallHint }}</text><button v-if="isToday && currentCalled" class="primary" :disabled="Boolean(pendingId) || currentCallExpired" @tap="startExamination(currentCalled)">确认患者到场并开始检查</button><button v-if="isToday && currentExamination" class="primary" :disabled="Boolean(pendingId)" @tap="endExamination(currentExamination)">结束本次检查</button></template>
        <text v-else class="idle">当前没有正在叫号或检查的患者</text>
      </view>
      <view class="queue-section"><view class="section-heading"><view><text class="section-title">候检队列</text><text class="section-count">{{ queuedBookings.length }} 人</text></view><button v-if="isToday" class="call-button" :disabled="Boolean(currentBooking) || !queuedBookings.length || Boolean(pendingId)" @tap="callNext">叫下一位</button></view><view v-if="!queuedBookings.length" class="empty">暂无已报到患者</view><view v-for="booking in queuedBookings" :key="booking.bookingId" class="queue-row" @tap="openBooking(booking)"><text class="ticket">{{ booking.queueNumber }}</text><view class="queue-person"><text>{{ booking.patientDisplayName }}</text><text>{{ booking.itemName }} · {{ booking.patientPhoneMasked }}</text></view><text class="attempts">{{ booking.callAttempts ? `已叫 ${booking.callAttempts} 次` : '等待叫号' }}</text></view></view>
      <view class="queue-section"><view class="section-heading"><view><text class="section-title">尚未报到</text><text class="section-count">{{ confirmedBookings.length }} 人</text></view></view><view v-if="!confirmedBookings.length" class="empty">暂无待报到患者</view><view v-for="booking in confirmedBookings" :key="booking.bookingId" class="waiting-row" @tap="openBooking(booking)"><view><text class="patient">{{ booking.patientDisplayName }}</text><text class="line">{{ booking.itemName }} · {{ booking.itemStartTime.slice(0,5) }}–{{ booking.itemEndTime.slice(0,5) }}</text></view><text class="status">待报到</text></view></view>
      <view class="queue-section"><view class="section-heading"><view><text class="section-title">未到场</text><text class="section-count">{{ noShowBookings.length }} 人</text></view></view><view v-if="!noShowBookings.length" class="empty">暂无未到场患者</view><view v-for="booking in noShowBookings" :key="booking.bookingId" class="waiting-row waiting-row--no-show" @tap="openBooking(booking)"><view><text class="patient">{{ booking.patientDisplayName }}</text><text class="line">{{ booking.itemName }} · {{ booking.itemStartTime.slice(0,5) }}–{{ booking.itemEndTime.slice(0,5) }}</text></view><text class="status status--no-show">未到场</text></view></view>
    </template>
    <view v-else-if="!loading" class="state">当前科室尚未配置房间</view>
  </view>
</template>

<style scoped>
button::after{display:none}.page{min-height:100vh;padding:22rpx 22rpx calc(32rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#f3f5f8;color:#263348}.summary,.search,.date-bar,.current-panel,.queue-section,.record,.state{background:#fff;border:1rpx solid #e1e6ed;border-radius:18rpx}.summary{display:flex;align-items:center;justify-content:space-between;padding:24rpx}.summary__title,.summary__minor,.project,.line,.section-label,.idle,.call-hint{display:block}.summary__title{font-size:28rpx;font-weight:700}.summary__minor{margin-top:7rpx;color:#8995a5;font-size:19rpx}.summary__actions{display:flex;gap:10rpx}.summary button,.search button{width:auto;margin:0;padding:0 20rpx;color:#247eb7;font-size:20rpx;line-height:56rpx;background:#edf6fb;border-radius:28rpx}.search{display:grid;grid-template-columns:1fr auto;gap:12rpx;margin-top:16rpx;padding:14rpx}.search input{height:60rpx;padding:0 18rpx;font-size:21rpx;background:#f3f6f9;border-radius:14rpx}.date-bar{display:flex;align-items:center;gap:18rpx;margin-top:16rpx;padding:20rpx 24rpx;font-size:21rpx}.date-value{color:#2188c7;font-weight:650}.readonly{margin-left:auto;color:#9aa5b3;font-size:18rpx}.rooms{margin:16rpx 0;white-space:nowrap}.room-row{display:inline-flex;gap:12rpx}.room-chip{padding:18rpx 22rpx;color:#68778a;font-size:20rpx;background:#fff;border:1rpx solid #dfe6ee;border-radius:16rpx}.room-chip--active{color:#fff;background:#2188c7;border-color:#2188c7}.state{margin-top:18rpx;padding:30rpx;color:#8490a0;font-size:22rpx;text-align:center}.state--error{color:#bd4d5c}.current-panel{padding:26rpx;border-left:7rpx solid #22a276}.current-panel--called{border-left-color:#2188c7;background:#f7fcff}.section-label{color:#8190a3;font-size:19rpx}.card-heading,.section-heading,.waiting-row{display:flex;align-items:center;justify-content:space-between}.current-name{margin-top:12rpx;font-size:34rpx;font-weight:750}.current-number{color:#2188c7;font-size:28rpx;font-weight:750}.project{margin-top:13rpx;color:#627188;font-size:22rpx}.call-hint{margin-top:12rpx;color:#247eb7;font-size:20rpx}.idle{margin-top:16rpx;color:#8995a5;font-size:22rpx}.primary,.call-button{margin:24rpx 0 0;padding:0;color:#fff;font-size:22rpx;line-height:68rpx;background:#2188c7;border-radius:34rpx}.primary[disabled]{color:#9aa6b4;background:#edf1f5}.queue-section{margin-top:18rpx;padding:24rpx}.section-title{font-size:26rpx;font-weight:700}.section-count{margin-left:10rpx;color:#8090a3;font-size:20rpx}.call-button{width:auto;margin:0;padding:0 22rpx;line-height:58rpx}.call-button[disabled]{color:#9aa6b4;background:#edf1f5}.empty{padding:34rpx 0 18rpx;color:#9aa5b3;font-size:21rpx;text-align:center}.queue-row,.waiting-row{margin-top:16rpx;padding:18rpx;background:#f7f9fb;border-radius:14rpx}.waiting-row--no-show{background:#faf7f3}.queue-row{display:flex;align-items:center;gap:16rpx}.ticket{display:flex;align-items:center;justify-content:center;width:58rpx;height:58rpx;color:#fff;font-size:24rpx;font-weight:700;background:#2188c7;border-radius:50%}.queue-person{flex:1}.queue-person text{display:block;font-size:23rpx}.queue-person text+text{margin-top:6rpx;color:#8190a3;font-size:18rpx}.attempts,.status{color:#247eb7;font-size:18rpx}.status--no-show{color:#8a5b35}.patient{font-size:25rpx;font-weight:700}.line{margin-top:8rpx;color:#7d8999;font-size:20rpx}.record{margin-top:16rpx;padding:24rpx}.record .project{color:#344157;font-weight:650}
</style>
