<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import {
  identityAdminApi,
} from "@/api/staffManagement";
import DoctorProfileDialog from "@/components/admin/DoctorProfileDialog.vue";
import { sessionState } from "@/stores/session";
import type {
  AdminAccountAction,
  AdminAccountDetail,
  DepartmentSummary,
  DoctorProfileDraft,
} from "@/types/staffManagement";
import { hasIdentityPermission } from "@/utils/appShell";
import {
  invalidateDepartments,
  invalidateDoctors,
  loadDepartments,
  loadOrganizationContext,
} from "@/services/organization";

const identityLabels = { patient: "普通用户", doctor: "医生", super_admin: "超级管理员" } as const;
const accountId = ref("");
const detail = ref<AdminAccountDetail>();
const departments = ref<DepartmentSummary[]>([]);
const loading = ref(true);
const error = ref("");
const pending = ref(false);
const profileDialogVisible = ref(false);
const profileDialogMode = ref<"promote" | "edit">("promote");
const selectedTargetDepartmentId = ref("");

const canManageUsers = computed(
  () =>
    sessionState.principal?.roles.includes("super_admin") === true &&
    (hasIdentityPermission(sessionState.principal, "identity.authorization.manage") ||
      hasIdentityPermission(sessionState.principal, "identity.account.manage")),
);
const displayName = computed(() => detail.value?.displayName || detail.value?.nickname || "未设置昵称");
const initialProfile = computed<DoctorProfileDraft>(() => ({
  displayName: detail.value?.displayName || detail.value?.nickname || "",
  staffNo: detail.value?.staffNo,
  description: detail.value?.description,
}));
const profileDialogTitle = computed(() =>
  profileDialogMode.value === "promote" ? "开通医生身份" : "编辑医生资料",
);

onLoad((query) => {
  accountId.value = typeof query?.account_id === "string" ? decodeURIComponent(query.account_id) : "";
  if (!canManageUsers.value) {
    uni.showToast({ title: "仅超级管理员可访问", icon: "none" });
    setTimeout(() => uni.navigateBack(), 600);
    return;
  }
  if (!accountId.value) {
    error.value = "缺少用户标识";
    loading.value = false;
    return;
  }
  void loadDetail();
});

async function loadDetail() {
  loading.value = true;
  error.value = "";
  try {
    const departmentsPromise = loadOrganizationContext()
      .then((context) =>
        Promise.all(
          context.campuses.map((campus) =>
            loadDepartments(campus.campusId),
          ),
        ),
      )
      .then((items) => items.flat());
    const [account, departmentList] = await Promise.all([
      identityAdminApi.getAccount(accountId.value),
      departmentsPromise,
    ]);
    detail.value = account;
    departments.value = departmentList.filter((item) => item.status === "active");
  } catch (caught) {
    error.value = messageOf(caught, "用户详情加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

function supports(action: AdminAccountAction): boolean {
  return detail.value?.availableActions.includes(action) ?? false;
}

async function selectDepartment(title: string): Promise<DepartmentSummary | null> {
  if (!departments.value.length) {
    uni.showToast({ title: "暂无可用部门", icon: "none" });
    return null;
  }
  return new Promise((resolve) => {
    uni.showActionSheet({
      title,
      itemList: departments.value.map((item) => item.name),
      success: (result) => resolve(departments.value[result.tapIndex] ?? null),
      fail: () => resolve(null),
    });
  });
}

async function openPromoteDialog() {
  const target = await selectDepartment("选择医生所属部门");
  if (!target) return;
  selectedTargetDepartmentId.value = target.departmentId;
  profileDialogMode.value = "promote";
  profileDialogVisible.value = true;
}

function openEditDialog() {
  profileDialogMode.value = "edit";
  profileDialogVisible.value = true;
}

async function saveProfile(profile: DoctorProfileDraft) {
  const account = detail.value;
  if (!account || pending.value) return;
  pending.value = true;
  try {
    const beforeDepartmentId = account.departmentId;
    const updated = profileDialogMode.value === "promote"
      ? await identityAdminApi.promoteDoctor(
          account.accountId,
          selectedTargetDepartmentId.value,
          profile,
          account.managementVersion,
        )
      : await identityAdminApi.updateDoctor(account.accountId, profile, account.managementVersion);
    acceptMutation(updated, beforeDepartmentId);
    profileDialogVisible.value = false;
    uni.showToast({ title: profileDialogMode.value === "promote" ? "医生身份已开通" : "医生资料已更新" });
  } catch (caught) {
    uni.showToast({ title: messageOf(caught, "保存失败"), icon: "none" });
  } finally {
    pending.value = false;
  }
}

async function changeDepartment() {
  const account = detail.value;
  if (!account || pending.value) return;
  const target = await selectDepartment("选择调入部门");
  if (!target || target.departmentId === account.departmentId) return;
  confirmAction(
    "确认调岗",
    `将 ${displayName.value} 调至“${target.name}”？`,
    async () => identityAdminApi.changeDoctorDepartment(
      account.accountId,
      target.departmentId,
      account.managementVersion,
    ),
    "调岗完成",
  );
}

function revokeDoctor() {
  const account = detail.value;
  if (!account) return;
  confirmAction(
    "撤销医生身份",
    "撤销后该账号保留普通用户身份，历史业务数据不会删除。",
    () => identityAdminApi.revokeDoctor(account.accountId, account.managementVersion),
    "医生身份已撤销",
    true,
  );
}

function changeAccountEnabled(enabled: boolean) {
  const account = detail.value;
  if (!account) return;
  confirmAction(
    enabled ? "启用账号" : "禁用账号",
    enabled
      ? "启用后，该手机号可以重新登录并获取验证码。"
      : "禁用后，该手机号不能登录或获取验证码，历史数据仍会保留。",
    () => identityAdminApi.setAccountEnabled(account.accountId, enabled, account.managementVersion),
    enabled ? "账号已启用" : "账号已禁用",
    !enabled,
  );
}

function confirmAction(
  title: string,
  content: string,
  operation: () => Promise<AdminAccountDetail>,
  successText: string,
  dangerous = false,
) {
  if (pending.value) return;
  uni.showModal({
    title,
    content,
    confirmText: "确认",
    confirmColor: dangerous ? "#d9485f" : "#168bd8",
    success: async (result) => {
      if (!result.confirm || !detail.value) return;
      const beforeDepartmentId = detail.value.departmentId;
      pending.value = true;
      try {
        acceptMutation(await operation(), beforeDepartmentId);
        uni.showToast({ title: successText });
      } catch (caught) {
        uni.showToast({ title: messageOf(caught, "操作失败"), icon: "none" });
      } finally {
        pending.value = false;
      }
    },
  });
}

function acceptMutation(updated: AdminAccountDetail, beforeDepartmentId?: string) {
  detail.value = updated;
  invalidateDepartments();
  if (beforeDepartmentId) invalidateDoctors(beforeDepartmentId);
  if (updated.departmentId) invalidateDoctors(updated.departmentId);
}

function messageOf(value: unknown, fallback: string): string {
  return value instanceof Error && value.message ? value.message : fallback;
}
</script>

<template>
  <view class="detail-page">
    <view v-if="loading" class="page-state">加载中...</view>
    <view v-else-if="error" class="page-state page-state--error">
      <text>{{ error }}</text><button @tap="loadDetail">重新加载</button>
    </view>
    <template v-else-if="detail">
      <view class="profile-card">
        <view class="profile-avatar">{{ displayName.slice(0, 1) }}</view>
        <text class="profile-name">{{ displayName }}</text>
        <view class="profile-badges">
          <text class="identity-badge">{{ identityLabels[detail.identityType] }}</text>
          <text class="status-badge" :class="{ 'status-badge--disabled': detail.accountStatus === 'disabled' }">
            {{ detail.accountStatus === "active" ? "账号正常" : "账号已禁用" }}
          </text>
        </view>
        <text class="profile-phone">{{ detail.maskedPhone || "未绑定手机号" }}</text>
      </view>

      <view class="info-card">
        <view class="info-row"><text>昵称</text><text>{{ detail.nickname || "未设置" }}</text></view>
        <view class="info-row"><text>所属部门</text><text>{{ detail.departmentName || "无" }}</text></view>
        <view v-if="detail.identityType === 'doctor'" class="info-row"><text>工号</text><text>{{ detail.staffNo || "未设置" }}</text></view>
        <view v-if="detail.staffStatus === 'revoked'" class="info-row"><text>医生档案</text><text>已撤销，可重新开通</text></view>
        <view v-if="detail.identityType === 'doctor'" class="info-row info-row--multiline"><text>医生简介</text><text>{{ detail.description || "未设置" }}</text></view>
      </view>

      <view class="action-card">
        <text class="section-title">身份与账号操作</text>
        <text v-if="detail.availableActions.length === 0" class="no-actions">当前账号没有可执行操作</text>
        <button v-if="supports('promote_doctor')" class="action-button" @tap="openPromoteDialog">开通医生身份<text>›</text></button>
        <button v-if="supports('edit_doctor')" class="action-button" @tap="openEditDialog">编辑医生资料<text>›</text></button>
        <button v-if="supports('change_department')" class="action-button" @tap="changeDepartment">调岗<text>›</text></button>
        <button v-if="supports('revoke_doctor')" class="action-button action-button--danger" @tap="revokeDoctor">撤销医生身份<text>›</text></button>
        <button v-if="supports('disable_account')" class="action-button action-button--danger" @tap="changeAccountEnabled(false)">禁用账号<text>›</text></button>
        <button v-if="supports('enable_account')" class="action-button" @tap="changeAccountEnabled(true)">启用账号<text>›</text></button>
      </view>
    </template>

    <DoctorProfileDialog
      :visible="profileDialogVisible"
      :title="profileDialogTitle"
      :initial-value="initialProfile"
      :pending="pending"
      @close="profileDialogVisible = false"
      @submit="saveProfile"
    />
  </view>
</template>

<style scoped>
button::after { display:none; }
.detail-page { min-height:100vh; padding:24rpx; box-sizing:border-box; background:#f3f6fa; }
.profile-card { display:flex; flex-direction:column; align-items:center; padding:38rpx 30rpx 32rpx; background:linear-gradient(145deg,#168edc,#386fdf); border-radius:24rpx; box-shadow:0 14rpx 34rpx rgba(28,112,205,.2); }
.profile-avatar { display:flex; align-items:center; justify-content:center; width:104rpx; height:104rpx; color:#247ec4; font-size:40rpx; font-weight:700; background:#fff; border-radius:50%; }
.profile-name { margin-top:18rpx; color:#fff; font-size:34rpx; font-weight:700; }
.profile-badges { display:flex; gap:12rpx; margin-top:14rpx; }
.identity-badge,.status-badge { padding:5rpx 12rpx; color:#fff; font-size:20rpx; background:rgba(255,255,255,.2); border-radius:10rpx; }
.status-badge--disabled { color:#ffe3e7; background:rgba(130,19,43,.3); }
.profile-phone { margin-top:16rpx; color:rgba(255,255,255,.78); font-size:22rpx; }
.info-card,.action-card { margin-top:22rpx; overflow:hidden; background:#fff; border-radius:22rpx; box-shadow:0 8rpx 28rpx rgba(43,62,89,.05); }
.info-row { display:flex; justify-content:space-between; gap:30rpx; padding:25rpx 24rpx; color:#8b95a6; font-size:24rpx; border-bottom:1rpx solid #edf1f5; }
.info-row text:last-child { color:#303c51; text-align:right; }
.info-row--multiline text:last-child { max-width:430rpx; line-height:1.5; }
.section-title { display:block; padding:25rpx 24rpx 18rpx; color:#263248; font-size:28rpx; font-weight:700; }
.no-actions { display:block; padding:20rpx 24rpx 30rpx; color:#99a3b2; font-size:23rpx; }
.action-button { display:flex; align-items:center; justify-content:space-between; width:100%; margin:0; padding:0 24rpx; color:#2d394e; font-size:26rpx; line-height:86rpx; text-align:left; background:#fff; border-radius:0; border-top:1rpx solid #edf1f5; }
.action-button text { color:#bec6d1; font-size:38rpx; }
.action-button--danger { color:#d24d61; }
.page-state { display:flex; flex-direction:column; align-items:center; justify-content:center; gap:20rpx; min-height:520rpx; color:#96a0af; font-size:24rpx; text-align:center; }
.page-state--error { color:#c44f61; }
.page-state button { margin:0; color:#1683d0; font-size:23rpx; line-height:58rpx; background:#edf7ff; border-radius:29rpx; }
</style>
