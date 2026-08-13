<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import {
  loadAppointmentRooms,
  loadRoomExaminationItems,
} from "@/services/appointment";
import {
  loadDepartmentOptions,
  loadDepartments,
  loadOrganizationContext,
} from "@/services/organization";
import { sessionState } from "@/stores/session";
import type { AppointmentRoom, RoomExaminationItem } from "@/types/appointment";
import type { CampusSummary, DepartmentSummary } from "@/types/staffManagement";
import {
  canReadAppointmentManagement,
  hasPermission,
  hasRole,
  messageOf,
} from "@/utils/appointmentManagement";

interface SearchInputEvent {
  detail: { value?: string };
}

let rememberedCampusId = "";
let rememberedDepartmentId = "";
let rememberedRoomId = "";
const hospitalName = ref("");
const campuses = ref<CampusSummary[]>([]);
const selectedCampusId = ref("");
const departments = ref<DepartmentSummary[]>([]);
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
const roomError = ref("");
const relationError = ref("");
const navigationPending = ref(false);
let roomGeneration = 0;
let relationGeneration = 0;
let loadedOnce = false;

const principal = computed(() => sessionState.principal);
const allowed = computed(() => canReadAppointmentManagement(principal.value));
const isDoctor = computed(() => hasRole(principal.value, "department_doctor"));
const canCreate = computed(() => hasPermission(principal.value, "appointment.create"));
const currentCampus = computed(() =>
  campuses.value.find((campus) => campus.campusId === selectedCampusId.value),
);
const selectedDepartment = computed(() =>
  departments.value.find(
    (department) => department.departmentId === selectedDepartmentId.value,
  ),
);
const selectedRoom = computed(() =>
  rooms.value.find((room) => room.roomId === selectedRoomId.value),
);
const filteredDepartments = computed(() => {
  const keyword = departmentSearch.value.trim().toLowerCase();
  if (!keyword) return departments.value;
  return departments.value.filter((department) =>
    department.name.toLowerCase().includes(keyword),
  );
});
const canLoadMore = computed(() => rooms.value.length < totalRooms.value);

onShow(() => {
  navigationPending.value = false;
  uni.setNavigationBarTitle({ title: "检查资源管理" });
  void initialize(loadedOnce);
});

async function initialize(force: boolean) {
  if (!allowed.value) {
    error.value = "当前账号没有检查资源管理权限";
    return;
  }
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    const context = await loadOrganizationContext(force);
    hospitalName.value = context.hospital.name;

    if (isDoctor.value) {
      const ownDepartmentId = principal.value?.department_id ?? "";
      const option = (await loadDepartmentOptions(force)).find(
        (value) => value.department.departmentId === ownDepartmentId,
      );
      campuses.value = option ? [option.campus] : [];
      selectedCampusId.value = option?.campus.campusId ?? "";
      departments.value = option ? [option.department] : [];
      selectedDepartmentId.value = option?.department.departmentId ?? "";
    } else {
      campuses.value = context.campuses.filter((campus) => campus.status === "active");
      const nextCampus = campuses.value.find(
        (campus) => campus.campusId === selectedCampusId.value,
      ) ?? campuses.value.find(
        (campus) => campus.campusId === rememberedCampusId,
      ) ?? campuses.value[0];
      selectedCampusId.value = nextCampus?.campusId ?? "";
      rememberedCampusId = selectedCampusId.value;
      await refreshDepartments(force);
    }

    await loadRooms(force);
    loadedOnce = true;
  } catch (cause) {
    error.value = messageOf(cause, "院区和科室加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function refreshDepartments(force: boolean) {
  if (!selectedCampusId.value) {
    departments.value = [];
    selectedDepartmentId.value = "";
    return;
  }
  departments.value = await loadDepartments(selectedCampusId.value, false, force);
  const nextDepartment = departments.value.find(
    (department) => department.departmentId === selectedDepartmentId.value,
  ) ?? departments.value.find(
    (department) => department.departmentId === rememberedDepartmentId,
  ) ?? departments.value[0];
  selectedDepartmentId.value = nextDepartment?.departmentId ?? "";
  rememberedDepartmentId = selectedDepartmentId.value;
}

async function loadRooms(force = false) {
  const departmentId = selectedDepartmentId.value;
  const generation = ++roomGeneration;
  page.value = 1;
  rooms.value = [];
  selectedRoomId.value = "";
  relations.value = [];
  roomError.value = "";
  relationError.value = "";
  if (!departmentId) return;
  loadingRooms.value = true;
  try {
    const result = await loadAppointmentRooms(departmentId, 1, force);
    if (generation !== roomGeneration) return;
    rooms.value = result.items;
    totalRooms.value = result.total;
    const nextRoom = rooms.value.find((room) => room.roomId === rememberedRoomId)
      ?? rooms.value[0];
    selectedRoomId.value = nextRoom?.roomId ?? "";
    rememberedRoomId = selectedRoomId.value;
    await loadRelations(force);
  } catch (cause) {
    if (generation === roomGeneration) {
      roomError.value = messageOf(cause, "房间加载失败");
    }
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
    const result = await loadRoomExaminationItems(roomId, "active", force);
    if (generation !== relationGeneration) return;
    relations.value = result.items;
  } catch (cause) {
    if (generation === relationGeneration) {
      relationError.value = messageOf(cause, "房间项目加载失败");
    }
  } finally {
    if (generation === relationGeneration) loadingRelations.value = false;
  }
}

function switchCampus() {
  if (isDoctor.value || campuses.value.length <= 1) return;
  uni.showActionSheet({
    title: "选择院区",
    itemList: campuses.value.map((campus) => campus.name),
    success: (result) => {
      const campus = campuses.value[result.tapIndex];
      if (!campus || campus.campusId === selectedCampusId.value) return;
      selectedCampusId.value = campus.campusId;
      rememberedCampusId = campus.campusId;
      selectedDepartmentId.value = "";
      rememberedDepartmentId = "";
      departmentSearch.value = "";
      void changeCampus();
    },
  });
}

async function changeCampus() {
  loading.value = true;
  error.value = "";
  try {
    await refreshDepartments(false);
    rememberedRoomId = "";
    await loadRooms(false);
  } catch (cause) {
    error.value = messageOf(cause, "院区科室加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

function chooseDepartment(department: DepartmentSummary) {
  if (department.departmentId === selectedDepartmentId.value) return;
  selectedDepartmentId.value = department.departmentId;
  rememberedDepartmentId = department.departmentId;
  rememberedRoomId = "";
  void loadRooms(false);
}

function chooseRoom(room: AppointmentRoom) {
  if (room.roomId === selectedRoomId.value) return;
  selectedRoomId.value = room.roomId;
  rememberedRoomId = room.roomId;
  void loadRelations(false);
}

function updateSearch(event: Event) {
  departmentSearch.value = (event as unknown as SearchInputEvent).detail.value ?? "";
}

async function loadMoreRooms() {
  if (!canLoadMore.value || loadingMore.value) return;
  loadingMore.value = true;
  const nextPage = page.value + 1;
  try {
    const result = await loadAppointmentRooms(selectedDepartmentId.value, nextPage);
    rooms.value.push(...result.items);
    page.value = nextPage;
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "加载更多房间失败"), icon: "none" });
  } finally {
    loadingMore.value = false;
  }
}

function openItem(item?: RoomExaminationItem) {
  const label = `${currentCampus.value?.name ?? "院区"} / ${selectedDepartment.value?.name ?? "所属科室"}`;
  navigate(`/pages/admin/appointment/item-detail?department_id=${encodeURIComponent(selectedDepartmentId.value)}&item_id=${encodeURIComponent(item?.itemId ?? "")}&department_label=${encodeURIComponent(label)}`);
}

function openRoom(room?: AppointmentRoom) {
  const label = `${currentCampus.value?.name ?? "院区"} / ${selectedDepartment.value?.name ?? "所属科室"}`;
  navigate(`/pages/admin/appointment/room-detail?department_id=${encodeURIComponent(selectedDepartmentId.value)}&campus_id=${encodeURIComponent(selectedCampusId.value)}&room_id=${encodeURIComponent(room?.roomId ?? "")}&department_label=${encodeURIComponent(label)}`);
}

function openBookings() {
  const label = `${currentCampus.value?.name ?? "院区"} / ${selectedDepartment.value?.name ?? "所属科室"}`;
  navigate(`/pages/admin/appointment/bookings?department_id=${encodeURIComponent(selectedDepartmentId.value)}&department_label=${encodeURIComponent(label)}`);
}

function navigate(url: string) {
  if (navigationPending.value || !selectedDepartmentId.value) return;
  navigationPending.value = true;
  uni.navigateTo({
    url,
    fail: () => uni.showToast({ title: "页面打开失败", icon: "none" }),
    complete: () => { navigationPending.value = false; },
  });
}
</script>

<template>
  <view class="resource-page">
    <view class="search-field">
      <view class="search-field__icon" />
      <input
        :value="departmentSearch"
        placeholder="搜索当前院区科室"
        placeholder-class="search-field__placeholder"
        @input="updateSearch"
      />
      <button v-if="selectedDepartmentId" class="search-field__booking" @tap="openBookings">预约</button>
    </view>

    <view class="campus-bar">
      <view class="campus-bar__copy">
        <text class="campus-bar__label">当前院区</text>
        <text class="campus-bar__name">{{ currentCampus?.name || "暂无可用院区" }}</text>
        <text class="campus-bar__hospital">{{ hospitalName }}</text>
      </view>
      <button
        v-if="!isDoctor && campuses.length > 1"
        class="campus-bar__switch"
        @tap="switchCampus"
      >
        切换院区 <text>⌄</text>
      </button>
    </view>

    <view v-if="error" class="page-state page-state--error">
      <text>{{ error }}</text>
      <button @tap="initialize(true)">重新加载</button>
    </view>

    <view v-else class="workspace">
      <view class="workspace-column workspace-column--departments">
        <view class="column-heading">
          <text class="column-heading__title">科室</text>
          <text class="column-heading__count">{{ filteredDepartments.length }}</text>
        </view>
        <scroll-view scroll-y class="column-body">
          <button
            v-for="department in filteredDepartments"
            :key="department.departmentId"
            class="department-row"
            :class="{ 'department-row--active': department.departmentId === selectedDepartmentId }"
            @tap="chooseDepartment(department)"
          >
            <text class="department-row__name">{{ department.name }}</text>
            <text class="department-row__minor">{{ department.doctorCount }} 位医生</text>
          </button>
          <text v-if="!loading && !filteredDepartments.length" class="column-empty">
            {{ departmentSearch ? "没有匹配科室" : "当前院区暂无科室" }}
          </text>
        </scroll-view>
      </view>

      <view class="workspace-column workspace-column--rooms">
        <view class="column-heading">
          <view>
            <text class="column-heading__title">房间</text>
            <text class="column-heading__count">{{ rooms.length }}</text>
          </view>
          <button v-if="canCreate && selectedDepartmentId" class="column-add" @tap="openRoom()">新增</button>
        </view>
        <scroll-view scroll-y class="column-body">
          <text v-if="loadingRooms" class="column-empty">加载中...</text>
          <button
            v-for="room in rooms"
            :key="room.roomId"
            class="resource-row"
            :class="{ 'resource-row--active': room.roomId === selectedRoomId }"
            @tap="chooseRoom(room)"
          >
            <text class="resource-row__name">{{ room.displayName }}</text>
            <text class="resource-row__minor">{{ room.building }} · {{ room.floorNumber }} 层</text>
          </button>
          <button
            v-if="canLoadMore"
            class="column-action"
            :disabled="loadingMore"
            @tap="loadMoreRooms"
          >
            {{ loadingMore ? "加载中..." : "更多房间" }}
          </button>
          <text v-if="roomError && !rooms.length" class="column-empty column-empty--error">
            {{ roomError }}
          </text>
          <view v-else-if="!loadingRooms && !rooms.length" class="empty-state">
            <text class="column-empty">当前部门暂无房间</text>
            <button v-if="canCreate" class="empty-state__action" @tap="openRoom()">
              新增房间
            </button>
          </view>
        </scroll-view>
      </view>

      <view class="workspace-column workspace-column--items">
        <view class="column-heading">
          <view>
            <text class="column-heading__title">项目</text>
            <text class="column-heading__count">{{ relations.length }}</text>
          </view>
          <button v-if="canCreate && selectedDepartmentId" class="column-add" @tap="openItem()">新增</button>
        </view>
        <scroll-view scroll-y class="column-body">
          <text v-if="loadingRelations" class="column-empty">加载中...</text>
          <button
            v-for="relation in relations"
            :key="relation.relationId"
            class="resource-row"
            @tap="openItem(relation)"
          >
            <text class="resource-row__name">{{ relation.itemName }}</text>
            <text class="resource-row__minor">当前房间可执行</text>
          </button>
          <text v-if="relationError" class="column-empty column-empty--error">{{ relationError }}</text>
          <text v-else-if="selectedRoom && !loadingRelations && !relations.length" class="column-empty">
            当前房间尚未配置项目
          </text>
          <text v-else-if="!selectedRoom && !loadingRooms" class="column-empty">
            {{ roomError ? "房间加载成功后显示项目" : "新增房间后配置项目" }}
          </text>
          <button
            v-if="selectedRoom"
            class="column-action column-action--primary"
            @tap="openRoom(selectedRoom)"
          >
            管理当前房间
          </button>
        </scroll-view>
      </view>
    </view>
  </view>
</template>

<style scoped>
button::after { display: none; }
.resource-page { display: flex; height: 100vh; min-height: 0; flex-direction: column; padding: 20rpx 20rpx env(safe-area-inset-bottom); overflow: hidden; box-sizing: border-box; background: #f3f5f8; }
.search-field { display: flex; flex: 0 0 auto; align-items: center; gap: 16rpx; height: 74rpx; padding: 0 24rpx; box-sizing: border-box; background: #fff; border: 1rpx solid #e1e6ed; border-radius: 20rpx; box-shadow: 0 6rpx 20rpx rgba(32,45,64,.04); }
.search-field__icon { position: relative; width: 22rpx; height: 22rpx; flex: 0 0 auto; border: 3rpx solid #8793a4; border-radius: 50%; }
.search-field__icon::after { position: absolute; right: -8rpx; bottom: -5rpx; width: 10rpx; height: 3rpx; content: ""; background: #8793a4; border-radius: 2rpx; transform: rotate(45deg); }
.search-field input { flex: 1; height: 100%; color: #253146; font-size: 25rpx; }
.search-field__booking { flex: 0 0 auto; padding: 0 20rpx; margin: 0; color: #fff; font-size: 20rpx; line-height: 50rpx; background: #2188c7; border-radius: 25rpx; }
.search-field__placeholder { color: #9ca6b4; }
.campus-bar { display: flex; flex: 0 0 auto; align-items: center; justify-content: space-between; gap: 16rpx; min-height: 98rpx; margin-top: 16rpx; padding: 18rpx 22rpx; box-sizing: border-box; background: #fff; border: 1rpx solid #e1e6ed; border-radius: 20rpx; box-shadow: 0 6rpx 20rpx rgba(32,45,64,.04); }
.campus-bar__copy { min-width: 0; }
.campus-bar__label,.campus-bar__name,.campus-bar__hospital { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.campus-bar__label { color: #929dad; font-size: 18rpx; }
.campus-bar__name { margin-top: 4rpx; color: #202c3f; font-size: 28rpx; font-weight: 700; }
.campus-bar__hospital { margin-top: 3rpx; color: #8a95a5; font-size: 18rpx; }
.campus-bar__switch { flex: 0 0 auto; margin: 0; padding: 0 18rpx; color: #177dbb; font-size: 20rpx; line-height: 52rpx; background: #edf6fb; border-radius: 26rpx; }
.workspace { display: grid; min-height: 0; flex: 1; grid-template-columns: 190rpx 220rpx minmax(0,1fr); margin-top: 16rpx; overflow: hidden; background: #fff; border: 1rpx solid #dfe5ec; border-radius: 20rpx; box-shadow: 0 8rpx 24rpx rgba(32,45,64,.045); }
.workspace-column { display: flex; min-width: 0; min-height: 0; flex-direction: column; border-right: 1rpx solid #e7ebf0; }
.workspace-column:last-child { border-right: 0; }
.workspace-column--departments { background: #f6f8fa; }
.column-heading { display: flex; flex: 0 0 auto; align-items: center; justify-content: space-between; min-height: 68rpx; padding: 0 16rpx; box-sizing: border-box; background: rgba(255,255,255,.94); border-bottom: 1rpx solid #e7ebf0; }
.column-heading__title { color: #46546a; font-size: 21rpx; font-weight: 700; }
.column-heading__count { margin-left: 8rpx; color: #98a3b2; font-size: 18rpx; }
.column-add { display: flex; align-items: center; justify-content: center; min-width: 48rpx; height: 40rpx; margin: 0; padding: 0 9rpx; color: #167fc0; font-size: 17rpx; line-height: 40rpx; background: #edf6fb; border-radius: 12rpx; }
.column-body { flex: 1; height: 0; }
.department-row,.resource-row { width: 100%; margin: 0; padding: 17rpx 16rpx; box-sizing: border-box; color: inherit; text-align: left; background: transparent; border-bottom: 1rpx solid #e9edf2; border-radius: 0; }
.department-row { position: relative; min-height: 84rpx; }
.department-row--active { background: #fff; }
.department-row--active::before { position: absolute; top: 0; bottom: 0; left: 0; width: 6rpx; content: ""; background: #2188c7; }
.department-row--active .department-row__name { color: #177dbb; }
.department-row__name,.department-row__minor,.resource-row__name,.resource-row__minor { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.department-row__name,.resource-row__name { color: #2c384b; font-size: 21rpx; font-weight: 620; }
.department-row__minor,.resource-row__minor { margin-top: 6rpx; color: #939eae; font-size: 17rpx; font-weight: 400; }
.resource-row { position: relative; min-height: 86rpx; background: #fff; }
.resource-row:active { background: #f5faff; }
.resource-row--active { background: #edf7fc; }
.resource-row--active::before { position: absolute; top: 0; bottom: 0; left: 0; width: 5rpx; content: ""; background: #2188c7; }
.resource-row--active .resource-row__name { color: #177dbb; }
.resource-row__title-line { display: flex; align-items: center; gap: 7rpx; min-width: 0; }
.status-tag { flex: 0 0 auto; padding: 2rpx 7rpx; color: #a05d67; font-size: 14rpx; line-height: 1.4; background: #f8e9ec; border-radius: 8rpx; }
.column-action { width: calc(100% - 20rpx); margin: 12rpx 10rpx 0; padding: 0 8rpx; color: #177dbb; font-size: 18rpx; line-height: 54rpx; background: #edf6fb; border-radius: 12rpx; }
.column-action--primary { color: #fff; background: #2188c7; }
.column-empty { display: block; padding: 28rpx 12rpx; color: #929dad; font-size: 18rpx; line-height: 1.5; text-align: center; }
.column-empty--error { color: #bf5363; }
.empty-state { padding: 20rpx 12rpx; text-align: center; }
.empty-state .column-empty { padding: 8rpx 0 18rpx; }
.empty-state__action { width: 100%; margin: 0; padding: 0 8rpx; color: #fff; font-size: 18rpx; line-height: 54rpx; background: #2188c7; border-radius: 12rpx; }
.page-state { display: flex; min-height: 0; flex: 1; flex-direction: column; align-items: center; justify-content: center; gap: 20rpx; margin-top: 16rpx; color: #8793a4; font-size: 22rpx; background: #fff; border: 1rpx solid #e1e6ed; border-radius: 20rpx; }
.page-state--error { color: #bf5363; }
.page-state button { margin: 0; padding: 0 22rpx; color: #167fc0; font-size: 20rpx; line-height: 54rpx; background: #edf6fb; border-radius: 27rpx; }
</style>
