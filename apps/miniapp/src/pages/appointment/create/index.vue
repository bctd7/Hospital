<script setup lang="ts">
import { computed, onMounted, ref } from "vue";

import {
  patientAppointmentMockAdapter,
  type PatientMockDepartment,
} from "@/mocks/appointmentBooking";

interface SearchInputEvent {
  detail: { value?: string };
}

const departments = ref<PatientMockDepartment[]>([]);
const departmentSearch = ref("");
const selectedDepartmentId = ref("");
const selectedItemId = ref("");
const selectedRoomId = ref("");
const selectedWindowId = ref("");

const filteredDepartments = computed(() => {
  const keyword = departmentSearch.value.trim().toLowerCase();
  if (!keyword) return departments.value;
  return departments.value.filter((value) =>
    `${value.campus} ${value.name}`.toLowerCase().includes(keyword),
  );
});
const selectedDepartment = computed(() => departments.value.find((value) => value.id === selectedDepartmentId.value));
const items = computed(() => selectedDepartment.value?.items ?? []);
const selectedItem = computed(() => items.value.find((value) => value.id === selectedItemId.value));
const rooms = computed(() => selectedItem.value?.rooms ?? []);
const selectedRoom = computed(() => rooms.value.find((value) => value.id === selectedRoomId.value));
const windows = computed(() => selectedRoom.value?.windows ?? []);
const selectedWindow = computed(() => windows.value.find((value) => value.id === selectedWindowId.value));
const canConfirm = computed(() => Boolean(selectedDepartment.value && selectedItem.value && selectedRoom.value && selectedWindow.value));

onMounted(async () => {
  departments.value = await patientAppointmentMockAdapter.listDepartments();
  const first = departments.value[0];
  if (first) chooseDepartment(first.id);
});

function chooseDepartment(id: string) {
  selectedDepartmentId.value = id;
  selectedItemId.value = "";
  selectedRoomId.value = "";
  selectedWindowId.value = "";
  const firstItem = departments.value.find((value) => value.id === id)?.items[0];
  if (firstItem) chooseItem(firstItem.id);
}

function chooseItem(id: string) {
  selectedItemId.value = id;
  selectedRoomId.value = "";
  selectedWindowId.value = "";
  const firstRoom = items.value.find((value) => value.id === id)?.rooms[0];
  if (firstRoom) chooseRoom(firstRoom.id);
}

function chooseRoom(id: string) {
  selectedRoomId.value = id;
  selectedWindowId.value = "";
}

function updateSearch(event: Event) {
  departmentSearch.value = (event as unknown as SearchInputEvent).detail.value ?? "";
}

function confirmMockAppointment() {
  if (!canConfirm.value) return;
  uni.showModal({
    title: "Mock 预约预览",
    content: `${selectedDepartment.value?.campus} / ${selectedDepartment.value?.name}\n${selectedItem.value?.name} · ${selectedRoom.value?.displayName}\n${selectedWindow.value?.weekday} ${selectedWindow.value?.session} ${selectedWindow.value?.time}\n\n当前只验证患者选择流程，不会创建真实预约。`,
    showCancel: false,
    confirmText: "我知道了",
  });
}
</script>

<template>
  <view class="booking-page">
    <view class="search-bar">
      <text class="search-bar__icon">⌕</text>
      <input :value="departmentSearch" placeholder="请输入科室名" @input="updateSearch" />
    </view>

    <view class="hospital-card">
      <view class="hospital-card__image"><text>Hospital</text><text>医疗服务中心</text></view>
      <view class="hospital-card__content">
        <text class="hospital-card__name">{{ selectedDepartment?.campus || '医院院区' }}</text>
        <text class="hospital-card__level">检查预约 · MOCK</text>
        <text class="hospital-card__address">{{ selectedDepartment?.name || '请选择科室' }}</text>
      </view>
      <text class="hospital-card__switch">← 左侧切换</text>
    </view>

    <view class="cascade">
      <scroll-view scroll-y class="cascade__column cascade__column--departments">
        <view class="column-heading"><text>科室</text><text>{{ filteredDepartments.length }}</text></view>
        <button
          v-for="department in filteredDepartments"
          :key="department.id"
          class="cascade-option cascade-option--department"
          :class="{ selected: selectedDepartmentId === department.id }"
          @tap="chooseDepartment(department.id)"
        >
          <text>{{ department.name }}</text><text class="cascade-option__minor">{{ department.campus }}</text>
        </button>
        <text v-if="!filteredDepartments.length" class="column-empty">没有匹配科室</text>
      </scroll-view>

      <scroll-view scroll-y class="cascade__column cascade__column--items">
        <view class="column-heading"><text>检查</text><text>{{ items.length }}</text></view>
        <button
          v-for="item in items"
          :key="item.id"
          class="cascade-option"
          :class="{ selected: selectedItemId === item.id }"
          @tap="chooseItem(item.id)"
        >
          <text>{{ item.name }}</text><text class="cascade-option__arrow">›</text>
        </button>
        <text v-if="selectedDepartment && !items.length" class="column-empty">暂无检查项目</text>
      </scroll-view>

      <scroll-view scroll-y class="cascade__column cascade__column--rooms">
        <view class="column-heading"><text>房间</text><text>{{ rooms.length }}</text></view>
        <button
          v-for="room in rooms"
          :key="room.id"
          class="cascade-option cascade-option--room"
          :class="{ selected: selectedRoomId === room.id }"
          @tap="chooseRoom(room.id)"
        >
          <text>{{ room.displayName }}</text><text class="cascade-option__check">{{ selectedRoomId === room.id ? '✓' : '›' }}</text>
        </button>
        <text v-if="selectedItem && !rooms.length" class="column-empty">暂无可选房间</text>
      </scroll-view>
    </view>

    <view v-if="selectedItem && selectedRoom" class="selection-detail">
      <view class="selection-detail__heading">
        <view><text class="selection-detail__title">{{ selectedItem.name }}</text><text class="selection-detail__room">{{ selectedRoom.displayName }}</text></view>
        <text class="mock-badge">MOCK</text>
      </view>
      <text class="selection-detail__description">{{ selectedItem.description }}</text>
      <view class="preparation"><text>检查准备</text><text>{{ selectedItem.preparation }}</text></view>
      <text class="window-heading">选择本周时间</text>
      <button
        v-for="window in windows"
        :key="window.id"
        class="window-card"
        :class="{ selected: selectedWindowId === window.id }"
        @tap="selectedWindowId = window.id"
      >
        <view><text class="window-card__date">{{ window.weekday }} · {{ window.date }} · {{ window.session }}</text><text class="window-card__time">{{ window.time }}，{{ window.cutoff }} 停止新增</text></view>
        <text class="cascade-option__check">{{ selectedWindowId === window.id ? '✓' : '›' }}</text>
      </button>
    </view>

    <view class="booking-footer">
      <text class="booking-footer__hint">演示数据不会写入后端，也不代表真实剩余容量</text>
      <button class="confirm-button" :disabled="!canConfirm" @tap="confirmMockAppointment">预览选择</button>
    </view>
  </view>
</template>

<style scoped>
button::after{display:none}.booking-page{min-height:100vh;padding:22rpx 0 180rpx;box-sizing:border-box;background:#f5f7fa}.search-bar{display:flex;align-items:center;gap:14rpx;margin:0 20rpx;padding:0 22rpx;background:#fff;border:2rpx solid #4d8dff;border-radius:42rpx}.search-bar__icon{color:#6f7886;font-size:38rpx}.search-bar input{height:82rpx;flex:1;color:#252d39;font-size:28rpx}.hospital-card{display:flex;align-items:center;gap:18rpx;margin:20rpx;padding:20rpx;background:#fff;border-radius:14rpx}.hospital-card__image{display:flex;flex:0 0 142rpx;height:94rpx;flex-direction:column;justify-content:flex-end;padding:10rpx;box-sizing:border-box;color:#fff;font-size:17rpx;background:linear-gradient(150deg,#77b8e4,#3b7fab 55%,#7eb85d);border-radius:5rpx}.hospital-card__image text:last-child{font-size:13rpx}.hospital-card__content{flex:1;min-width:0}.hospital-card__name,.hospital-card__level,.hospital-card__address{display:block}.hospital-card__name{color:#252b34;font-size:25rpx;font-weight:650;line-height:1.35}.hospital-card__level{width:max-content;margin-top:6rpx;padding:2rpx 10rpx;color:#347ff0;font-size:17rpx;background:#eef4ff}.hospital-card__address{margin-top:7rpx;color:#7b8492;font-size:18rpx}.hospital-card__switch{align-self:flex-start;color:#347ff0;font-size:18rpx}.flow-hint{display:flex;align-items:center;justify-content:center;gap:12rpx;padding:0 20rpx 18rpx;color:#8490a0;font-size:21rpx}.flow-hint__active{color:#347ff0;font-weight:700}.cascade{display:grid;grid-template-columns:190rpx 238rpx minmax(0,1fr);height:620rpx;overflow:hidden;background:#fff;border-top:1rpx solid #e6eaf0;border-bottom:1rpx solid #e6eaf0}.cascade__column{height:620rpx;box-sizing:border-box}.cascade__column--departments{background:#f1f4f8}.cascade__column--items{background:#fafbfd;border-right:1rpx solid #e7ebf0}.column-heading{position:sticky;top:0;z-index:2;display:flex;align-items:center;justify-content:space-between;height:70rpx;padding:0 18rpx;color:#788496;font-size:19rpx;font-weight:650;background:inherit;border-bottom:1rpx solid #e6eaf0}.cascade-option{display:flex;align-items:center;justify-content:space-between;width:100%;min-height:86rpx;margin:0;padding:16rpx 18rpx;color:#29313c;font-size:22rpx;line-height:1.35;text-align:left;background:transparent;border-bottom:1rpx solid #e8ecf1;border-radius:0}.cascade-option--department{display:flex;flex-direction:column;align-items:flex-start;justify-content:center}.cascade-option.selected{position:relative;color:#347ff0;font-weight:700;background:#fff}.cascade-option.selected::before{position:absolute;top:0;bottom:0;left:0;width:7rpx;background:#347ff0;content:""}.cascade-option__minor{margin-top:5rpx;color:#8c96a4;font-size:16rpx;font-weight:400}.cascade-option__arrow,.cascade-option__check{color:#347ff0;font-size:27rpx}.cascade-option--room{min-height:94rpx}.column-empty{display:block;padding:30rpx 14rpx;color:#9ba5b2;font-size:18rpx;line-height:1.5;text-align:center}.selection-detail{margin:22rpx;padding:24rpx;background:#fff;border:1rpx solid #e5eaf0;border-radius:18rpx}.selection-detail__heading{display:flex;align-items:flex-start;justify-content:space-between}.selection-detail__title,.selection-detail__room,.selection-detail__description,.window-heading,.window-card__date,.window-card__time{display:block}.selection-detail__title{color:#293547;font-size:28rpx;font-weight:700}.selection-detail__room{margin-top:6rpx;color:#347ff0;font-size:21rpx}.mock-badge{padding:5rpx 10rpx;color:#a16914;font-size:16rpx;background:#fff1d6;border-radius:8rpx}.selection-detail__description{margin-top:16rpx;color:#7e8999;font-size:20rpx;line-height:1.6}.preparation{margin-top:18rpx;padding:16rpx;color:#765e36;font-size:19rpx;line-height:1.55;background:#fff8e8;border-radius:12rpx}.preparation text{display:block}.preparation text:first-child{font-weight:700}.window-heading{margin-top:22rpx;color:#354154;font-size:23rpx;font-weight:700}.window-card{display:flex;align-items:center;justify-content:space-between;width:100%;margin:12rpx 0 0;padding:18rpx;color:inherit;text-align:left;background:#f8fafc;border:1rpx solid #e5eaf0;border-radius:14rpx}.window-card.selected{background:#edf5ff;border-color:#4b91f3}.window-card__date{color:#334055;font-size:22rpx;font-weight:650}.window-card__time{margin-top:6rpx;color:#8c97a8;font-size:18rpx}.booking-footer{position:fixed;right:0;bottom:0;left:0;z-index:5;display:flex;align-items:center;gap:18rpx;padding:18rpx 22rpx calc(18rpx + env(safe-area-inset-bottom));background:rgba(255,255,255,.97);border-top:1rpx solid #e5eaf0}.booking-footer__hint{flex:1;color:#8c97a7;font-size:18rpx;line-height:1.4}.confirm-button{flex:0 0 auto;margin:0;padding:0 32rpx;color:#fff;font-size:23rpx;line-height:70rpx;background:#347ff0;border-radius:35rpx}.confirm-button[disabled]{color:#fff;background:#bdc8d3}
</style>
