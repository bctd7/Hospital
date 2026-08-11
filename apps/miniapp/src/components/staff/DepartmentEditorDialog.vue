<script setup lang="ts">
const props = defineProps<{
  visible: boolean;
  mode: "create" | "edit";
  unitLabel?: string;
  name: string;
  pending: boolean;
}>();

const emit = defineEmits<{
  (event: "update:name", value: string): void;
  (event: "close"): void;
  (event: "save"): void;
}>();

function handleInput(event: Event) {
  const inputEvent = event as unknown as { detail: { value: string } };
  emit("update:name", inputEvent.detail.value);
}
</script>

<template>
  <view v-if="visible" class="dialog-mask" @tap="$emit('close')">
    <view class="editor-dialog" @tap.stop>
      <text class="editor-dialog__title">
        {{ mode === "create" ? `新增${unitLabel ?? "部门"}` : `编辑${unitLabel ?? "部门"}` }}
      </text>
      <input
        :value="props.name"
        class="editor-dialog__input"
        maxlength="32"
        :placeholder="`请输入${unitLabel ?? '部门'}名称`"
        focus
        @input="handleInput"
      />
      <view class="editor-dialog__actions">
        <button class="dialog-button dialog-button--secondary" @tap="$emit('close')">取消</button>
        <button
          class="dialog-button dialog-button--primary"
          :loading="pending"
          @tap="$emit('save')"
        >
          保存
        </button>
      </view>
    </view>
  </view>
</template>

<style scoped>
.dialog-button::after {
  display: none;
}

.dialog-mask {
  position: fixed;
  z-index: 20;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 42rpx;
  background: rgba(19, 29, 45, 0.45);
}

.editor-dialog {
  width: 100%;
  padding: 34rpx;
  background: #ffffff;
  border-radius: 26rpx;
  box-shadow: 0 28rpx 80rpx rgba(9, 24, 50, 0.25);
}

.editor-dialog__title {
  color: #202c41;
  font-size: 32rpx;
  font-weight: 700;
}

.editor-dialog__input {
  height: 82rpx;
  margin-top: 28rpx;
  padding: 0 24rpx;
  color: #29354a;
  font-size: 27rpx;
  background: #f5f7fa;
  border: 1rpx solid #e1e7ef;
  border-radius: 16rpx;
}

.editor-dialog__actions {
  display: flex;
  gap: 20rpx;
  margin-top: 30rpx;
}

.dialog-button {
  flex: 1;
  margin: 0;
  font-size: 26rpx;
  line-height: 76rpx;
  border-radius: 38rpx;
}

.dialog-button--secondary {
  color: #687488;
  background: #f0f3f7;
}

.dialog-button--primary {
  color: #ffffff;
  background: #168bd8;
}
</style>
