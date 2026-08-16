<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import {
  loadAppointmentRooms,
  loadExaminationItems,
  loadRoomExaminationItems,
} from "@/services/appointment";
import {
  loadDepartmentOptions,
  loadDepartments,
  loadOrganizationContext,
} from "@/services/organization";
import { sessionState } from "@/stores/session";
import type { AppointmentRoom, ExaminationItem, RoomExaminationItem } from "@/types/appointment";
import type { CampusSummary, DepartmentSummary } from "@/types/staffManagement";
import {
  canReadAppointmentManagement,
  formatEstimatedDuration,
  hasPermission,
  hasRole,
  messageOf,
} from "@/utils/appointmentManagement";
import { currentStaffDepartmentId, rememberStaffDepartmentId } from "@/utils/staffDepartmentContext";

interface SearchInputEvent {
  detail: { value?: string };
}

let rememberedCampusId = "";
let rememberedDepartmentId = currentStaffDepartmentId();
let rememberedRoomId = "";
const hospitalName = ref("");
const campuses = ref<CampusSummary[]>([]);
const selectedCampusId = ref("");
const departments = ref<DepartmentSummary[]>([]);
const selectedDepartmentId = ref("");
const rooms = ref<AppointmentRoom[]>([]);
const examinationItems = ref<ExaminationItem[]>([]);
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
const showingRoomItems = ref(false);
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
const examinationItemById = computed(() => new Map(
  examinationItems.value.map((item) => [item.itemId, item]),
));
const filteredDepartments = computed(() => {
  const keyword = departmentSearch.value.trim().toLowerCase();
  if (!keyword) return departments.value;
  return departments.value.filter((department) =>
    department.name.toLowerCase().includes(keyword),
  );
});
const canLoadMore = computed(() => rooms.value.length < totalRooms.value);
const workspaceClass = computed(() => ({
  "workspace-track--room-items": showingRoomItems.value,
}));

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
      if (selectedDepartmentId.value) rememberStaffDepartmentId(selectedDepartmentId.value);
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
  if (selectedDepartmentId.value) rememberStaffDepartmentId(selectedDepartmentId.value);
}

async function loadRooms(force = false) {
  const departmentId = selectedDepartmentId.value;
  const generation = ++roomGeneration;
  page.value = 1;
  rooms.value = [];
  examinationItems.value = [];
  selectedRoomId.value = "";
  relations.value = [];
  roomError.value = "";
  relationError.value = "";
  if (!departmentId) return;
  loadingRooms.value = true;
  try {
    const [result, itemResult] = await Promise.all([
      loadAppointmentRooms(departmentId, 1, force),
      loadExaminationItems(departmentId, "active", 1, 100, force),
    ]);
    if (generation !== roomGeneration) return;
    rooms.value = result.items;
    examinationItems.value = itemResult.items;
    totalRooms.value = result.total;
    const nextRoom = rooms.value.find((room) => room.roomId === rememberedRoomId)
      ?? rooms.value[0];
    selectedRoomId.value = nextRoom?.roomId ?? "";
    rememberedRoomId = selectedRoomId.value;
    if (!nextRoom) showingRoomItems.value = false;
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
      showingRoomItems.value = false;
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
  showingRoomItems.value = false;
  selectedDepartmentId.value = department.departmentId;
  rememberedDepartmentId = department.departmentId;
  rememberStaffDepartmentId(department.departmentId);
  rememberedRoomId = "";
  void loadRooms(false);
}

function chooseRoom(room: AppointmentRoom) {
  if (room.roomId !== selectedRoomId.value) {
    selectedRoomId.value = room.roomId;
    rememberedRoomId = room.roomId;
    void loadRelations(false);
  }
  showingRoomItems.value = true;
}

function returnToDepartments() {
  showingRoomItems.value = false;
}

function roomCampusName(room: AppointmentRoom) {
  return campuses.value.find((campus) => campus.campusId === room.campusId)?.name
    ?? currentCampus.value?.name
    ?? "未知园区";
}

function floorLabel(floor: number) {
  return floor < 0 ? `B${Math.abs(floor)}层` : `${floor}层`;
}

function relationDuration(relation: RoomExaminationItem) {
  return formatEstimatedDuration(
    examinationItemById.value.get(relation.itemId)?.estimatedDurationMinutes ?? 0,
  );
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

function openItem(item: RoomExaminationItem) {
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

    <view
      class="hierarchy-bar"
      :class="{ 'hierarchy-bar--visible': showingRoomItems }"
    >
      <button class="hierarchy-bar__back" @tap="returnToDepartments">‹ 返回科室</button>
      <view class="hierarchy-bar__breadcrumb">
        <text>{{ selectedDepartment?.name || "所属科室" }}</text>
        <text class="hierarchy-bar__separator">/</text>
        <text>{{ selectedRoom ? `${selectedRoom.roomNumber}室` : "所选房间" }}</text>
      </view>
    </view>

    <view v-if="error" class="page-state page-state--error">
      <text>{{ error }}</text>
      <button @tap="initialize(true)">重新加载</button>
    </view>

    <view v-else class="workspace">
      <view class="workspace-track" :class="workspaceClass">
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
            <view
              v-for="room in rooms"
              :key="room.roomId"
              class="resource-row"
              :class="{ 'resource-row--active': room.roomId === selectedRoomId }"
              @tap="chooseRoom(room)"
            >
              <view class="resource-row__copy">
                <text class="room-card__number">{{ room.roomNumber }}室</text>
                <text class="room-card__address">园区：{{ roomCampusName(room) }}</text>
                <text class="room-card__address">楼栋：{{ room.building }}</text>
                <text class="room-card__address">楼层：{{ floorLabel(room.floorNumber) }}</text>
                <text class="room-card__address">房间号：{{ room.roomNumber }}</text>
              </view>
              <button class="resource-row__detail" @tap.stop="openRoom(room)">编辑 ›</button>
            </view>
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
              <text class="column-heading__title">检查项目</text>
              <text class="column-heading__count">{{ relations.length }}</text>
            </view>
          </view>
          <scroll-view scroll-y class="column-body">
            <text v-if="loadingRelations" class="column-empty">加载中...</text>
            <button
              v-for="relation in relations"
              :key="relation.relationId"
              class="resource-row"
              @tap="openItem(relation)"
            >
              <view class="resource-row__copy">
                <text class="resource-row__name">{{ relation.itemName }}</text>
                <text class="resource-row__duration">{{ relationDuration(relation) }}</text>
              </view>
              <text class="resource-row__detail">编辑 ›</text>
            </button>
            <text v-if="relationError" class="column-empty column-empty--error">{{ relationError }}</text>
            <text v-else-if="selectedRoom && !loadingRelations && !relations.length" class="column-empty">
              当前房间尚未配置项目
            </text>
            <text v-else-if="!selectedRoom && !loadingRooms" class="column-empty">
              {{ roomError ? "房间加载成功后显示项目" : "新增房间后配置项目" }}
            </text>
          </scroll-view>
        </view>
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
.hierarchy-bar { display: flex; max-height: 0; flex: 0 0 auto; align-items: center; gap: 18rpx; margin-top: 0; padding: 0 4rpx; overflow: hidden; box-sizing: border-box; opacity: 0; pointer-events: none; transition: max-height 260ms ease, margin-top 260ms ease, opacity 180ms ease; }
.hierarchy-bar--visible { max-height: 58rpx; margin-top: 12rpx; opacity: 1; pointer-events: auto; }
.hierarchy-bar__back { flex: 0 0 auto; margin: 0; padding: 0; color: #177dbb; font-size: 20rpx; line-height: 50rpx; background: transparent; }
.hierarchy-bar__breadcrumb { display: flex; min-width: 0; align-items: center; gap: 12rpx; color: #35435a; font-size: 21rpx; font-weight: 650; }
.hierarchy-bar__breadcrumb text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.hierarchy-bar__separator { flex: 0 0 auto; color: #9aa4b2; font-weight: 400; }
.workspace { min-height: 0; flex: 1; margin-top: 16rpx; overflow: hidden; background: #fff; border: 1rpx solid #dfe5ec; border-radius: 20rpx; }
.workspace-track { display: grid; width: 150%; height: 100%; grid-template-columns: repeat(3,minmax(0,1fr)); transform: translateX(0); transition: transform 260ms cubic-bezier(.2,0,0,1); }
.workspace-track--room-items { transform: translateX(-33.333333%); }
.workspace-column { display: flex; min-width: 0; min-height: 0; flex-direction: column; box-sizing: border-box; background: #fbfcfd; border-right: 1rpx solid #e3e8ee; transition: opacity 200ms ease, transform 260ms cubic-bezier(.2,0,0,1); }
.workspace-column--departments { background: #f6f8fa; opacity: 1; }
.workspace-column--items { border-right: 0; opacity: 0; transform: translateX(18rpx); }
.workspace-track--room-items .workspace-column--departments { opacity: 0; transform: translateX(-18rpx); }
.workspace-track--room-items .workspace-column--items { opacity: 1; transform: translateX(0); }
.column-heading { display: flex; flex: 0 0 auto; align-items: center; justify-content: space-between; min-height: 76rpx; padding: 0 20rpx; box-sizing: border-box; background: #fff; border-bottom: 1rpx solid #e4e9ef; }
.column-heading__title { color: #405069; font-size: 23rpx; font-weight: 700; }
.column-heading__count { margin-left: 9rpx; padding: 1rpx 8rpx; color: #718097; font-size: 18rpx; background: #f0f3f6; border-radius: 9rpx; }
.column-add { display: flex; align-items: center; justify-content: center; min-width: 56rpx; height: 42rpx; margin: 0; padding: 0 11rpx; color: #167fc0; font-size: 18rpx; line-height: 42rpx; background: #edf6fb; border-radius: 12rpx; }
.column-body { flex: 1; height: 0; }
.department-row,.resource-row { width: calc(100% - 24rpx); margin: 12rpx 12rpx 0; padding: 20rpx; box-sizing: border-box; color: inherit; text-align: left; background: #fff; border: 1rpx solid #e3e8ee; border-radius: 16rpx; }
.department-row { position: relative; min-height: 96rpx; }
.department-row--active { background: #eef7fc; border-color: #b9dcec; }
.department-row--active::before { position: absolute; top: 12rpx; bottom: 12rpx; left: 0; width: 6rpx; content: ""; background: #2188c7; border-radius: 0 5rpx 5rpx 0; }
.department-row--active .department-row__name { color: #177dbb; }
.department-row__name,.department-row__minor,.resource-row__name,.room-card__number { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.department-row__name,.resource-row__name,.room-card__number { color: #29374c; font-size: 23rpx; font-weight: 650; }
.department-row__minor { margin-top: 8rpx; color: #77859a; font-size: 19rpx; font-weight: 400; }
.resource-row { position: relative; display: flex; min-height: 104rpx; align-items: center; gap: 14rpx; }
.workspace-column--rooms .resource-row { min-height: 210rpx; align-items: flex-start; }
.resource-row:active { background: #f5faff; }
.resource-row--active { background: #eef7fc; border-color: #9fcfe5; }
.resource-row--active::before { position: absolute; top: 14rpx; bottom: 14rpx; left: 0; width: 6rpx; content: ""; background: #2188c7; border-radius: 0 5rpx 5rpx 0; }
.resource-row--active .resource-row__name,.resource-row--active .room-card__number { color: #177dbb; }
.resource-row__copy { min-width: 0; flex: 1; }
.resource-row__duration { display: block; margin-top: 10rpx; color: #5e6d82; font-size: 19rpx; line-height: 1.4; }
.resource-row__detail { flex: 0 0 auto; padding: 0; margin: 0; color: #177dbb; font-size: 18rpx; line-height: 44rpx; background: transparent; }
.room-card__number { margin-bottom: 12rpx; }
.room-card__address { display: block; margin-top: 5rpx; color: #5e6d82; font-size: 19rpx; line-height: 1.45; white-space: normal; word-break: break-all; }
.workspace-column--items .resource-row__name { white-space: normal; word-break: break-all; }
.column-action { width: calc(100% - 24rpx); margin: 12rpx; padding: 0 10rpx; color: #177dbb; font-size: 19rpx; line-height: 58rpx; background: #edf6fb; border-radius: 14rpx; }
.column-empty { display: block; padding: 32rpx 18rpx; color: #7f8b9d; font-size: 20rpx; line-height: 1.55; text-align: center; }
.column-empty--error { color: #bf5363; }
.empty-state { padding: 20rpx 16rpx; text-align: center; }
.empty-state .column-empty { padding: 8rpx 0 18rpx; }
.empty-state__action { width: 100%; margin: 0; padding: 0 8rpx; color: #fff; font-size: 19rpx; line-height: 58rpx; background: #2188c7; border-radius: 14rpx; }
.page-state { display: flex; min-height: 0; flex: 1; flex-direction: column; align-items: center; justify-content: center; gap: 20rpx; margin-top: 16rpx; color: #8793a4; font-size: 22rpx; background: #fff; border: 1rpx solid #e1e6ed; border-radius: 20rpx; }
.page-state--error { color: #bf5363; }
.page-state button { margin: 0; padding: 0 22rpx; color: #167fc0; font-size: 20rpx; line-height: 54rpx; background: #edf6fb; border-radius: 27rpx; }
@media (prefers-reduced-motion: reduce) {
  .workspace-track,.workspace-column,.hierarchy-bar { transition-duration: 1ms; }
}
</style>
