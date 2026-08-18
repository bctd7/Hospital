<script setup lang="ts">
import { onShow } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { loadExaminationItems } from "@/services/appointment";
import { loadDepartmentOptions, type DepartmentOption } from "@/services/organization";
import { sessionState } from "@/stores/session";
import type { ExaminationItem } from "@/types/appointment";
import { formatEstimatedDuration, hasPermission, hasRole, messageOf } from "@/utils/appointmentManagement";
import { currentStaffDepartmentId, rememberStaffDepartmentId } from "@/utils/staffDepartmentContext";

const departmentOptions = ref<DepartmentOption[]>([]);
const selectedDepartmentId = ref(currentStaffDepartmentId());
const items = ref<ExaminationItem[]>([]);
const search = ref("");
const loading = ref(false);
const error = ref("");

const isDoctor = computed(() => hasRole(sessionState.principal, "department_doctor"));
const canEdit = computed(() => hasPermission(sessionState.principal, "rule.edit"));
const selectedDepartment = computed(() => departmentOptions.value.find((value) => value.department.departmentId === selectedDepartmentId.value));
const filteredItems = computed(() => {
  const keyword = search.value.trim().toLowerCase();
  return keyword ? items.value.filter((item) => item.name.toLowerCase().includes(keyword)) : items.value;
});

onShow(() => void initialize());

async function initialize() {
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  try {
    departmentOptions.value = await loadDepartmentOptions(false);
    const ownDepartmentId = sessionState.principal?.department_id ?? "";
    const preferred = departmentOptions.value.find((value) => value.department.departmentId === (isDoctor.value ? ownDepartmentId : selectedDepartmentId.value))
      ?? departmentOptions.value[0];
    selectedDepartmentId.value = preferred?.department.departmentId ?? "";
    if (selectedDepartmentId.value) rememberStaffDepartmentId(selectedDepartmentId.value);
    await loadItems(true);
  } catch (cause) {
    error.value = messageOf(cause, "导诊项目加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

async function loadItems(force = false) {
  if (!selectedDepartmentId.value) {
    items.value = [];
    return;
  }
  const active = await loadExaminationItems(selectedDepartmentId.value, "active", 1, 100, force);
  items.value = active.items;
}

function switchDepartment() {
  if (isDoctor.value || departmentOptions.value.length <= 1) return;
  uni.showActionSheet({
    title: "选择科室",
    itemList: departmentOptions.value.map((value) => value.label),
    success: ({ tapIndex }) => {
      const option = departmentOptions.value[tapIndex];
      if (!option || option.department.departmentId === selectedDepartmentId.value) return;
      selectedDepartmentId.value = option.department.departmentId;
      rememberStaffDepartmentId(selectedDepartmentId.value);
      void loadItems(true);
    },
  });
}

function updateSearch(event: Event) {
  search.value = (event as unknown as { detail: { value?: string } }).detail.value ?? "";
}

function openConfiguration(item?: ExaminationItem) {
  if (!selectedDepartmentId.value || !canEdit.value) return;
  const label = selectedDepartment.value?.label ?? "所属科室";
  uni.navigateTo({
    url: `/pages/admin/guidance/configuration?department_id=${encodeURIComponent(selectedDepartmentId.value)}&department_label=${encodeURIComponent(label)}&item_id=${encodeURIComponent(item?.itemId ?? "")}`,
  });
}
</script>

<template>
  <view class="page">
    <view class="hero">
      <view>
        <text class="hero__eyebrow">智能导诊规则</text>
        <text class="hero__title">检查项目配置</text>
        <text class="hero__description">项目必须完成基本信息、先后关系和准备规则后才会发布。</text>
      </view>
      <button v-if="canEdit && selectedDepartmentId" class="hero__action" @tap="openConfiguration()">新建项目</button>
    </view>

    <button class="department" :disabled="isDoctor" @tap="switchDepartment">
      <view><text class="department__label">当前科室</text><text class="department__name">{{ selectedDepartment?.label || "暂无科室" }}</text></view>
      <text v-if="!isDoctor" class="department__switch">切换 ›</text>
    </button>

    <view class="search"><text class="search__icon">⌕</text><input :value="search" placeholder="搜索检查项目" @input="updateSearch" /></view>

    <view v-if="error" class="state state--error"><text>{{ error }}</text><button @tap="initialize">重新加载</button></view>
    <view v-else-if="loading" class="state">正在加载项目…</view>
    <view v-else class="list">
      <button v-for="item in filteredItems" :key="item.itemId" class="item" @tap="openConfiguration(item)">
        <view class="item__copy">
          <text class="item__name">{{ item.name }}</text>
          <text class="item__meta">预计 {{ formatEstimatedDuration(item.estimatedDurationMinutes) }}</text>
        </view>
        <view class="item__side"><text class="item__status">配置完成</text><text class="item__arrow">›</text></view>
      </button>
      <view v-if="!filteredItems.length" class="state"><text>{{ search ? "没有匹配项目" : "当前科室尚无检查项目" }}</text><button v-if="canEdit && !search" @tap="openConfiguration()">新建第一个项目</button></view>
    </view>
  </view>
</template>

<style scoped>
button::after{display:none}.page{min-height:100vh;padding:24rpx;box-sizing:border-box;background:#f3f6f9}.hero{display:flex;align-items:flex-end;justify-content:space-between;gap:24rpx;padding:32rpx;background:#fff;border:1rpx solid #e4eaf0;border-radius:26rpx}.hero__eyebrow,.hero__title,.hero__description{display:block}.hero__eyebrow{color:#1489cf;font-size:20rpx;font-weight:650}.hero__title{margin-top:8rpx;color:#253247;font-size:34rpx;font-weight:760}.hero__description{max-width:470rpx;margin-top:10rpx;color:#7f8ca0;font-size:21rpx;line-height:1.55}.hero__action{flex:0 0 auto;margin:0;padding:0 25rpx;color:#fff;font-size:22rpx;line-height:66rpx;background:#168bd7;border-radius:33rpx}.department{display:flex;width:100%;align-items:center;justify-content:space-between;margin:18rpx 0 0;padding:22rpx 26rpx;text-align:left;background:#fff;border:1rpx solid #e4eaf0;border-radius:22rpx}.department__label,.department__name{display:block}.department__label{color:#98a3b2;font-size:19rpx}.department__name{margin-top:6rpx;color:#344157;font-size:24rpx;font-weight:680}.department__switch{color:#168bd7;font-size:21rpx}.search{display:flex;align-items:center;gap:12rpx;height:72rpx;margin-top:18rpx;padding:0 22rpx;background:#fff;border:1rpx solid #e4eaf0;border-radius:20rpx}.search__icon{color:#8290a3;font-size:34rpx}.search input{flex:1;height:100%;color:#2c394d;font-size:23rpx}.list{margin-top:18rpx}.item{display:flex;width:100%;align-items:center;justify-content:space-between;margin:0 0 14rpx;padding:24rpx;text-align:left;background:#fff;border:1rpx solid #e4eaf0;border-radius:22rpx}.item__name,.item__meta{display:block}.item__name{color:#29364a;font-size:26rpx;font-weight:690}.item__meta{margin-top:8rpx;color:#8995a6;font-size:20rpx}.item__side{display:flex;align-items:center;gap:12rpx}.item__status{padding:6rpx 12rpx;color:#158464;font-size:18rpx;background:#e4f7ef;border-radius:14rpx}.item__arrow{color:#8b97a8;font-size:34rpx}.state{display:flex;flex-direction:column;align-items:center;gap:18rpx;margin-top:18rpx;padding:70rpx 24rpx;color:#8592a4;font-size:22rpx;text-align:center;background:#fff;border-radius:22rpx}.state--error{color:#c34e61}.state button{width:auto;margin:0;padding:0 24rpx;color:#168bd7;font-size:21rpx;line-height:56rpx;background:#eaf5fc;border-radius:28rpx}
</style>
