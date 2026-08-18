<script setup lang="ts">
import { useGuidanceConfiguration } from "@/features/guidance/useGuidanceConfiguration";

const {
  departmentLabel, name, step, stepTitle, error, loading, durationIndex, durationLabels,
  draftRules, candidateIndex, candidateLabels, directionIndex, directionLabels, staffReason, patientMessage,
  description, parsing, saving, preview, ruleTypes, ruleTypeLabels, startModeLabels,
  advanceLabels, reminderAdvanceOptions, reminderAdvanceLabels,
  selectDuration, nextFromBasics, removeRule, selectCandidate, selectDirection, addRule, nextFromRules,
  parseDescription, ruleLabel, sourceLabel, setRuleType, setStartMode, setAdvance, setPreviousDayTime,
  advanceIndex, setReminderAdvance, removePreparationRule, addPreparationRule, removeReminder, addReminder, submit,
} = useGuidanceConfiguration();
</script>

<template>
  <view class="page">
    <view class="context"><text>所属科室</text><text>{{ departmentLabel }}</text></view>
    <view class="steps">
      <view v-for="index in 3" :key="index" class="step" :class="{ 'step--active': step === index, 'step--done': step > index }"><text>{{ index }}</text><view /></view>
    </view>
    <view class="heading"><text class="heading__eyebrow">第 {{ step }} 步，共 3 步</text><text class="heading__title">{{ stepTitle }}</text></view>

    <view v-if="error" class="state state--error"><text>{{ error }}</text></view>
    <view v-else-if="loading" class="state">正在加载配置…</view>
    <template v-else>
      <view v-if="step === 1" class="panel">
        <text class="label">项目名称</text><input v-model="name" class="input" maxlength="128" placeholder="例如：增强 CT" />
        <text class="label">预计检查时长</text><picker :value="durationIndex" :range="durationLabels" @change="selectDuration"><view class="picker">{{ durationLabels[durationIndex] }}<text>›</text></view></picker>
        <text class="hint">项目事实保存后用于预约容量和当日路线估算，时长按 5 分钟划分。</text>
        <button class="primary" @tap="nextFromBasics">下一步：配置先后关系</button>
      </view>

      <view v-else-if="step === 2" class="panel">
        <text class="hint hint--top">当前正在配置“{{ name }}”。请选择另一个项目，并直接说明患者建议先完成哪一项；这里只影响当前项目的导诊规则，不会修改对方项目。</text>
        <view v-for="(rule, index) in draftRules" :key="`${rule.predecessor_item_id}-${rule.successor_item_id}`" class="rule-card">
          <view><text class="rule-card__title">{{ rule.direction === 'before' ? `${rule.candidateName} 建议先于当前项目` : `当前项目建议先于 ${rule.candidateName}` }}</text><text class="rule-card__reason">{{ rule.staff_reason }}</text><text v-if="rule.patient_message" class="rule-card__patient">患者提示：{{ rule.patient_message }}</text></view>
          <button @tap="removeRule(index)">删除</button>
        </view>
        <view class="rule-editor">
          <text class="label">另一个检查项目</text><picker :value="candidateIndex" :range="candidateLabels" @change="selectCandidate"><view class="picker">{{ candidateLabels[candidateIndex] || '暂无可选项目' }}<text>›</text></view></picker>
          <text class="label">建议按照什么顺序完成？</text><picker :value="directionIndex" :range="directionLabels" @change="selectDirection"><view class="picker">{{ directionLabels[directionIndex] }}<text>›</text></view></picker>
          <text class="label">为什么建议这样安排？</text><input v-model="staffReason" class="input" maxlength="512" placeholder="例如：增强 CT 前需要先取得抽血结果" />
          <text class="label">患者可见提示（可选）</text><input v-model="patientMessage" class="input" maxlength="512" placeholder="例如：建议先完成抽血" />
          <button class="secondary" :disabled="!candidateLabels.length" @tap="addRule">＋ 添加这条先后建议</button>
        </view>
        <view class="actions"><button @tap="step = 1">上一步</button><button class="primary" @tap="nextFromRules">下一步：解析准备规则</button></view>
      </view>

      <view v-else class="panel">
        <text class="label label--first">患者可见检查说明</text><textarea v-model="description" class="textarea" maxlength="4000" placeholder="填写空腹、禁水、憋尿、提前用药或妊娠提醒等说明" />
        <button class="secondary" :disabled="parsing || saving" @tap="parseDescription">{{ parsing ? '解析中…' : '智能解析说明' }}</button>
        <view v-if="preview" class="preview">
          <text class="preview__title">解析结果（请确认或修改）</text>
          <text v-if="preview.warning" class="preview__warning">{{ preview.warning }}</text>
          <view v-for="(rule, ruleIndex) in preview.preparation_rules" :key="`${rule.rule_type}-${ruleIndex}`" class="rule-result">
            <view class="rule-result__heading"><picker :value="ruleTypes.indexOf(rule.rule_type)" :range="ruleTypeLabels" @change="setRuleType(rule, $event)"><view class="preview__tag">{{ ruleLabel(rule.rule_type) }} ▾</view></picker><text>{{ sourceLabel(rule.source) }}</text><button @tap="removePreparationRule(ruleIndex)">删除</button></view>
            <text class="label">时间计算方式</text><picker :value="rule.start_mode === 'advance_range' ? 0 : 1" :range="startModeLabels" @change="setStartMode(rule, $event)"><view class="picker picker--compact">{{ rule.start_mode === 'advance_range' ? '相对于预计检查时间提前' : '从前一天固定时间开始' }}<text>›</text></view></picker>
            <view v-if="rule.start_mode === 'advance_range'" class="range-grid">
              <view><text>最少提前</text><picker :value="advanceIndex(rule.min_advance_minutes)" :range="advanceLabels" @change="setAdvance(rule, 'min_advance_minutes', $event)"><view>{{ advanceLabels[advanceIndex(rule.min_advance_minutes)] }}</view></picker></view>
              <view><text>建议提前</text><picker :value="advanceIndex(rule.recommended_advance_minutes)" :range="advanceLabels" @change="setAdvance(rule, 'recommended_advance_minutes', $event)"><view>{{ advanceLabels[advanceIndex(rule.recommended_advance_minutes)] }}</view></picker></view>
              <view><text>最多提前</text><picker :value="advanceIndex(rule.max_advance_minutes)" :range="advanceLabels" @change="setAdvance(rule, 'max_advance_minutes', $event)"><view>{{ advanceLabels[advanceIndex(rule.max_advance_minutes)] }}</view></picker></view>
            </view>
            <picker v-else mode="time" :value="rule.previous_day_time || '20:00'" @change="setPreviousDayTime(rule, $event)"><view class="picker picker--compact">前一天 {{ rule.previous_day_time || '20:00' }} 起<text>›</text></view></picker>
            <text class="label">达到条件时的提示（可选）</text><input v-model="rule.readiness_hint" class="input" maxlength="256" placeholder="例如：出现明显尿意即可前往检查" @input="rule.source = 'manual'" />
          </view>
          <button class="secondary secondary--small" @tap="addPreparationRule">＋ 补充准备状态</button>
          <text v-if="preview.reminders.length" class="preview__subtitle">患者提醒</text>
          <view v-for="(reminder, reminderIndex) in preview.reminders" :key="reminderIndex" class="reminder-editor"><view><input v-model="reminder.text" class="input" maxlength="512" placeholder="患者可见提醒" /><picker :value="Math.max(0, reminderAdvanceOptions.indexOf(reminder.advance_minutes))" :range="reminderAdvanceLabels" @change="setReminderAdvance(reminderIndex, $event)"><view class="reminder-editor__time">{{ reminderAdvanceLabels[Math.max(0, reminderAdvanceOptions.indexOf(reminder.advance_minutes))] }}<text>›</text></view></picker></view><button @tap="removeReminder(reminderIndex)">删除</button></view>
          <button class="secondary secondary--small" @tap="addReminder">＋ 补充患者提醒</button>
          <view v-if="preview.unresolved_fragments.length" class="preview__warning">以下内容未能可靠归类，请人工补充：{{ preview.unresolved_fragments.join('；') }}</view>
          <text v-if="!preview.preparation_rules.length && !preview.reminders.length" class="preview__empty">未识别到准备状态或提醒；确认后可以按无规则发布。</text>
        </view>
        <view class="actions"><button @tap="step = 2">上一步</button><button class="primary" :disabled="parsing || saving || !preview" @tap="submit">{{ saving ? '保存中…' : '一次性保存并发布' }}</button></view>
      </view>
    </template>
  </view>
</template>

<style scoped>
button::after{display:none}.page{min-height:100vh;padding:24rpx 24rpx calc(28rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#f3f6f9}.context{display:flex;justify-content:space-between;padding:20rpx 24rpx;color:#8b97a7;font-size:20rpx;background:#fff;border:1rpx solid #e5eaf0;border-radius:20rpx}.context text:last-child{color:#354257;font-weight:650}.steps{display:flex;align-items:center;margin:28rpx 22rpx 18rpx}.step{display:flex;flex:1;align-items:center;color:#9aa5b4}.step:last-child{flex:0}.step text{display:flex;width:42rpx;height:42rpx;align-items:center;justify-content:center;color:#8793a5;font-size:20rpx;background:#e5eaf0;border-radius:50%}.step view{width:100%;height:3rpx;background:#e1e6ec}.step--active text,.step--done text{color:#fff;background:#168bd7}.step--done view{background:#168bd7}.heading{margin:0 4rpx 18rpx}.heading__eyebrow,.heading__title{display:block}.heading__eyebrow{color:#168bd7;font-size:20rpx}.heading__title{margin-top:7rpx;color:#273449;font-size:32rpx;font-weight:750}.panel{padding:26rpx;background:#fff;border:1rpx solid #e4eaf0;border-radius:24rpx}.label{display:block;margin-top:22rpx;color:#647186;font-size:21rpx}.label--first{margin-top:0}.input,.picker,.textarea{width:100%;margin-top:9rpx;padding:0 20rpx;box-sizing:border-box;color:#2d3a4e;font-size:23rpx;background:#f5f8fa;border:1rpx solid #e3e9ef;border-radius:16rpx}.input,.picker{height:72rpx}.picker{display:flex;align-items:center;justify-content:space-between}.picker text{color:#8c98a8;font-size:30rpx}.textarea{height:220rpx;padding-top:18rpx}.hint{display:block;margin-top:18rpx;color:#8995a6;font-size:20rpx;line-height:1.6}.hint--top{margin-top:0}.primary,.secondary{width:100%;margin:24rpx 0 0;font-size:23rpx;line-height:70rpx;border-radius:35rpx}.primary{color:#fff;background:#168bd7}.secondary{color:#167fc2;background:#eaf5fc}.actions{display:grid;grid-template-columns:1fr 2fr;gap:14rpx;margin-top:24rpx}.actions button{margin:0;line-height:70rpx;border-radius:35rpx}.actions .primary{margin:0}.rule-card{display:flex;align-items:flex-start;justify-content:space-between;gap:16rpx;margin-top:14rpx;padding:20rpx;background:#f3f8fc;border-left:5rpx solid #168bd7;border-radius:16rpx}.rule-card__title,.rule-card__reason,.rule-card__patient{display:block}.rule-card__title{color:#304057;font-size:22rpx;font-weight:680}.rule-card__reason{margin-top:7rpx;color:#6f7d90;font-size:19rpx}.rule-card__patient{margin-top:5rpx;color:#2380b8;font-size:18rpx}.rule-card button{width:auto;margin:0;padding:0 12rpx;color:#c34f61;font-size:19rpx;line-height:46rpx;background:#fbecef;border-radius:23rpx}.rule-editor{margin-top:20rpx;padding:20rpx;background:#f8fafc;border-radius:18rpx}.preview{margin-top:18rpx;padding:20rpx;background:#f7fafc;border-radius:18rpx}.preview__title{display:block;color:#344157;font-size:23rpx;font-weight:700}.preview__row{display:flex;align-items:center;gap:12rpx;margin-top:13rpx;color:#59687d;font-size:20rpx}.preview__tag{padding:5rpx 11rpx;color:#147e62;background:#e2f7ef;border-radius:12rpx}.preview__reminder{margin-top:12rpx;color:#765f42;font-size:20rpx;line-height:1.5}.preview__empty{display:block;margin-top:14rpx;color:#8995a6;font-size:20rpx}.state{padding:70rpx 20rpx;color:#8491a4;font-size:22rpx;text-align:center;background:#fff;border-radius:22rpx}.state--error{color:#c44f61}.primary[disabled],.secondary[disabled]{opacity:.55}
.preview__warning{display:block;margin-top:14rpx;padding:14rpx;color:#8a6530;font-size:19rpx;line-height:1.5;background:#fff6e5;border-radius:12rpx}.preview__subtitle{display:block;margin-top:24rpx;color:#344157;font-size:22rpx;font-weight:700}.rule-result{margin-top:16rpx;padding:18rpx;background:#fff;border:1rpx solid #e4eaf0;border-radius:16rpx}.rule-result__heading{display:flex;align-items:center;gap:12rpx}.rule-result__heading>text{flex:1;color:#8793a3;font-size:18rpx}.rule-result__heading button,.reminder-editor>button{width:auto;margin:0;padding:0 12rpx;color:#c34f61;font-size:18rpx;line-height:44rpx;background:#fbecef;border-radius:22rpx}.picker--compact{height:62rpx;font-size:20rpx}.range-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8rpx;margin-top:14rpx}.range-grid>view{padding:12rpx 8rpx;text-align:center;background:#f5f8fa;border-radius:12rpx}.range-grid text{display:block;color:#8a96a6;font-size:17rpx}.range-grid picker view{margin-top:6rpx;color:#354358;font-size:18rpx;font-weight:650}.secondary--small{margin-top:14rpx;font-size:20rpx;line-height:58rpx}.reminder-editor{display:flex;align-items:center;gap:10rpx;margin-top:10rpx}.reminder-editor>view{min-width:0;flex:1}.reminder-editor .input{margin-top:0}.reminder-editor__time{display:flex;align-items:center;justify-content:space-between;margin-top:8rpx;padding:0 16rpx;color:#69778b;font-size:18rpx;line-height:54rpx;background:#f5f8fa;border-radius:12rpx}.reminder-editor__time text{font-size:26rpx}
</style>
