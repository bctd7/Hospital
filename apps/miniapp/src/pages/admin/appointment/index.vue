<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import {
  loadAppointmentRooms,
  loadRoomExaminationItems,
} from "@/services/appointment";
import { loadDepartmentOptions, type DepartmentOption } from "@/services/organization";
import { sessionState } from "@/stores/session";
import type {
  AppointmentRoom,
  AppointmentStatus,
  RoomExaminationItem,
} from "@/types/appointment";
import {
  canReadAppointmentManagement,
  hasPermission,
  hasRole,
  messageOf,
} from "@/utils/appointmentManagement";

interface SearchInputEvent {
  detail: { value?: string };
}

let rememberedDepartmentId = "";
let rememberedRoomId = "";
const status = ref<AppointmentStatus>("active");
const departments = ref<DepartmentOption[]>([]);
const selectedDepartmentId = ref("");
const rooms = ref<AppointmentRoom[]>([]);
const selectedRoomId = ref("");
const relations = ref<RoomExaminationItem[]>([]);
const departmentSearch = ref("");
const totalRooms = ref(0);
const page = ref(1);
const loading = ref(false);
const loadingRooms = ref(false);
const loadingRelations = ref(false);
const loadingMore = ref(false);
const error = ref("");
const relationError = ref("");
const navigationPending = ref(false);
let roomGeneration = 0;
let relationGeneration = 0;

const principal = computed(() => sessionState.principal);
const allowed = computed(() => canReadAppointmentManagement(principal.value));
const canCreate = computed(() => hasPermission(principal.value, "appointment.create"));
const isDoctor = computed(() => hasRole(principal.value, "department_doctor"));
const selectedDepartment = computed(() =>
  departments.value.find((option) => option.department.departmentId === selectedDepartmentId.value),
);
const selectedRoom = computed(() => rooms.value.find((room) => room.roomId === selectedRoomId.value));
const filteredDepartments = computed(() => {
  const keyword = departmentSearch.value.trim().toLowerCase();
  if (!keyword) return departments.value;
  return departments.value.filter((option) => option.label.toLowerCase().includes(keyword));
});
const canLoadMore = computed(() => rooms.value.length < totalRooms.value);

onShow(() => {
  navigationPending.value = false;
  uni.setNavigationBarTitle({ title: "检查资源管理" });
  void initialize(false);
});

async function initialize(force: boolean) {
  if (!allowed.value) {
    error.value = "当前账号没有检查资源管理权限";
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    const options = await loadDepartmentOptions(force);
    const ownDepartmentId = principal.value?.department_id ?? "";
    departments.value = isDoctor.value
      ? options.filter((option) => option.department.departmentId === ownDepartmentId)
      : options;
    const next = departments.value.find((option) => option.department.departmentId === selectedDepartmentId.value)
      ?? departments.value.find((option) => option.department.departmentId === rememberedDepartmentId)
      ?? departments.value[0];
    selectedDepartmentId.value = next?.department.departmentId ?? ownDepartmentId;
    rememberedDepartmentId = selectedDepartmentId.value;
    await loadRooms(force);
  } catch (cause) {
    error.value = messageOf(cause, "管理数据加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function loadRooms(force = false) {
  const departmentId = selectedDepartmentId.value;
  const generation = ++roomGeneration;
  page.value = 1;
  rooms.value = [];
  selectedRoomId.value = "";
  relations.value = [];
  relationError.value = "";
  if (!departmentId) return;
  loadingRooms.value = true;
  try {
    const result = await loadAppointmentRooms(departmentId, status.value, 1, force);
    if (generation !== roomGeneration) return;
    rooms.value = result.items;
    totalRooms.value = result.total;
    const next = rooms.value.find((room) => room.roomId === rememberedRoomId) ?? rooms.value[0];
    selectedRoomId.value = next?.roomId ?? "";
    rememberedRoomId = selectedRoomId.value;
    await loadRelations(force);
  } catch (cause) {
    if (generation === roomGeneration) error.value = messageOf(cause, "房间加载失败，请重试");
  } finally {
    if (generation === roomGeneration) loadingRooms.value = false;
  }
}

async function loadRelations(force = false) {
  const roomId = selectedRoomId.value;
  const generation = ++relationGeneration;
  relations.value = [];
  relationError.value = "";
  if (!roomId) return;
  loadingRelations.value = true;
  try {
    const result = await loadRoomExaminationItems(roomId, status.value, force);
    if (generation !== relationGeneration) return;
    relations.value = result.items;
  } catch (cause) {
    if (generation === relationGeneration) {
      relationError.value = messageOf(cause, "可执行项目加载失败");
    }
  } finally {
    if (generation === relationGeneration) loadingRelations.value = false;
  }
}

function chooseDepartment(departmentId: string) {
  if (departmentId === selectedDepartmentId.value) return;
  selectedDepartmentId.value = departmentId;
  rememberedDepartmentId = departmentId;
  rememberedRoomId = "";
  error.value = "";
  void loadRooms(false);
}

function chooseRoom(roomId: string) {
  if (roomId === selectedRoomId.value) return;
  selectedRoomId.value = roomId;
  rememberedRoomId = roomId;
  void loadRelations(false);
}

function updateSearch(event: Event) {
  departmentSearch.value = (event as unknown as SearchInputEvent).detail.value ?? "";
}

function toggleStatus() {
  status.value = status.value === "active" ? "disabled" : "active";
  rememberedRoomId = "";
  error.value = "";
  void loadRooms(false);
}

async function loadMoreRooms() {
  if (!canLoadMore.value || loadingMore.value) return;
  loadingMore.value = true;
  const nextPage = page.value + 1;
  try {
    const result = await loadAppointmentRooms(selectedDepartmentId.value, status.value, nextPage);
    rooms.value.push(...result.items);
    page.value = nextPage;
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "加载更多房间失败"), icon: "none" });
  } finally {
    loadingMore.value = false;
  }
}

function openItem(value?: RoomExaminationItem) {
  navigate(`/pages/admin/appointment/item-detail?department_id=${encodeURIComponent(selectedDepartmentId.value)}&item_id=${encodeURIComponent(value?.itemId ?? "")}&department_label=${encodeURIComponent(selectedDepartment.value?.label ?? "所属科室")}`);
}

function openRoom(value?: AppointmentRoom) {
  navigate(`/pages/admin/appointment/room-detail?department_id=${encodeURIComponent(selectedDepartmentId.value)}&room_id=${encodeURIComponent(value?.roomId ?? "")}&department_label=${encodeURIComponent(selectedDepartment.value?.label ?? "所属科室")}`);
}

function navigate(url: string) {
  if (navigationPending.value) return;
  navigationPending.value = true;
  uni.navigateTo({
    url,
    fail: () => uni.showToast({ title: "页面打开失败", icon: "none" }),
    complete: () => { navigationPending.value = false; },
  });
}
</script>

<template>
  <view class="management-page">
    <view class="search-bar">
      <text class="search-bar__icon">⌕</text>
      <input :value="departmentSearch" placeholder="请输入科室名" @input="updateSearch" />
    </view>

    <view class="organization-card">
      <view>
        <text class="organization-card__eyebrow">APPOINTMENT · 真实数据</text>
        <text class="organization-card__title">{{ selectedDepartment?.label || '正在读取所属科室…' }}</text>
      </view>
      <button v-if="!isDoctor" class="organization-card__switch">左侧切换科室</button>
    </view>

    <view class="flow-hint">
      <text class="flow-hint__active">1 科室</text><text>→</text><text>2 房间</text><text>→</text><text>3 可执行检查</text>
    </view>

    <view v-if="error && !departments.length" class="page-state page-state--error">
      <text>{{ error }}</text><button @tap="initialize(true)">重新加载</button>
    </view>

    <view v-else class="cascade">
      <scroll-view scroll-y class="cascade__column cascade__column--departments">
        <view class="column-heading"><text>科室</text><text>{{ filteredDepartments.length }}</text></view>
        <button
          v-for="option in filteredDepartments"
          :key="option.department.departmentId"
          class="cascade-option cascade-option--department"
          :class="{ selected: selectedDepartmentId === option.department.departmentId }"
          @tap="chooseDepartment(option.department.departmentId)"
        >
          <text>{{ option.department.name }}</text>
          <text class="cascade-option__minor">{{ option.campus.name }}</text>
        </button>
        <text v-if="!filteredDepartments.length" class="column-empty">没有匹配科室</text>
      </scroll-view>

      <scroll-view scroll-y class="cascade__column cascade__column--rooms">
        <view class="column-heading"><text>房间</text><text>{{ rooms.length }}</text></view>
        <text v-if="loadingRooms" class="column-empty">加载中…</text>
        <button
          v-for="room in rooms"
          :key="room.roomId"
          class="cascade-option"
          :class="{ selected: selectedRoomId === room.roomId }"
          @tap="chooseRoom(room.roomId)"
        >
          <text>{{ room.name }}</text><text class="cascade-option__arrow">›</text>
        </button>
        <button v-if="canLoadMore" class="column-action" :disabled="loadingMore" @tap="loadMoreRooms">
          {{ loadingMore ? '加载中…' : '更多房间' }}
        </button>
        <button v-if="canCreate && status === 'active'" class="column-action" @tap="openRoom()">＋ 新建房间</button>
        <text v-if="!loadingRooms && !rooms.length" class="column-empty">暂无{{ status === 'active' ? '启用' : '停用' }}房间</text>
      </scroll-view>

      <scroll-view scroll-y class="cascade__column cascade__column--items">
        <view class="column-heading"><text>检查项目</text><text>{{ relations.length }}</text></view>
        <text v-if="loadingRelations" class="column-empty">加载中…</text>
        <button
          v-for="relation in relations"
          :key="relation.relationId"
          class="cascade-option cascade-option--item"
          @tap="openItem(relation)"
        >
          <text>{{ relation.itemName }}</text><text class="cascade-option__arrow">›</text>
        </button>
        <text v-if="relationError" class="column-empty column-empty--error">{{ relationError }}</text>
        <text v-else-if="selectedRoom && !loadingRelations && !relations.length" class="column-empty">该房间尚未配置项目</text>
        <text v-else-if="!selectedRoom && !loadingRooms" class="column-empty">请先选择房间</text>
        <button v-if="canCreate && status === 'active'" class="column-action" @tap="openItem()">＋ 新建项目</button>
      </scroll-view>
    </view>

    <view class="management-actions">
      <button @tap="toggleStatus">{{ status === 'active' ? '查看停用资源' : '返回启用资源' }}</button>
      <button v-if="selectedRoom" class="management-actions__primary" @tap="openRoom(selectedRoom)">管理所选房间</button>
    </view>
    <text class="page-note">项目属于科室；房间属于科室。进入房间后维护可执行项目、开放时间和共享容量。</text>
  </view>
</template>

<style scoped>
button::after{display:none}.management-page{min-height:100vh;padding:22rpx 0 60rpx;box-sizing:border-box;background:#f5f7fa}.search-bar{display:flex;align-items:center;gap:14rpx;margin:0 22rpx;padding:0 22rpx;background:#fff;border:2rpx solid #4d8dff;border-radius:42rpx}.search-bar__icon{color:#6f7886;font-size:38rpx}.search-bar input{height:82rpx;flex:1;color:#252d39;font-size:28rpx}.organization-card{display:flex;align-items:center;justify-content:space-between;gap:18rpx;margin:22rpx;padding:24rpx;color:#fff;background:linear-gradient(135deg,#3485df,#37a9c9);border-radius:18rpx}.organization-card__eyebrow,.organization-card__title{display:block}.organization-card__eyebrow{font-size:18rpx;opacity:.76;letter-spacing:1rpx}.organization-card__title{margin-top:8rpx;font-size:28rpx;font-weight:700;line-height:1.35}.organization-card__switch{flex:0 0 auto;margin:0;padding:0;color:#fff;font-size:20rpx;line-height:52rpx;background:transparent}.flow-hint{display:flex;align-items:center;justify-content:center;gap:12rpx;padding:0 20rpx 18rpx;color:#8490a0;font-size:21rpx}.flow-hint__active{color:#347ff0;font-weight:700}.cascade{display:grid;grid-template-columns:190rpx 224rpx minmax(0,1fr);height:720rpx;overflow:hidden;background:#fff;border-top:1rpx solid #e6eaf0;border-bottom:1rpx solid #e6eaf0}.cascade__column{height:720rpx;box-sizing:border-box}.cascade__column--departments{background:#f2f5f9}.cascade__column--rooms{background:#fafbfd;border-right:1rpx solid #e8ebf0}.column-heading{position:sticky;top:0;z-index:2;display:flex;align-items:center;justify-content:space-between;height:72rpx;padding:0 18rpx;color:#778396;font-size:20rpx;font-weight:650;background:inherit;border-bottom:1rpx solid #e7ebf0}.cascade-option{display:flex;align-items:center;justify-content:space-between;width:100%;min-height:84rpx;margin:0;padding:16rpx 18rpx;color:#26303d;font-size:23rpx;line-height:1.35;text-align:left;background:transparent;border-bottom:1rpx solid #e9edf2;border-radius:0}.cascade-option--department{display:flex;flex-direction:column;align-items:flex-start;justify-content:center}.cascade-option.selected{position:relative;color:#347ff0;font-weight:700;background:#fff}.cascade-option.selected::before{position:absolute;top:0;bottom:0;left:0;width:7rpx;background:#347ff0;content:""}.cascade-option__minor{margin-top:5rpx;color:#8b96a5;font-size:17rpx;font-weight:400}.cascade-option__arrow{color:#91a0b1;font-size:28rpx}.cascade-option--item{min-height:96rpx}.column-empty{display:block;padding:30rpx 16rpx;color:#9aa4b2;font-size:19rpx;line-height:1.5;text-align:center}.column-empty--error{color:#c15a69}.column-action{width:calc(100% - 20rpx);margin:12rpx 10rpx 0;padding:0 8rpx;color:#347ff0;font-size:19rpx;line-height:58rpx;background:#edf4ff;border-radius:10rpx}.management-actions{display:flex;gap:16rpx;padding:22rpx}.management-actions button{flex:1;margin:0;color:#607087;font-size:22rpx;line-height:72rpx;background:#fff;border:1rpx solid #dfe5ec;border-radius:16rpx}.management-actions .management-actions__primary{color:#fff;background:#347ff0;border-color:#347ff0}.page-note{display:block;padding:0 28rpx;color:#8b96a5;font-size:19rpx;line-height:1.6;text-align:center}.page-state{display:flex;flex-direction:column;align-items:center;gap:20rpx;margin:22rpx;padding:80rpx 30rpx;color:#7d899b;font-size:24rpx;background:#fff;border-radius:20rpx}.page-state--error{color:#c34f61}.page-state button{margin:0;padding:0 24rpx;color:#347ff0;font-size:21rpx;line-height:60rpx;background:#edf4ff;border-radius:30rpx}
</style>
