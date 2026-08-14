<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import { patientAppointmentApi } from "@/api/appointment";
import { ApiError } from "@/api/client";
import { loadDepartmentOptions, type DepartmentOption } from "@/services/organization";
import type { BookingOption, ExaminationItem } from "@/types/appointment";

interface SearchInputEvent { detail: { value?: string } }

const departments = ref<DepartmentOption[]>([]);
const items = ref<ExaminationItem[]>([]);
const options = ref<BookingOption[]>([]);
const departmentSearch = ref("");
const selectedDepartmentId = ref("");
const selectedItemId = ref("");
const selectedRoomId = ref("");
const selectedOptionKey = ref("");
const loading = ref(false);
const loadingItems = ref(false);
const loadingOptions = ref(false);
const submitting = ref(false);
const errorMessage = ref("");
const showingItemRooms = ref(false);
let requestGeneration = 0;

const filteredDepartments = computed(() => {
  const keyword = departmentSearch.value.trim().toLowerCase();
  return keyword
    ? departments.value.filter((value) => value.label.toLowerCase().includes(keyword))
    : departments.value;
});
const selectedDepartment = computed(() => departments.value.find((value) => value.department.departmentId === selectedDepartmentId.value));
const selectedItem = computed(() => items.value.find((value) => value.itemId === selectedItemId.value));
const rooms = computed(() => {
  const values = new Map<string, BookingOption>();
  options.value.forEach((value) => values.set(value.roomId, value));
  return [...values.values()];
});
const selectedRoom = computed(() => rooms.value.find((value) => value.roomId === selectedRoomId.value));
const roomOptions = computed(() => options.value.filter((value) => value.roomId === selectedRoomId.value));
const selectedOption = computed(() => roomOptions.value.find((value) => optionKey(value) === selectedOptionKey.value));
const canConfirm = computed(() => Boolean(selectedItem.value && selectedRoom.value && selectedOption.value && !submitting.value));
const workspaceClass = computed(() => ({
  "workspace-track--item-rooms": showingItemRooms.value,
}));

onMounted(async () => {
  loading.value = true;
  try {
    departments.value = await loadDepartmentOptions();
    const first = departments.value[0];
    if (first) await chooseDepartment(first.department.departmentId);
  } catch (error) {
    errorMessage.value = messageOf(error, "预约资源加载失败，请重试");
  } finally {
    loading.value = false;
  }
});

async function chooseDepartment(id: string) {
  const generation = ++requestGeneration;
  showingItemRooms.value = false;
  selectedDepartmentId.value = id;
  selectedItemId.value = "";
  selectedRoomId.value = "";
  selectedOptionKey.value = "";
  items.value = [];
  options.value = [];
  errorMessage.value = "";
  loadingItems.value = true;
  try {
    const result = await patientAppointmentApi.listItems(id);
    if (generation !== requestGeneration) return;
    items.value = result;
    loadingItems.value = false;
    if (result[0]) await chooseItem(result[0].itemId, false);
  } catch (error) {
    if (generation === requestGeneration) errorMessage.value = messageOf(error, "检查项目加载失败");
  } finally {
    if (generation === requestGeneration) loadingItems.value = false;
  }
}

async function chooseItem(id: string, revealRooms = true) {
  if (revealRooms) showingItemRooms.value = true;
  if (id === selectedItemId.value && options.value.length) return;

  const generation = ++requestGeneration;
  selectedItemId.value = id;
  selectedRoomId.value = "";
  selectedOptionKey.value = "";
  options.value = [];
  errorMessage.value = "";
  loadingOptions.value = true;
  try {
    const result = await patientAppointmentApi.listBookingOptions(id);
    if (generation !== requestGeneration) return;
    options.value = result.options;
    if (rooms.value[0]) chooseRoom(rooms.value[0].roomId);
  } catch (error) {
    if (generation === requestGeneration) errorMessage.value = messageOf(error, "可预约房间加载失败");
  } finally {
    if (generation === requestGeneration) loadingOptions.value = false;
  }
}

function returnToDepartments() {
  showingItemRooms.value = false;
}

function chooseRoom(id: string) {
  selectedRoomId.value = id;
  selectedOptionKey.value = "";
}

function optionKey(value: BookingOption) {
  return `${value.roomId}:${value.serviceDate}:${value.session}`;
}

function updateSearch(event: Event) {
  departmentSearch.value = (event as unknown as SearchInputEvent).detail.value ?? "";
}

async function confirmAppointment() {
  const item = selectedItem.value;
  const room = selectedRoom.value;
  const option = selectedOption.value;
  if (!item || !room || !option || submitting.value) return;
  submitting.value = true;
  try {
    await patientAppointmentApi.createBooking(item.itemId, room.roomId, option.serviceDate, option.session);
    uni.showToast({ title: "预约成功", icon: "success" });
    setTimeout(() => void uni.navigateTo({ url: "/pages/profile/appointments/index" }), 500);
  } catch (error) {
    const content = error instanceof ApiError && error.code === "PATIENT_SESSION_OCCUPIED"
      ? "同一天的同一上午或下午只能保留一个待核销预约；完成核销后可立即再次预约。"
      : messageOf(error, "请刷新后重试");
    uni.showModal({ title: "预约失败", content, showCancel: false });
  } finally {
    submitting.value = false;
  }
}

function messageOf(error: unknown, fallback: string) {
  return error instanceof Error && error.message.trim() ? error.message : fallback;
}

function sessionLabel(value: string) { return value === "morning" ? "上午" : "下午"; }
function floorLabel(value: number) { return value < 0 ? `地下${Math.abs(value)}层` : `${value}层`; }
</script>

<template>
  <view class="booking-page">
    <view class="search-bar"><text class="search-bar__icon">⌕</text><input :value="departmentSearch" placeholder="请输入科室名" @input="updateSearch" /></view>

    <view class="hospital-card">
      <view class="hospital-card__image"><text>Hospital</text><text>医疗服务中心</text></view>
      <view class="hospital-card__content">
        <text class="hospital-card__name">{{ selectedDepartment?.campus.name || '医院院区' }}</text>
        <text class="hospital-card__level">检查预约</text>
        <text class="hospital-card__address">{{ selectedDepartment?.department.name || '请选择科室' }}</text>
      </view>
      <text class="hospital-card__switch">院区可切换</text>
    </view>

    <text v-if="errorMessage" class="page-message page-message--error">{{ errorMessage }}</text>
    <text v-else-if="loading" class="page-message">正在加载预约资源…</text>

    <view class="hierarchy-bar" :class="{ 'hierarchy-bar--visible': showingItemRooms }">
      <button class="hierarchy-bar__back" @tap="returnToDepartments">‹ 返回科室</button>
      <view class="hierarchy-bar__breadcrumb">
        <text>{{ selectedDepartment?.department.name || '所选科室' }}</text>
        <text class="hierarchy-bar__separator">/</text>
        <text>{{ selectedItem?.name || '所选项目' }}</text>
      </view>
    </view>

    <view class="workspace">
      <view class="workspace-track" :class="workspaceClass">
        <view class="workspace-column workspace-column--departments">
          <view class="column-heading"><text>科室</text><text>{{ filteredDepartments.length }}</text></view>
          <scroll-view scroll-y class="column-body">
            <button v-for="entry in filteredDepartments" :key="entry.department.departmentId" class="cascade-option cascade-option--department" :class="{ selected: selectedDepartmentId === entry.department.departmentId }" @tap="chooseDepartment(entry.department.departmentId)">
              <text>{{ entry.department.name }}</text><text class="cascade-option__minor">{{ entry.campus.name }}</text>
            </button>
            <text v-if="!filteredDepartments.length" class="column-empty">没有匹配科室</text>
          </scroll-view>
        </view>

        <view class="workspace-column workspace-column--items">
          <view class="column-heading"><text>检查项目</text><text>{{ items.length }}</text></view>
          <scroll-view scroll-y class="column-body">
            <text v-if="loadingItems" class="column-empty">加载中...</text>
            <button v-for="item in items" :key="item.itemId" class="cascade-option" :class="{ selected: selectedItemId === item.itemId }" @tap="chooseItem(item.itemId)">
              <text>{{ item.name }}</text><text class="cascade-option__arrow">›</text>
            </button>
            <text v-if="selectedDepartment && !loadingItems && !items.length" class="column-empty">暂无检查项目</text>
          </scroll-view>
        </view>

        <view class="workspace-column workspace-column--rooms">
          <view class="column-heading"><text>房间</text><text>{{ rooms.length }}</text></view>
          <scroll-view scroll-y class="column-body">
            <text v-if="loadingOptions" class="column-empty">加载中...</text>
            <button v-for="room in rooms" :key="room.roomId" class="room-card" :class="{ 'room-card--selected': selectedRoomId === room.roomId }" @tap="chooseRoom(room.roomId)">
              <text class="room-card__number">{{ room.roomNumber }}室</text>
              <text class="room-card__address">园区：{{ selectedDepartment?.campus.name || room.campusId }}</text>
              <text class="room-card__address">楼栋：{{ room.building }}</text>
              <text class="room-card__address">楼层：{{ floorLabel(room.floorNumber) }}</text>
              <text class="room-card__address">房间号：{{ room.roomNumber }}</text>
              <text class="room-card__check">{{ selectedRoomId === room.roomId ? '✓' : '›' }}</text>
            </button>
            <text v-if="selectedItem && !loadingOptions && !rooms.length" class="column-empty">本周暂无可预约房间</text>
          </scroll-view>
        </view>
      </view>
    </view>

    <view v-if="showingItemRooms && selectedItem && selectedRoom" class="selection-detail">
      <text class="selection-detail__title">{{ selectedItem.name }}</text>
      <text class="selection-detail__room">{{ selectedRoom.roomDisplayName }}</text>
      <text class="selection-detail__description">{{ selectedItem.description }}</text>
      <text class="window-heading">选择本周时间</text>
      <button v-for="option in roomOptions" :key="optionKey(option)" class="window-card" :class="{ selected: selectedOptionKey === optionKey(option) }" @tap="selectedOptionKey = optionKey(option)">
        <view><text class="window-card__date">{{ option.serviceDate }} · {{ sessionLabel(option.session) }}</text><text class="window-card__time">{{ option.itemStartTime }}–{{ option.itemEndTime }}，剩余 {{ option.remainingCapacity }}</text></view>
        <text class="cascade-option__check">{{ selectedOptionKey === optionKey(option) ? '✓' : '›' }}</text>
      </button>
      <text v-if="!roomOptions.length" class="column-empty">这个房间本周暂无剩余容量</text>
    </view>

    <view class="booking-footer">
      <text class="booking-footer__hint">每条预约占用一个房间容量单位</text>
      <button class="confirm-button" :disabled="!canConfirm" @tap="confirmAppointment">{{ submitting ? '提交中…' : '确认预约' }}</button>
    </view>
  </view>
</template>

<style scoped>
button::after{display:none}.booking-page{min-height:100vh;padding:22rpx 0 190rpx;box-sizing:border-box;background:#f5f7fa}.search-bar{display:flex;align-items:center;gap:14rpx;margin:0 20rpx;padding:0 22rpx;background:#fff;border:2rpx solid #4d8dff;border-radius:42rpx}.search-bar__icon{color:#6f7886;font-size:38rpx}.search-bar input{height:82rpx;flex:1;color:#252d39;font-size:28rpx}.hospital-card{display:flex;align-items:center;gap:18rpx;margin:20rpx;padding:20rpx;background:#fff;border-radius:14rpx}.hospital-card__image{display:flex;flex:0 0 142rpx;height:94rpx;flex-direction:column;justify-content:flex-end;padding:10rpx;box-sizing:border-box;color:#fff;font-size:17rpx;background:linear-gradient(150deg,#77b8e4,#3b7fab 55%,#7eb85d);border-radius:5rpx}.hospital-card__image text:last-child{font-size:13rpx}.hospital-card__content{flex:1;min-width:0}.hospital-card__name,.hospital-card__level,.hospital-card__address{display:block}.hospital-card__name{color:#252b34;font-size:25rpx;font-weight:650}.hospital-card__level{width:max-content;margin-top:6rpx;padding:2rpx 10rpx;color:#347ff0;font-size:17rpx;background:#eef4ff}.hospital-card__address{margin-top:7rpx;color:#7b8492;font-size:18rpx}.hospital-card__switch{align-self:flex-start;color:#347ff0;font-size:18rpx}.page-message{display:block;margin:0 20rpx 18rpx;color:#748093;font-size:20rpx}.page-message--error{color:#c54d4d}.hierarchy-bar{display:flex;max-height:0;align-items:center;gap:18rpx;margin:0 20rpx;padding:0 4rpx;overflow:hidden;box-sizing:border-box;opacity:0;pointer-events:none;transition:max-height 260ms ease,margin 260ms ease,opacity 180ms ease}.hierarchy-bar--visible{max-height:58rpx;margin:0 20rpx 12rpx;opacity:1;pointer-events:auto}.hierarchy-bar__back{flex:0 0 auto;margin:0;padding:0;color:#347ff0;font-size:20rpx;line-height:50rpx;background:transparent}.hierarchy-bar__breadcrumb{display:flex;min-width:0;align-items:center;gap:12rpx;color:#35435a;font-size:21rpx;font-weight:650}.hierarchy-bar__breadcrumb text{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.hierarchy-bar__separator{flex:0 0 auto;color:#9aa4b2;font-weight:400}.workspace{height:620rpx;margin:0 20rpx;overflow:hidden;background:#fff;border:1rpx solid #dfe5ec;border-radius:20rpx}.workspace-track{display:grid;width:150%;height:100%;grid-template-columns:repeat(3,minmax(0,1fr));transform:translateX(0);transition:transform 260ms cubic-bezier(.2,0,0,1)}.workspace-track--item-rooms{transform:translateX(-33.333333%)}.workspace-column{display:flex;min-width:0;min-height:0;flex-direction:column;box-sizing:border-box;background:#fbfcfd;border-right:1rpx solid #e3e8ee;transition:opacity 200ms ease,transform 260ms cubic-bezier(.2,0,0,1)}.workspace-column--departments{background:#f6f8fa;opacity:1}.workspace-column--rooms{border-right:0;opacity:0;transform:translateX(18rpx)}.workspace-track--item-rooms .workspace-column--departments{opacity:0;transform:translateX(-18rpx)}.workspace-track--item-rooms .workspace-column--rooms{opacity:1;transform:translateX(0)}.column-heading{display:flex;flex:0 0 auto;align-items:center;justify-content:space-between;height:70rpx;padding:0 18rpx;color:#788496;font-size:19rpx;font-weight:650;background:#fff;border-bottom:1rpx solid #e6eaf0}.column-body{height:0;flex:1}.cascade-option{display:flex;align-items:center;justify-content:space-between;width:100%;min-height:92rpx;margin:0;padding:16rpx 18rpx;color:#29313c;font-size:22rpx;line-height:1.35;text-align:left;background:transparent;border-bottom:1rpx solid #e8ecf1;border-radius:0}.cascade-option--department{flex-direction:column;align-items:flex-start;justify-content:center}.cascade-option.selected{position:relative;color:#347ff0;font-weight:700;background:#eef5ff}.cascade-option.selected::before,.room-card--selected::before{position:absolute;inset:12rpx auto 12rpx 0;width:7rpx;background:#347ff0;border-radius:0 5rpx 5rpx 0;content:""}.cascade-option__minor{margin-top:5rpx;color:#8c96a4;font-size:16rpx;font-weight:400}.cascade-option__arrow,.cascade-option__check{color:#347ff0;font-size:27rpx}.room-card{position:relative;display:block;width:calc(100% - 24rpx);min-height:210rpx;margin:12rpx 12rpx 0;padding:20rpx;color:inherit;text-align:left;background:#fff;border:1rpx solid #e3e8ee;border-radius:16rpx}.room-card--selected{background:#eef5ff;border-color:#a9cfff}.room-card__number,.room-card__address{display:block}.room-card__number{margin-bottom:12rpx;color:#29374c;font-size:23rpx;font-weight:650}.room-card--selected .room-card__number{color:#347ff0}.room-card__address{margin-top:5rpx;color:#5e6d82;font-size:19rpx;line-height:1.45;word-break:break-all}.room-card__check{position:absolute;top:18rpx;right:18rpx;color:#347ff0;font-size:27rpx}.column-empty{display:block;padding:30rpx 14rpx;color:#9ba5b2;font-size:18rpx;line-height:1.5;text-align:center}.selection-detail{margin:22rpx;padding:24rpx;background:#fff;border:1rpx solid #e5eaf0;border-radius:18rpx}.selection-detail__title,.selection-detail__room,.selection-detail__description,.window-heading,.window-card__date,.window-card__time{display:block}.selection-detail__title{color:#293547;font-size:28rpx;font-weight:700}.selection-detail__room{margin-top:6rpx;color:#347ff0;font-size:21rpx}.selection-detail__description{margin-top:16rpx;color:#7e8999;font-size:20rpx;line-height:1.6}.window-heading{margin-top:22rpx;color:#354154;font-size:23rpx;font-weight:700}.window-card{display:flex;align-items:center;justify-content:space-between;width:100%;margin:12rpx 0 0;padding:18rpx;color:inherit;text-align:left;background:#f8fafc;border:1rpx solid #e5eaf0;border-radius:14rpx}.window-card.selected{background:#edf5ff;border-color:#4b91f3}.window-card__date{color:#334055;font-size:22rpx;font-weight:650}.window-card__time{margin-top:6rpx;color:#8c97a8;font-size:18rpx}.booking-footer{position:fixed;right:0;bottom:0;left:0;z-index:5;display:flex;align-items:center;gap:18rpx;padding:18rpx 22rpx calc(18rpx + env(safe-area-inset-bottom));background:rgba(255,255,255,.98);border-top:1rpx solid #e5eaf0}.booking-footer__hint{flex:1;color:#8c97a7;font-size:18rpx}.confirm-button{margin:0;padding:0 32rpx;color:#fff;font-size:23rpx;line-height:70rpx;background:#347ff0;border-radius:35rpx}.confirm-button[disabled]{color:#fff;background:#bdc8d3}@media (prefers-reduced-motion:reduce){.workspace-track,.workspace-column,.hierarchy-bar{transition-duration:1ms}}
</style>
