<script setup lang="ts">
import type { DepartmentSummary } from "@/types/staffManagement";

defineProps<{
  departments: DepartmentSummary[];
  selectedId: string;
  loading: boolean;
  showDisabled: boolean;
  canManage: boolean;
}>();

defineEmits<{
  (event: "select", department: DepartmentSummary): void;
  (event: "toggle-view"): void;
  (event: "create"): void;
}>();
</script>

<template>
  <view class="department-sidebar">
    <view class="department-sidebar__toolbar">
      <text>{{ showDisabled ? "已停用" : "部门" }}</text>
      <button v-if="canManage" class="toolbar-link" @tap="$emit('toggle-view')">
        {{ showDisabled ? "有效" : "停用" }}
      </button>
    </view>
    <button
      v-if="canManage && !showDisabled"
      class="add-department-button"
      @tap="$emit('create')"
    >
      ＋ 新增
    </button>
    <scroll-view class="department-list" scroll-y>
      <view v-if="loading" class="sidebar-state">加载中...</view>
      <view v-else-if="departments.length === 0" class="sidebar-state">
        {{ showDisabled ? "暂无停用部门" : "暂无部门" }}
      </view>
      <button
        v-for="department in departments"
        :key="department.departmentId"
        class="department-item"
        :class="{ 'department-item--active': department.departmentId === selectedId }"
        @tap="$emit('select', department)"
      >
        <text class="department-item__name">{{ department.name }}</text>
        <text class="department-item__count">
          {{ department.status === "active" ? `${department.doctorCount}人` : "已停用" }}
        </text>
      </button>
    </scroll-view>
  </view>
</template>

<style scoped>
.toolbar-link::after,
.add-department-button::after,
.department-item::after {
  display: none;
}

.department-sidebar {
  display: flex;
  flex: 0 0 34%;
  flex-direction: column;
  min-width: 0;
  background: #f7f9fc;
  border-right: 1rpx solid #e7ecf3;
}

.department-sidebar__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 88rpx;
  padding: 0 20rpx;
  box-sizing: border-box;
  color: #354055;
  font-size: 25rpx;
  font-weight: 650;
  border-bottom: 1rpx solid #eef1f6;
}

.toolbar-link {
  width: auto;
  margin: 0;
  padding: 0;
  color: #1d86d5;
  font-size: 22rpx;
  line-height: 1.4;
  background: transparent;
}

.add-department-button {
  margin: 16rpx;
  color: #167ac3;
  font-size: 23rpx;
  line-height: 58rpx;
  background: #eaf5ff;
  border: 1rpx dashed #87c4ee;
  border-radius: 14rpx;
}

.department-list {
  flex: 1;
  height: 0;
}

.department-item {
  width: 100%;
  margin: 0;
  padding: 24rpx 16rpx 22rpx 20rpx;
  color: #6f7a8c;
  text-align: left;
  background: transparent;
  border-radius: 0;
}

.department-item--active {
  position: relative;
  color: #147bc5;
  background: #ffffff;
}

.department-item--active::before {
  position: absolute;
  top: 20rpx;
  bottom: 20rpx;
  left: 0;
  width: 6rpx;
  content: "";
  background: #1497e3;
  border-radius: 0 6rpx 6rpx 0;
}

.department-item__name,
.department-item__count {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.department-item__name {
  font-size: 27rpx;
  font-weight: 600;
}

.department-item__count {
  margin-top: 8rpx;
  color: #a0a8b5;
  font-size: 20rpx;
}

.sidebar-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 240rpx;
  padding: 30rpx;
  color: #98a2b2;
  font-size: 23rpx;
  text-align: center;
}
</style>
