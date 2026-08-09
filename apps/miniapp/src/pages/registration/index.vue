<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { STAFF_MANAGEMENT_USES_MOCK, staffManagementApi } from "@/api/staffManagement";
import DepartmentDoctorPanel from "@/components/staff/DepartmentDoctorPanel.vue";
import DepartmentEditorDialog from "@/components/staff/DepartmentEditorDialog.vue";
import DepartmentSidebar from "@/components/staff/DepartmentSidebar.vue";
import { sessionState } from "@/stores/session";
import type { DepartmentSummary, DoctorSummary } from "@/types/staffManagement";
import { hasIdentityPermission } from "@/utils/appShell";
import {
  invalidateDepartments,
  invalidateDoctors,
  loadDepartments,
  loadDoctors,
} from "@/utils/staffManagementCache";
import { createEmptyDepartment } from "@/utils/staffManagementView";

let lastSelectedDepartmentId = "";
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
  void refreshDepartments(false);
});

async function refreshDepartments(force = false) {
  if (departmentLoading.value) {
    return;
  }
  departmentLoading.value = true;
  departmentError.value = "";
  try {
    departments.value = await loadDepartments(canManageDepartments.value, force);
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
  const next = visibleDepartments.value[0];
  if (next) {
    selectDepartment(next);
  }
}

function openCreateDepartment() {
  editorMode.value = "create";
  editorName.value = "";
  editorVisible.value = true;
}

function openEditDepartment() {
  if (!selectedDepartment.value.departmentId) {
    return;
  }
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
    if (editorMode.value === "create") {
      const created = await staffManagementApi.createDepartment({
        name,
        parentId: selectedDepartment.value?.parentId,
      });
      lastSelectedDepartmentId = created.departmentId;
    } else if (selectedDepartment.value.departmentId) {
      await staffManagementApi.updateDepartment(
        selectedDepartment.value.departmentId,
        { name, parentId: selectedDepartment.value.parentId },
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
          <text v-if="STAFF_MANAGEMENT_USES_MOCK" class="mock-badge">Mock</text>
        </view>
        <text class="department-page__subtitle">选择部门查看当前医生</text>
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
      <button @tap="refreshDepartments(true)">重新加载</button>
    </view>

    <view v-else class="department-board">
      <DepartmentSidebar
        :departments="visibleDepartments"
        :selected-id="selectedDepartmentId"
        :loading="departmentLoading"
        :show-disabled="showDisabled"
        :can-manage="canManageDepartments"
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

    <DepartmentEditorDialog
      :visible="editorVisible"
      :mode="editorMode"
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

.mock-badge {
  padding: 4rpx 10rpx;
  color: #886600;
  font-size: 20rpx;
  background: #fff4c9;
  border-radius: 10rpx;
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

.department-board {
  display: flex;
  height: calc(100vh - 180rpx);
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
