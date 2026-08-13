<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { loadDepartmentOptions, type DepartmentOption } from "@/services/organization";
import {
  loadAppointmentRooms,
  loadExaminationItems,
} from "@/services/appointment";
import { sessionState } from "@/stores/session";
import type {
  AppointmentRoom,
  AppointmentStatus,
  ExaminationItem,
} from "@/types/appointment";
import {
  canReadAppointmentManagement,
  hasPermission,
  hasRole,
  messageOf,
} from "@/utils/appointmentManagement";

type ResourceTab = "items" | "rooms";

let rememberedDepartmentId = "";
const tab = ref<ResourceTab>("items");
const status = ref<AppointmentStatus>("active");
const departments = ref<DepartmentOption[]>([]);
const selectedDepartmentId = ref("");
const items = ref<ExaminationItem[]>([]);
const rooms = ref<AppointmentRoom[]>([]);
const total = ref(0);
const page = ref(1);
const loading = ref(false);
const loadingMore = ref(false);
const error = ref("");
const navigationPending = ref(false);
let requestGeneration = 0;

const principal = computed(() => sessionState.principal);
const allowed = computed(() => canReadAppointmentManagement(principal.value));
const canCreate = computed(() => hasPermission(principal.value, "appointment.create"));
const isDoctor = computed(() => hasRole(principal.value, "department_doctor"));
const selectedDepartment = computed(() =>
  departments.value.find((option) => option.department.departmentId === selectedDepartmentId.value),
);
const records = computed(() => (tab.value === "items" ? items.value : rooms.value));
const canLoadMore = computed(() => records.value.length < total.value);

onShow(() => {
  navigationPending.value = false;
  uni.setNavigationBarTitle({ title: "检查项目管理" });
  void initialize(false);
});

async function initialize(force: boolean) {
  if (!allowed.value) {
    error.value = "当前账号没有检查项目管理权限";
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    const allOptions = await loadDepartmentOptions(force);
    const ownDepartmentId = principal.value?.department_id ?? "";
    departments.value = isDoctor.value
      ? allOptions.filter((option) => option.department.departmentId === ownDepartmentId)
      : allOptions;
    const next = departments.value.find((option) =>
      option.department.departmentId === selectedDepartmentId.value,
    ) ?? departments.value.find((option) =>
      option.department.departmentId === rememberedDepartmentId,
    ) ?? departments.value[0];
    selectedDepartmentId.value = next?.department.departmentId ?? ownDepartmentId;
    rememberedDepartmentId = selectedDepartmentId.value;
    await refresh(true);
  } catch (cause) {
    error.value = messageOf(cause, "管理数据加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function refresh(force = false) {
  const departmentId = selectedDepartmentId.value;
  if (!departmentId) {
    items.value = [];
    rooms.value = [];
    total.value = 0;
    return;
  }
  const generation = ++requestGeneration;
  page.value = 1;
  error.value = "";
  try {
    if (tab.value === "items") {
      const result = await loadExaminationItems(departmentId, status.value, 1, force);
      if (generation !== requestGeneration) return;
      items.value = result.items;
      total.value = result.total;
    } else {
      const result = await loadAppointmentRooms(departmentId, status.value, 1, force);
      if (generation !== requestGeneration) return;
      rooms.value = result.items;
      total.value = result.total;
    }
  } catch (cause) {
    if (generation === requestGeneration) error.value = messageOf(cause, "列表加载失败，请重试");
  }
}

async function loadMore() {
  if (!canLoadMore.value || loadingMore.value) return;
  loadingMore.value = true;
  const nextPage = page.value + 1;
  try {
    if (tab.value === "items") {
      const result = await loadExaminationItems(selectedDepartmentId.value, status.value, nextPage);
      items.value.push(...result.items);
    } else {
      const result = await loadAppointmentRooms(selectedDepartmentId.value, status.value, nextPage);
      rooms.value.push(...result.items);
    }
    page.value = nextPage;
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "加载下一页失败"), icon: "none" });
  } finally {
    loadingMore.value = false;
  }
}

function switchTab(next: ResourceTab) {
  if (tab.value === next) return;
  tab.value = next;
  void refresh(false);
}

function toggleStatus() {
  status.value = status.value === "active" ? "disabled" : "active";
  void refresh(false);
}

function chooseDepartment() {
  if (isDoctor.value || departments.value.length < 2) return;
  uni.showActionSheet({
    title: "选择院区 / 科室",
    itemList: departments.value.map((option) => option.label),
    success: ({ tapIndex }) => {
      const option = departments.value[tapIndex];
      if (!option || option.department.departmentId === selectedDepartmentId.value) return;
      selectedDepartmentId.value = option.department.departmentId;
      rememberedDepartmentId = selectedDepartmentId.value;
      items.value = [];
      rooms.value = [];
      void refresh(false);
    },
  });
}

function openItem(value?: ExaminationItem) {
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
    <view class="hero">
      <text class="hero__eyebrow">APPOINTMENT · 真实数据</text>
      <text class="hero__title">检查项目与房间资源</text>
      <button class="department-button" :disabled="isDoctor" @tap="chooseDepartment">
        <text>{{ selectedDepartment?.label || '正在读取所属科室…' }}</text>
        <text v-if="!isDoctor">切换 ›</text>
      </button>
    </view>

    <view v-if="error && !records.length" class="state state--error">
      <text>{{ error }}</text>
      <button @tap="initialize(true)">重新加载</button>
    </view>
    <template v-else>
      <view class="toolbar">
        <view class="tabs">
          <button :class="{ active: tab === 'items' }" @tap="switchTab('items')">检查项目</button>
          <button :class="{ active: tab === 'rooms' }" @tap="switchTab('rooms')">房间资源</button>
        </view>
        <button class="status-button" @tap="toggleStatus">
          {{ status === 'active' ? '查看已停用' : '查看启用中' }}
        </button>
      </view>

      <view v-if="loading && !records.length" class="state">正在加载真实数据…</view>
      <view v-else-if="!records.length" class="state">
        <text>{{ status === 'active' ? '当前科室还没有可用资源' : '当前没有已停用资源' }}</text>
        <button v-if="canCreate && status === 'active'" @tap="tab === 'items' ? openItem() : openRoom()">
          {{ tab === 'items' ? '新建第一个检查项目' : '新建第一个房间' }}
        </button>
      </view>

      <view v-else class="resource-list">
        <button
          v-for="value in records"
          :key="tab === 'items' ? (value as ExaminationItem).itemId : (value as AppointmentRoom).roomId"
          class="resource-card"
          @tap="tab === 'items' ? openItem(value as ExaminationItem) : openRoom(value as AppointmentRoom)"
        >
          <view class="resource-card__content">
            <text class="resource-card__title">{{ value.name }}</text>
            <text v-if="tab === 'items'" class="resource-card__description">
              {{ (value as ExaminationItem).description || '暂无检查说明' }}
            </text>
            <text v-else class="resource-card__description">进入后维护可执行项目、开放时间和容量</text>
          </view>
          <view class="resource-card__aside">
            <text class="status" :class="`status--${value.status}`">{{ value.status === 'active' ? '启用中' : '已停用' }}</text>
            <text class="arrow">›</text>
          </view>
        </button>
        <button v-if="canLoadMore" class="load-more" :disabled="loadingMore" @tap="loadMore">
          {{ loadingMore ? '加载中…' : `加载更多（${records.length}/${total}）` }}
        </button>
      </view>

      <button v-if="canCreate && status === 'active' && records.length" class="floating-create" @tap="tab === 'items' ? openItem() : openRoom()">
        ＋ {{ tab === 'items' ? '新建项目' : '新建房间' }}
      </button>
    </template>
  </view>
</template>

<style scoped>
button::after { display:none; }
.management-page { min-height:100vh; padding:24rpx 24rpx 150rpx; box-sizing:border-box; background:#f2f6fa; }
.hero { padding:30rpx; color:#fff; background:linear-gradient(135deg,#147fcb,#24adb5); border-radius:28rpx; box-shadow:0 16rpx 34rpx rgba(20,128,190,.2); }
.hero__eyebrow,.hero__title { display:block; }.hero__eyebrow { font-size:20rpx; opacity:.74; letter-spacing:2rpx; }.hero__title { margin-top:10rpx; font-size:36rpx; font-weight:750; }
.department-button { display:flex; justify-content:space-between; width:100%; margin:24rpx 0 0; padding:0 22rpx; color:#eaf8ff; font-size:23rpx; line-height:68rpx; text-align:left; background:rgba(255,255,255,.14); border:1rpx solid rgba(255,255,255,.24); border-radius:18rpx; }
.department-button[disabled] { color:#fff; opacity:1; }
.toolbar { display:flex; align-items:center; justify-content:space-between; gap:16rpx; margin-top:22rpx; }.tabs { display:flex; flex:1; padding:6rpx; background:#e6edf4; border-radius:18rpx; }.tabs button { flex:1; margin:0; color:#6e7b8e; font-size:23rpx; line-height:58rpx; background:transparent; border-radius:14rpx; }.tabs button.active { color:#147fbd; font-weight:700; background:#fff; box-shadow:0 4rpx 12rpx rgba(37,64,94,.08); }.status-button { flex:0 0 auto; margin:0; padding:0 20rpx; color:#607187; font-size:21rpx; line-height:66rpx; background:#fff; border-radius:18rpx; }
.state { display:flex; flex-direction:column; align-items:center; gap:20rpx; margin-top:22rpx; padding:90rpx 30rpx; color:#7d899b; font-size:25rpx; text-align:center; background:#fff; border-radius:24rpx; }.state--error { color:#c34f61; }.state button { margin:0; padding:0 26rpx; color:#1684ca; font-size:22rpx; line-height:62rpx; background:#e9f5fc; border-radius:31rpx; }
.resource-list { margin-top:18rpx; }.resource-card { display:flex; align-items:center; justify-content:space-between; width:100%; margin:14rpx 0 0; padding:24rpx; color:inherit; text-align:left; background:#fff; border:1rpx solid #e6ecf2; border-radius:22rpx; box-shadow:0 8rpx 22rpx rgba(43,62,89,.05); }.resource-card__content { flex:1; min-width:0; }.resource-card__title,.resource-card__description { display:block; }.resource-card__title { color:#263348; font-size:27rpx; font-weight:700; }.resource-card__description { max-width:540rpx; margin-top:9rpx; overflow:hidden; color:#8b96a7; font-size:21rpx; line-height:1.5; text-overflow:ellipsis; white-space:nowrap; }.resource-card__aside { display:flex; align-items:center; gap:12rpx; margin-left:18rpx; }.status { padding:6rpx 12rpx; font-size:18rpx; border-radius:14rpx; }.status--active { color:#128968; background:#e1f7ef; }.status--disabled { color:#9a6870; background:#f6e9eb; }.arrow { color:#9aa6b6; font-size:34rpx; }.load-more { width:100%; margin:18rpx 0 0; color:#1683c6; font-size:22rpx; line-height:68rpx; background:#e9f5fc; border-radius:20rpx; }
.floating-create { position:fixed; right:28rpx; bottom:36rpx; margin:0; padding:0 30rpx; color:#fff; font-size:24rpx; line-height:76rpx; background:linear-gradient(135deg,#168bd7,#1db4b2); border-radius:38rpx; box-shadow:0 14rpx 30rpx rgba(22,139,215,.28); }
</style>
