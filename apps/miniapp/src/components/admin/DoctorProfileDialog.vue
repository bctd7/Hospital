<script setup lang="ts">
import { reactive, watch } from "vue";

import type { DoctorProfileDraft } from "@/types/staffManagement";

const props = defineProps<{
  visible: boolean;
  title: string;
  initialValue: DoctorProfileDraft;
  pending: boolean;
}>();
const emit = defineEmits<{
  (event: "close"): void;
  (event: "submit", value: DoctorProfileDraft): void;
}>();

const form = reactive<DoctorProfileDraft>({ displayName: "", staffNo: "", description: "" });

watch(
  () => [props.visible, props.initialValue] as const,
  () => {
    if (!props.visible) return;
    form.displayName = props.initialValue.displayName;
    form.staffNo = props.initialValue.staffNo ?? "";
    form.description = props.initialValue.description ?? "";
  },
  { immediate: true },
);

function valueOf(event: Event): string {
  return (event as unknown as { detail: { value: string } }).detail.value;
}

function submit() {
  if (!form.displayName.trim()) {
    uni.showToast({ title: "请输入医生显示名称", icon: "none" });
    return;
  }
  emit("submit", {
    displayName: form.displayName.trim(),
    staffNo: form.staffNo?.trim(),
    description: form.description?.trim(),
  });
}
</script>

<template>
  <view v-if="visible" class="dialog-mask" @tap="$emit('close')">
    <view class="profile-dialog" @tap.stop>
      <text class="profile-dialog__title">{{ title }}</text>
      <text class="field-label">显示名称</text>
      <input :value="form.displayName" class="field-input" maxlength="32" placeholder="例如：林医生" @input="form.displayName = valueOf($event)" />
      <text class="field-label">工号（选填）</text>
      <input :value="form.staffNo" class="field-input" maxlength="32" placeholder="院内工号" @input="form.staffNo = valueOf($event)" />
      <text class="field-label">医生简介（选填）</text>
      <textarea :value="form.description" class="field-textarea" maxlength="160" placeholder="擅长方向或公开介绍" @input="form.description = valueOf($event)" />
      <view class="dialog-actions">
        <button class="dialog-button dialog-button--secondary" @tap="$emit('close')">取消</button>
        <button class="dialog-button dialog-button--primary" :loading="pending" @tap="submit">保存</button>
      </view>
    </view>
  </view>
</template>

<style scoped>
.dialog-button::after { display:none; }
.dialog-mask { position:fixed; z-index:30; inset:0; display:flex; align-items:center; justify-content:center; padding:42rpx; background:rgba(18,28,45,.48); }
.profile-dialog { width:100%; padding:34rpx; box-sizing:border-box; background:#fff; border-radius:26rpx; }
.profile-dialog__title { display:block; margin-bottom:28rpx; color:#202c41; font-size:32rpx; font-weight:700; }
.field-label { display:block; margin:20rpx 0 10rpx; color:#606c80; font-size:23rpx; }
.field-input,.field-textarea { width:100%; box-sizing:border-box; color:#29354a; font-size:26rpx; background:#f5f7fa; border:1rpx solid #e1e7ef; border-radius:14rpx; }
.field-input { height:76rpx; padding:0 20rpx; }
.field-textarea { height:150rpx; padding:18rpx 20rpx; }
.dialog-actions { display:flex; gap:20rpx; margin-top:30rpx; }
.dialog-button { flex:1; margin:0; font-size:26rpx; line-height:74rpx; border-radius:37rpx; }
.dialog-button--secondary { color:#687488; background:#f0f3f7; }
.dialog-button--primary { color:#fff; background:#168bd8; }
</style>
