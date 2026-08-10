<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { staffManagementApi } from "@/api/staffManagement";
import DepartmentDoctorPanel from "@/components/staff/DepartmentDoctorPanel.vue";
import DepartmentEditorDialog from "@/components/staff/DepartmentEditorDialog.vue";
import DepartmentSidebar from "@/components/staff/DepartmentSidebar.vue";
import { sessionState } from "@/stores/session";
import type {
  CampusSummary,
  DepartmentSummary,
  DoctorSummary,
} from "@/types/staffManagement";
import { hasIdentityPermission } from "@/utils/appShell";
import {
  invalidateDepartments,
  invalidateDoctors,
  loadDepartments,
  loadDoctors,
} from "@/utils/staffManagementCache";
import { createEmptyDepartment } from "@/utils/staffManagementView";

let lastSelectedDepartmentId = "";
let lastSelectedCampusId = "";
const emptyDepartment = createEmptyDepartment();

const isStaffApp = computed(() => sessionState.appVariant === "staff");
const isSuperAdmin = computed(
  () => sessionState.principal?.roles.includes("super_admin") ?? false,
);
const canManageDepartments = computed(
  () =>
    isStaffApp.value &&
    isSuperAdmin.value &&
    hasIdentityPermission(sessionState.principal, "identity.department.manage"),
);
const canOpenUserManagement = computed(
  () =>
    isStaffApp.value &&
    isSuperAdmin.value &&
    (hasIdentityPermission(
      sessionState.principal,
      "identity.authorization.manage",
    ) ||
      hasIdentityPermission(sessionState.principal, "identity.account.manage")),
);

const departments = ref<DepartmentSummary[]>([]);
const hospitalId = ref("");
const hospitalName = ref("");
const campuses = ref<CampusSummary[]>([]);
const selectedCampusId = ref("");
const doctors = ref<DoctorSummary[]>([]);
const selectedDepartmentId = ref("");
const showDisabled = ref(false);
const departmentLoading = ref(false);
const doctorLoading = ref(false);
const departmentError = ref("");
const doctorError = ref("");
const navigationPending = ref(false);
const mutationPending = ref(false);
const editorVisible = ref(false);
const editorMode = ref<"create" | "edit">("create");
const editorTarget = ref<"campus" | "department">("department");
const editorName = ref("");
let doctorRequestGeneration = 0;

const selectedDepartment = computed(() =>
  departments.value.find(
    (department) => department.departmentId === selectedDepartmentId.value,
  ) ?? emptyDepartment,
);

const visibleDepartments = computed(() =>
  showDisabled.value
    ? departments.value.filter((department) => department.status === "disabled")
    : departments.value.filter((department) => department.status === "active"),
);

onShow(() => {
  navigationPending.value = false;
  uni.setNavigationBarTitle({ title: isStaffApp.value ? "部门管理" : "挂号" });
  void refreshOrganization(false);
});

async function refreshOrganization(force = false) {
  if (departmentLoading.value) {
    return;
  }
  departmentLoading.value = true;
  departmentError.value = "";
  try {
    const context = await staffManagementApi.getOrganizationContext();
    hospitalId.value = context.hospital.hospitalId;
    hospitalName.value = context.hospital.name;
    campuses.value = context.campuses;
    const nextCampus =
      context.campuses.find((campus) => campus.campusId === selectedCampusId.value) ??
      context.campuses.find((campus) => campus.campusId === lastSelectedCampusId) ??
      context.campuses[0];
    selectedCampusId.value = nextCampus?.campusId ?? "";
    lastSelectedCampusId = selectedCampusId.value;
    departmentLoading.value = false;
    await refreshDepartments(force);
  } catch (error) {
    departmentError.value = messageOf(error, "组织信息加载失败，请重试");
    departmentLoading.value = false;
  }
}

async function refreshDepartments(force = false) {
  if (departmentLoading.value) {
    return;
  }
  departmentLoading.value = true;
  departmentError.value = "";
  try {
    if (!selectedCampusId.value) {
      departments.value = [];
      selectedDepartmentId.value = "";
      doctors.value = [];
      return;
    }
    departments.value = await loadDepartments(
      selectedCampusId.value,
      canManageDepartments.value && showDisabled.value,
      force,
    );
    const choices = visibleDepartments.value;
    const principalDepartment = sessionState.principal?.department_id;
    const nextSelected =
      choices.find((item) => item.departmentId === selectedDepartmentId.value) ??
      choices.find((item) => item.departmentId === lastSelectedDepartmentId) ??
      choices.find((item) => item.departmentId === principalDepartment) ??
      choices[0];
    selectedDepartmentId.value = nextSelected?.departmentId ?? "";
    if (nextSelected?.status === "active") {
      await refreshDoctors(nextSelected.departmentId, force);
    } else {
      doctors.value = [];
    }
  } catch (error) {
    departmentError.value = messageOf(error, "部门加载失败，请重试");
  } finally {
    departmentLoading.value = false;
  }
}

function selectCampus(campus: CampusSummary) {
  if (campus.campusId === selectedCampusId.value) {
    return;
  }
  selectedCampusId.value = campus.campusId;
  lastSelectedCampusId = campus.campusId;
  selectedDepartmentId.value = "";
  doctors.value = [];
  void refreshDepartments(false);
}

async function refreshDoctors(departmentId: string, force = false) {
  const generation = ++doctorRequestGeneration;
  doctorLoading.value = true;
  doctorError.value = "";
  try {
    const result = await loadDoctors(departmentId, force);
    if (generation === doctorRequestGeneration && selectedDepartmentId.value === departmentId) {
      doctors.value = result;
    }
  } catch (error) {
    if (generation === doctorRequestGeneration) {
      doctorError.value = messageOf(error, "医生列表加载失败，请重试");
      doctors.value = [];
    }
  } finally {
    if (generation === doctorRequestGeneration) {
      doctorLoading.value = false;
    }
  }
}

function selectDepartment(department: DepartmentSummary) {
  if (selectedDepartmentId.value === department.departmentId) {
    return;
  }
  selectedDepartmentId.value = department.departmentId;
  lastSelectedDepartmentId = department.departmentId;
  doctors.value = [];
  doctorError.value = "";
  if (department.status === "active") {
    void refreshDoctors(department.departmentId);
  }
}

function toggleDepartmentView() {
  showDisabled.value = !showDisabled.value;
  selectedDepartmentId.value = "";
  doctors.value = [];
  void refreshDepartments(true);
}

function openCreateCampus() {
  if (!hospitalId.value) {
    uni.showToast({ title: "医院信息尚未加载", icon: "none" });
    return;
  }
  editorTarget.value = "campus";
  editorMode.value = "create";
  editorName.value = "";
  editorVisible.value = true;
}

function openCreateDepartment() {
  if (!selectedCampusId.value) {
    uni.showToast({ title: "请先新增并选择院区", icon: "none" });
    return;
  }
  editorTarget.value = "department";
  editorMode.value = "create";
  editorName.value = "";
  editorVisible.value = true;
}

function openEditDepartment() {
  if (!selectedDepartment.value.departmentId) {
    return;
  }
  editorTarget.value = "department";
  editorMode.value = "edit";
  editorName.value = selectedDepartment.value.name;
  editorVisible.value = true;
}

async function saveDepartment() {
  const name = editorName.value.trim();
  if (!name || mutationPending.value) {
    if (!name) {
      uni.showToast({ title: "请输入部门名称", icon: "none" });
    }
    return;
  }
  mutationPending.value = true;
  try {
    if (editorTarget.value === "campus") {
      const created = await staffManagementApi.createCampus({
        name,
        hospitalId: hospitalId.value,
      });
      selectedCampusId.value = created.campusId;
      lastSelectedCampusId = created.campusId;
      editorVisible.value = false;
      invalidateDepartments();
      await refreshOrganization(true);
      uni.showToast({ title: "院区已新增" });
      return;
    }
    if (editorMode.value === "create") {
      const created = await staffManagementApi.createDepartment({
        name,
        parentId: selectedCampusId.value,
      });
      lastSelectedDepartmentId = created.departmentId;
    } else if (selectedDepartment.value.departmentId) {
      await staffManagementApi.updateDepartment(
        selectedDepartment.value.departmentId,
        { name, parentId: selectedCampusId.value },
        selectedDepartment.value.version,
      );
      lastSelectedDepartmentId = selectedDepartment.value.departmentId;
    }
    editorVisible.value = false;
    invalidateDepartments();
    await refreshDepartments(true);
    uni.showToast({ title: editorMode.value === "create" ? "部门已新增" : "部门已更新" });
  } catch (error) {
    uni.showToast({ title: messageOf(error, "保存失败"), icon: "none" });
  } finally {
    mutationPending.value = false;
  }
}

function changeDepartmentStatus(enabled: boolean) {
  const department = selectedDepartment.value;
  if (!department.departmentId || mutationPending.value) {
    return;
  }
  uni.showModal({
    title: enabled ? "恢复部门" : "停用部门",
    content: enabled
      ? `恢复“${department.name}”后可重新关联医生和业务。`
      : `“${department.name}”将停止新增业务，历史数据不会删除。`,
    confirmText: enabled ? "确认恢复" : "确认停用",
    confirmColor: enabled ? "#1497e3" : "#d9485f",
    success: async (result) => {
      if (!result.confirm) {
        return;
      }
      mutationPending.value = true;
      try {
        await staffManagementApi.setDepartmentEnabled(
          department.departmentId,
          enabled,
          department.version,
        );
        invalidateDepartments();
        invalidateDoctors(department.departmentId);
        selectedDepartmentId.value = "";
        await refreshDepartments(true);
        uni.showToast({ title: enabled ? "部门已恢复" : "部门已停用" });
      } catch (error) {
        uni.showToast({ title: messageOf(error, "操作失败"), icon: "none" });
      } finally {
        mutationPending.value = false;
      }
    },
  });
}

function openUserManagement() {
  if (!canOpenUserManagement.value || navigationPending.value) {
    return;
  }
  navigationPending.value = true;
  uni.navigateTo({
    url: "/pages/admin/users/index",
    fail: () => {
      navigationPending.value = false;
      uni.showToast({ title: "用户管理打开失败", icon: "none" });
    },
  });
}

function openDoctor(doctor: DoctorSummary) {
  if (!canOpenUserManagement.value || navigationPending.value) {
    return;
  }
  navigationPending.value = true;
  uni.navigateTo({
    url: `/pages/admin/users/detail?account_id=${encodeURIComponent(doctor.doctorId)}`,
    fail: () => {
      navigationPending.value = false;
      uni.showToast({ title: "用户详情打开失败", icon: "none" });
    },
  });
}

function retryDoctors(departmentId: string) {
  void refreshDoctors(departmentId, true);
}

function messageOf(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
</script>

<template>
  <view class="department-page">
    <view class="department-page__header">
      <view>
        <view class="department-page__title-row">
          <text class="department-page__title">科室与医生</text>
        </view>
        <text class="department-page__subtitle">选择院区和科室查看当前医生</text>
      </view>
      <button
        v-if="canOpenUserManagement"
        class="user-management-button"
        @tap="openUserManagement"
      >
        用户管理
      </button>
    </view>

    <view v-if="departmentError" class="page-error">
      <text>{{ departmentError }}</text>
      <button @tap="refreshOrganization(true)">重新加载</button>
    </view>

    <template v-else>
      <view class="organization-bar">
        <view class="organization-bar__hospital">
          <text class="organization-bar__label">医院</text>
          <text class="organization-bar__name">{{ hospitalName }}</text>
        </view>
        <scroll-view class="campus-list" scroll-x enable-flex>
          <button
            v-for="campus in campuses"
            :key="campus.campusId"
            class="campus-chip"
            :class="{ 'campus-chip--active': campus.campusId === selectedCampusId }"
            @tap="selectCampus(campus)"
          >
            {{ campus.name }}
          </button>
        </scroll-view>
        <button
          v-if="canManageDepartments"
          class="add-campus-button"
          @tap="openCreateCampus"
        >
          ＋ 院区
        </button>
      </view>

      <view v-if="campuses.length === 0" class="campus-empty">
        <text>当前还没有院区</text>
        <text class="campus-empty__hint">新增院区后，才能在该院区下新增科室。</text>
        <button v-if="canManageDepartments" @tap="openCreateCampus">新增第一个院区</button>
      </view>

      <view v-else class="department-board">
      <DepartmentSidebar
        :departments="visibleDepartments"
        :selected-id="selectedDepartmentId"
        :loading="departmentLoading"
        :show-disabled="showDisabled"
        :can-manage="canManageDepartments && Boolean(selectedCampusId)"
        @select="selectDepartment"
        @toggle-view="toggleDepartmentView"
        @create="openCreateDepartment"
      />
      <DepartmentDoctorPanel
        :department="selectedDepartment"
        :doctors="doctors"
        :loading="doctorLoading"
        :error="doctorError"
        :can-manage="canManageDepartments"
        :can-open-doctor="canOpenUserManagement"
        @edit="openEditDepartment"
        @set-enabled="changeDepartmentStatus"
        @retry="retryDoctors"
        @open-doctor="openDoctor"
      />
      </view>
    </template>

    <DepartmentEditorDialog
      :visible="editorVisible"
      :mode="editorMode"
      :unit-label="editorTarget === 'campus' ? '院区' : '科室'"
      :name="editorName"
      :pending="mutationPending"
      @update:name="editorName = $event"
      @close="editorVisible = false"
      @save="saveDepartment"
    />
  </view>
</template>

<style scoped>
button::after {
  display: none;
}

.department-page {
  min-height: 100vh;
  padding: 30rpx 24rpx 28rpx;
  box-sizing: border-box;
  background: #f3f6fa;
}

.department-page__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20rpx;
  margin-bottom: 24rpx;
}

.department-page__title-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.department-page__title {
  color: #172033;
  font-size: 40rpx;
  font-weight: 700;
}

.department-page__subtitle {
  display: block;
  margin-top: 8rpx;
  color: #8792a5;
  font-size: 24rpx;
}

.user-management-button {
  flex: 0 0 auto;
  margin: 0;
  padding: 0 24rpx;
  color: #ffffff;
  font-size: 25rpx;
  line-height: 64rpx;
  background: linear-gradient(135deg, #168ee1, #3a72e8);
  border-radius: 32rpx;
  box-shadow: 0 10rpx 24rpx rgba(31, 124, 218, 0.2);
}

.organization-bar {
  display: flex;
  align-items: center;
  gap: 18rpx;
  margin-bottom: 20rpx;
  padding: 18rpx 20rpx;
  background: #ffffff;
  border: 1rpx solid #e8edf4;
  border-radius: 20rpx;
}

.organization-bar__hospital {
  flex: 0 0 auto;
  padding-right: 18rpx;
  border-right: 1rpx solid #e8edf4;
}

.organization-bar__label,
.organization-bar__name {
  display: block;
}

.organization-bar__label {
  color: #98a2b2;
  font-size: 19rpx;
}

.organization-bar__name {
  margin-top: 4rpx;
  color: #344055;
  font-size: 24rpx;
  font-weight: 650;
}

.campus-list {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
}

.campus-chip,
.add-campus-button {
  display: inline-block;
  width: auto;
  margin: 0 12rpx 0 0;
  padding: 0 22rpx;
  font-size: 23rpx;
  line-height: 58rpx;
  border-radius: 29rpx;
}

.campus-chip {
  color: #687488;
  background: #f2f5f8;
}

.campus-chip--active {
  color: #ffffff;
  background: #168bd8;
}

.add-campus-button {
  flex: 0 0 auto;
  color: #167ac3;
  background: #eaf5ff;
}

.campus-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18rpx;
  padding: 100rpx 30rpx;
  color: #536078;
  font-size: 28rpx;
  background: #ffffff;
  border-radius: 22rpx;
}

.campus-empty__hint {
  color: #98a2b2;
  font-size: 23rpx;
}

.campus-empty button {
  margin: 10rpx 0 0;
  padding: 0 30rpx;
  color: #ffffff;
  font-size: 24rpx;
  line-height: 66rpx;
  background: #168bd8;
  border-radius: 33rpx;
}

.department-board {
  display: flex;
  height: calc(100vh - 290rpx);
  min-height: 720rpx;
  overflow: hidden;
  background: #ffffff;
  border: 1rpx solid #e8edf4;
  border-radius: 24rpx;
  box-shadow: 0 12rpx 34rpx rgba(43, 62, 89, 0.07);
}

.page-error button {
  margin: 0;
  color: #1683d0;
  font-size: 23rpx;
  line-height: 58rpx;
  background: #edf7ff;
  border-radius: 28rpx;
}

.page-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24rpx;
  padding: 80rpx 30rpx;
  color: #c44f61;
  font-size: 25rpx;
  background: #ffffff;
  border-radius: 22rpx;
}
</style>
