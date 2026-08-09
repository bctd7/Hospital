<script setup lang="ts">
import { computed } from "vue";

import type { DepartmentSummary, DoctorSummary } from "@/types/staffManagement";
import { createEmptyDepartment } from "@/utils/staffManagementView";

const emptyDepartment = createEmptyDepartment();

const props = withDefaults(defineProps<{
  department?: DepartmentSummary | null;
  doctors: DoctorSummary[];
  loading: boolean;
  error: string;
  canManage: boolean;
  canOpenDoctor: boolean;
}>(), {
  department: createEmptyDepartment,
});

defineEmits<{
  (event: "edit"): void;
  (event: "set-enabled", enabled: boolean): void;
  (event: "retry", departmentId: string): void;
  (event: "open-doctor", doctor: DoctorSummary): void;
}>();

const departmentView = computed(() => props.department ?? emptyDepartment);
const hasDepartment = computed(() => departmentView.value.departmentId.length > 0);
</script>

<template>
  <view class="doctor-panel">
    <view v-if="hasDepartment" class="doctor-panel__header">
      <view>
        <text class="doctor-panel__title">{{ departmentView.name }}</text>
        <text class="doctor-panel__count">
          {{ departmentView.status === "active" ? `${departmentView.doctorCount} 位医生` : "部门已停用" }}
        </text>
      </view>
      <view v-if="canManage" class="department-actions">
        <button v-if="departmentView.status === 'active'" class="action-link" @tap="$emit('edit')">
          编辑
        </button>
        <button
          class="action-link"
          :class="{ 'action-link--danger': departmentView.status === 'active' }"
          @tap="$emit('set-enabled', departmentView.status !== 'active')"
        >
          {{ departmentView.status === "active" ? "停用" : "恢复" }}
        </button>
      </view>
    </view>

    <scroll-view class="doctor-list" scroll-y>
      <view v-if="!hasDepartment" class="panel-state">请选择部门</view>
      <view v-else-if="departmentView.status === 'disabled'" class="panel-state">
        恢复部门后才能重新关联医生
      </view>
      <view v-else-if="loading" class="panel-state">医生加载中...</view>
      <view v-else-if="error" class="panel-state panel-state--error">
        <text>{{ error }}</text>
        <button class="panel-state__retry" @tap="$emit('retry', departmentView.departmentId)">重试</button>
      </view>
      <view v-else-if="doctors.length === 0" class="panel-state">
        <text class="panel-state__icon">医</text>
        <text>当前部门暂无医生</text>
      </view>
      <template v-else>
        <view
          v-for="doctor in doctors"
          :key="doctor.doctorId"
          class="doctor-item"
          :class="{ 'doctor-item--interactive': canOpenDoctor }"
          @tap="canOpenDoctor && $emit('open-doctor', doctor)"
        >
          <view class="doctor-avatar">{{ doctor.displayName.slice(0, 1) }}</view>
          <view class="doctor-item__body">
            <text class="doctor-item__name">{{ doctor.displayName }}</text>
            <text class="doctor-item__description">
              {{ doctor.description || "暂无擅长描述" }}
            </text>
          </view>
          <text v-if="canOpenDoctor" class="doctor-item__arrow">›</text>
        </view>
      </template>
    </scroll-view>
  </view>
</template>

<style scoped>
.action-link::after,
.panel-state__retry::after {
  display: none;
}

.doctor-panel {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.doctor-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 88rpx;
  padding: 0 24rpx;
  box-sizing: border-box;
  border-bottom: 1rpx solid #eef1f6;
}

.doctor-panel__title,
.doctor-panel__count {
  display: block;
}

.doctor-panel__title {
  max-width: 230rpx;
  overflow: hidden;
  color: #253148;
  font-size: 29rpx;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.doctor-panel__count {
  margin-top: 5rpx;
  color: #9aa4b4;
  font-size: 20rpx;
}

.department-actions {
  display: flex;
  gap: 18rpx;
}

.action-link {
  width: auto;
  margin: 0;
  padding: 0;
  color: #1d86d5;
  font-size: 22rpx;
  line-height: 1.4;
  background: transparent;
}

.action-link--danger {
  color: #d9485f;
}

.doctor-list {
  flex: 1;
  height: 0;
}

.doctor-item {
  display: flex;
  align-items: center;
  width: 100%;
  margin: 0;
  padding: 24rpx;
  text-align: left;
  background: #ffffff;
  border-radius: 0;
  border-bottom: 1rpx solid #f0f2f6;
}

.doctor-item--interactive:active {
  background: #f7fbff;
}

.doctor-avatar {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 70rpx;
  height: 70rpx;
  color: #ffffff;
  font-size: 26rpx;
  font-weight: 650;
  background: linear-gradient(145deg, #61b7ee, #5685ea);
  border-radius: 50%;
}

.doctor-item__body {
  flex: 1;
  min-width: 0;
  margin-left: 18rpx;
}

.doctor-item__name,
.doctor-item__description {
  display: block;
}

.doctor-item__name {
  color: #27334a;
  font-size: 28rpx;
  font-weight: 650;
}

.doctor-item__description {
  margin-top: 8rpx;
  overflow: hidden;
  color: #929bab;
  font-size: 21rpx;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.doctor-item__arrow {
  margin-left: 12rpx;
  color: #c2c9d4;
  font-size: 38rpx;
}

.panel-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 18rpx;
  min-height: 240rpx;
  padding: 30rpx;
  color: #98a2b2;
  font-size: 23rpx;
  text-align: center;
}

.panel-state__retry {
  margin: 0;
  color: #1683d0;
  font-size: 23rpx;
  line-height: 58rpx;
  background: #edf7ff;
  border-radius: 28rpx;
}

.panel-state__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 74rpx;
  height: 74rpx;
  color: #71a9d1;
  background: #edf7ff;
  border-radius: 50%;
}

.panel-state--error {
  color: #c44f61;
}
</style>
