<script setup lang="ts">
import type { HomeAction, HomeWorkbenchView } from "@/types/homeWorkbench";

defineProps<{ view: HomeWorkbenchView }>();
const emit = defineEmits<{ action: [action: HomeAction] }>();

function searchComingSoon() {
  uni.showToast({ title: "服务搜索功能正在建设中", icon: "none" });
}
</script>

<template>
  <view class="portal-home">
    <view class="portal-top">
      <button class="search-bar" @tap="searchComingSoon">
        <view class="search-bar__icon" />
        <text class="search-bar__placeholder">请输入关键词</text>
      </button>

      <view class="hero-card">
        <image
          class="hero-card__image"
          src="/static/home/hospital-campus-hero.jpg"
          mode="aspectFill"
        />
        <view class="hero-card__caption">
          <text class="hero-card__hospital">Hospital 医疗服务中心</text>
          <text class="hero-card__slogan">专业 · 便捷 · 安心</text>
        </view>
        <view class="hero-card__pager"><view class="hero-card__pager-dot" /></view>
      </view>

      <view class="featured-actions">
        <button
          v-if="view.primaryAction"
          class="featured-action"
          @tap="emit('action', view.primaryAction)"
        >
          <view class="featured-action__icon">{{ view.primaryAction.symbol }}</view>
          <view class="featured-action__copy">
            <text class="featured-action__title">{{ view.primaryAction.title }}</text>
            <text class="featured-action__description">{{ view.primaryAction.description }}</text>
          </view>
        </button>
        <button
          v-if="view.secondaryAction"
          class="featured-action featured-action--second"
          @tap="emit('action', view.secondaryAction)"
        >
          <view class="featured-action__icon">{{ view.secondaryAction.symbol }}</view>
          <view class="featured-action__copy">
            <text class="featured-action__title">{{ view.secondaryAction.title }}</text>
            <text class="featured-action__description">{{ view.secondaryAction.description }}</text>
          </view>
        </button>
      </view>
    </view>

    <view class="service-content">
      <view v-for="group in view.serviceGroups" :key="group.id" class="service-section">
        <text class="service-section__title">{{ group.title }}</text>
        <view class="service-grid" :class="{ 'service-grid--four': group.actions.length >= 4 }">
          <button
            v-for="action in group.actions"
            :key="action.id"
            class="service-item"
            @tap="emit('action', action)"
          >
            <view class="service-item__icon" :class="`service-item__icon--${action.tone}`">
              <text>{{ action.symbol }}</text>
              <view class="service-item__accent" />
            </view>
            <text class="service-item__title">{{ action.title }}</text>
          </button>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped>
button::after { display: none; }
.portal-home { min-height: 100vh; background: #f5f6f8; }
.portal-top { padding: 28rpx 24rpx 32rpx; background: linear-gradient(180deg, #62b8f8 0%, #68b5f2 100%); }
.search-bar { display: flex; align-items: center; width: 100%; height: 76rpx; margin: 0 0 24rpx; padding: 0 28rpx; color: inherit; text-align: left; background: rgba(255,255,255,.96); border-radius: 38rpx; }
.search-bar__icon { width: 29rpx; height: 29rpx; border: 3rpx solid #a6adb6; border-radius: 50%; position: relative; }
.search-bar__icon::after { content: ""; position: absolute; right: -10rpx; bottom: -6rpx; width: 14rpx; height: 3rpx; background: #a6adb6; transform: rotate(45deg); border-radius: 2rpx; }
.search-bar__placeholder { margin-left: 24rpx; color: #a0a5ad; font-size: 27rpx; }
.hero-card { position: relative; height: 300rpx; overflow: hidden; background: #d7e9f7; border-radius: 24rpx 24rpx 0 0; }
.hero-card__image { width: 100%; height: 100%; }
.hero-card__caption { position: absolute; left: 0; right: 0; bottom: 0; padding: 52rpx 26rpx 24rpx; color: #fff; background: linear-gradient(transparent, rgba(10,42,72,.62)); }
.hero-card__hospital,.hero-card__slogan { display: block; text-shadow: 0 2rpx 8rpx rgba(0,0,0,.25); }
.hero-card__hospital { font-size: 28rpx; font-weight: 700; }.hero-card__slogan { margin-top: 5rpx; font-size: 20rpx; opacity: .9; }
.hero-card__pager { position: absolute; left: 0; right: 0; bottom: 12rpx; display: flex; justify-content: center; }.hero-card__pager-dot { width: 34rpx; height: 7rpx; background: #fff; border-radius: 6rpx; }
.featured-actions { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); overflow: hidden; background: #fff; border-radius: 0 0 24rpx 24rpx; }
.featured-action { display: flex; align-items: center; min-height: 142rpx; margin: 0; padding: 24rpx 20rpx; color: inherit; text-align: left; background: #fff; border-radius: 0; }
.featured-action--second { border-left: 1rpx solid #eceef1; }
.featured-action__icon { display: flex; align-items: center; justify-content: center; width: 76rpx; height: 76rpx; flex: 0 0 auto; color: #fff; font-size: 30rpx; font-weight: 750; background: linear-gradient(135deg,#13d9c2,#05b7df); border-radius: 50%; box-shadow: 0 10rpx 20rpx rgba(10,194,203,.2); }
.featured-action__copy { min-width: 0; margin-left: 18rpx; }.featured-action__title,.featured-action__description { display: block; }.featured-action__title { color: #22262c; font-size: 31rpx; font-weight: 730; white-space: nowrap; }.featured-action__description { margin-top: 7rpx; overflow: hidden; color: #9a9fa7; font-size: 20rpx; text-overflow: ellipsis; white-space: nowrap; }
.service-content { background: #fff; }
.service-section { padding: 32rpx 0 20rpx; border-bottom: 1rpx solid #e8eaed; }
.service-section__title { display: block; padding: 0 38rpx 24rpx; color: #777d86; font-size: 29rpx; font-weight: 650; }
.service-grid { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); min-height: 190rpx; align-items: start; padding: 10rpx 18rpx 18rpx; }
.service-grid--four { grid-template-columns: repeat(4,minmax(0,1fr)); }
.service-item { display: flex; flex-direction: column; align-items: center; min-width: 0; margin: 0; padding: 14rpx 4rpx; color: inherit; background: transparent; }
.service-item__icon { position: relative; display: flex; align-items: center; justify-content: center; width: 70rpx; height: 70rpx; color: #34404d; font-size: 24rpx; font-weight: 750; border: 5rpx solid #34404d; border-radius: 19rpx; }
.service-item__icon--cyan,.service-item__icon--green { border-color: #34404d; }.service-item__icon--violet { border-radius: 50%; }.service-item__icon--amber { border-radius: 16rpx 16rpx 22rpx 22rpx; }
.service-item__accent { position: absolute; right: -8rpx; bottom: -7rpx; width: 24rpx; height: 24rpx; background: #168fe4; border: 5rpx solid #fff; border-radius: 50%; }
.service-item__title { width: 100%; margin-top: 22rpx; overflow: hidden; color: #444951; font-size: 23rpx; line-height: 1.35; text-align: center; text-overflow: ellipsis; white-space: nowrap; }
</style>
