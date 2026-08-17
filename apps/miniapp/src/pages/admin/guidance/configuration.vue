<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { guidanceApi } from "@/api/guidance";
import { operationId } from "@/api/management/httpShared";
import { loadExaminationItems } from "@/services/appointment";
import { loadDepartmentOptions } from "@/services/organization";
import { sessionState } from "@/stores/session";
import type { ExaminationItem } from "@/types/appointment";
import type { ConfiguredPrecedenceRule, GuidancePreparationRule, PreparationRulePreview, PreparationRuleType } from "@/types/guidance";
import { formatEstimatedDuration, hasPermission, messageOf } from "@/utils/appointmentManagement";

interface Candidate { item: ExaminationItem; departmentLabel: string }
interface DraftRule extends ConfiguredPrecedenceRule { candidateName: string; direction: "before" | "after" }

const departmentId = ref("");
const departmentLabel = ref("所属科室");
const itemId = ref("");
const itemVersion = ref(0);
const configurationVersion = ref(0);
const name = ref("");
const estimatedDurationMinutes = ref(30);
const description = ref("");
const step = ref(1);
const candidates = ref<Candidate[]>([]);
const draftRules = ref<DraftRule[]>([]);
const candidateIndex = ref(0);
const directionIndex = ref(0);
const staffReason = ref("");
const patientMessage = ref("");
const preview = ref<PreparationRulePreview>();
const loading = ref(false);
const saving = ref(false);
const error = ref("");
const operation = ref(operationId());
const durationOptions = Array.from({ length: 96 }, (_, index) => (index + 1) * 5);
const advanceMinuteOptions = Array.from({ length: 288 }, (_, index) => (index + 1) * 5);
const advanceLabels = advanceMinuteOptions.map(advanceLabel);
const reminderAdvanceOptions = [0, ...advanceMinuteOptions];
const reminderAdvanceLabels = reminderAdvanceOptions.map((value) => value ? `提前 ${advanceLabel(value)}` : "不设固定提前时间");
const ruleTypes: PreparationRuleType[] = ["fasting", "no_water", "drink_water"];
const ruleTypeLabels = ["禁食", "禁水", "提前饮水"];
const startModeLabels = ["提前一段时间", "前一天固定时间"];

const isNew = computed(() => !itemVersion.value);
const canEdit = computed(() => hasPermission(sessionState.principal, "rule.edit"));
const durationIndex = computed(() => Math.max(0, durationOptions.indexOf(estimatedDurationMinutes.value)));
const durationLabels = durationOptions.map(formatEstimatedDuration);
const candidateLabels = computed(() => candidates.value.map((value) => `${value.item.name} · ${value.departmentLabel}`));
const selectedCandidateName = computed(() => candidates.value[candidateIndex.value]?.item.name || "所选项目");
const directionLabels = computed(() => [
  `先做“${selectedCandidateName.value}”，再做“${name.value || "当前项目"}”`,
  `先做“${name.value || "当前项目"}”，再做“${selectedCandidateName.value}”`,
]);
const stepTitle = computed(() => ["项目基本信息", "检查项目先后建议", "检查准备规则"][step.value - 1]);

onLoad((options) => {
  departmentId.value = decodeURIComponent(String(options?.department_id ?? ""));
  departmentLabel.value = decodeURIComponent(String(options?.department_label ?? "所属科室"));
  itemId.value = decodeURIComponent(String(options?.item_id ?? "")) || operationId();
  uni.setNavigationBarTitle({ title: options?.item_id ? "修改导诊项目" : "新建导诊项目" });
  if (!canEdit.value) {
    error.value = "当前账号没有导诊规则编辑权限";
    return;
  }
  void initialize(Boolean(options?.item_id));
});

async function initialize(editing: boolean) {
  loading.value = true;
  error.value = "";
  try {
    const options = await loadDepartmentOptions(false);
    const pages = await Promise.all(options.map(async (value) => ({
      option: value,
      page: await loadExaminationItems(value.department.departmentId, "active", 1, 100, false),
    })));
    candidates.value = pages.flatMap(({ option, page }) => page.items
      .filter((item) => item.itemId !== itemId.value)
      .map((item) => ({ item, departmentLabel: option.label })));
    if (editing) {
      const configuration = await guidanceApi.getExaminationItemConfiguration(itemId.value);
      departmentId.value = configuration.owner_department_id;
      name.value = configuration.item_name;
      estimatedDurationMinutes.value = configuration.estimated_duration_minutes;
      description.value = configuration.description;
      itemVersion.value = configuration.item_version;
      configurationVersion.value = configuration.configuration_version;
      draftRules.value = configuration.precedence_rules.map((rule) => {
        const relatedId = rule.predecessor_item_id === itemId.value ? rule.successor_item_id : rule.predecessor_item_id;
        const candidate = candidates.value.find((value) => value.item.itemId === relatedId);
        return { ...rule, candidateName: candidate?.item.name ?? "关联项目", direction: rule.successor_item_id === itemId.value ? "before" : "after" };
      });
      preview.value = { description: configuration.description, preparation_rules: configuration.preparation_rules, reminders: configuration.reminders, unresolved_fragments: [], parser_mode: "saved", warning: "" };
    }
  } catch (cause) {
    error.value = messageOf(cause, "导诊配置加载失败，请重试");
  } finally {
    loading.value = false;
  }
}

function selectDuration(event: { detail: { value: string | number } }) {
  estimatedDurationMinutes.value = durationOptions[Number(event.detail.value)] ?? 30;
}
function selectCandidate(event: { detail: { value: string | number } }) { candidateIndex.value = Number(event.detail.value); }
function selectDirection(event: { detail: { value: string | number } }) { directionIndex.value = Number(event.detail.value); }

function nextFromBasics() {
  if (!name.value.trim()) return void uni.showToast({ title: "请输入项目名称", icon: "none" });
  step.value = 2;
}

function addRule() {
  const candidate = candidates.value[candidateIndex.value];
  if (!candidate) return void uni.showToast({ title: "请选择关联项目", icon: "none" });
  if (!staffReason.value.trim()) return void uni.showToast({ title: "请填写设置理由", icon: "none" });
  if (draftRules.value.some((value) => value.predecessor_item_id === candidate.item.itemId || value.successor_item_id === candidate.item.itemId)) {
    return void uni.showToast({ title: "该关联项目已经配置", icon: "none" });
  }
  const candidateFirst = directionIndex.value === 0;
  draftRules.value.push({
    predecessor_item_id: candidateFirst ? candidate.item.itemId : itemId.value,
    successor_item_id: candidateFirst ? itemId.value : candidate.item.itemId,
    staff_reason: staffReason.value.trim(), patient_message: patientMessage.value.trim(),
    candidateName: candidate.item.name, direction: candidateFirst ? "before" : "after",
  });
  staffReason.value = "";
  patientMessage.value = "";
}

function removeRule(index: number) { draftRules.value.splice(index, 1); }

function nextFromRules() {
  if (draftRules.value.length) {
    step.value = 3;
    return;
  }
  uni.showModal({ title: "确认无先后规则", content: "该项目不配置与其他项目的直接先后关系，是否继续？", confirmText: "继续", success: ({ confirm }) => { if (confirm) step.value = 3; } });
}

async function parseDescription() {
  if (saving.value) return;
  saving.value = true;
  try {
    preview.value = await guidanceApi.previewPreparationRules(description.value.trim());
    if (!preview.value.preparation_rules.length && !preview.value.reminders.length) {
      uni.showToast({ title: "未识别到结构化规则，可确认无规则后保存", icon: "none" });
    }
  } catch (cause) {
    uni.showToast({ title: messageOf(cause, "准备规则解析失败"), icon: "none" });
  } finally { saving.value = false; }
}

function ruleLabel(type: string) {
  return ({ fasting: "禁食", no_water: "禁水", drink_water: "提前饮水" } as Record<string, string>)[type] ?? type;
}

function sourceLabel(source: string) {
  return ({ explicit: "根据原文时间", default: "医院默认补全", model: "模型解析", manual: "人工调整" } as Record<string, string>)[source] ?? "待确认";
}

function setRuleType(rule: GuidancePreparationRule, event: { detail: { value: string | number } }) {
  rule.rule_type = ruleTypes[Number(event.detail.value)] ?? "fasting";
  rule.source = "manual";
}

function setStartMode(rule: GuidancePreparationRule, event: { detail: { value: string | number } }) {
  rule.start_mode = Number(event.detail.value) === 0 ? "advance_range" : "previous_day_time";
  if (rule.start_mode === "advance_range" && !rule.min_advance_minutes) {
    rule.min_advance_minutes = 480; rule.recommended_advance_minutes = 480; rule.max_advance_minutes = 720;
  }
  if (rule.start_mode === "previous_day_time" && !rule.previous_day_time) rule.previous_day_time = "20:00";
  rule.source = "manual";
}

function setAdvance(rule: GuidancePreparationRule, field: "min_advance_minutes" | "recommended_advance_minutes" | "max_advance_minutes", event: { detail: { value: string | number } }) {
  rule[field] = advanceMinuteOptions[Number(event.detail.value)] ?? 60;
  rule.source = "manual";
}

function setPreviousDayTime(rule: GuidancePreparationRule, event: { detail: { value: string | number } }) {
  rule.previous_day_time = String(event.detail.value);
  rule.source = "manual";
}

function advanceIndex(value: number) { return Math.max(0, advanceMinuteOptions.indexOf(value)); }
function advanceLabel(value: number) {
  if (value < 60) return `${value}分钟`;
  const hours = Math.floor(value / 60);
  const minutes = value % 60;
  return minutes ? `${hours}小时${minutes}分钟` : `${hours}小时`;
}
function setReminderAdvance(index: number, event: { detail: { value: string | number } }) {
  const reminder = preview.value?.reminders[index];
  if (reminder) reminder.advance_minutes = reminderAdvanceOptions[Number(event.detail.value)] ?? 0;
}
function removePreparationRule(index: number) { preview.value?.preparation_rules.splice(index, 1); }
function addPreparationRule() {
  if (!preview.value) return;
  const type = ruleTypes.find((candidate) => !preview.value?.preparation_rules.some((rule) => rule.rule_type === candidate));
  if (!type) return void uni.showToast({ title: "三种准备状态均已存在", icon: "none" });
  preview.value.preparation_rules.push({ rule_type: type, start_mode: "advance_range", min_advance_minutes: 60, recommended_advance_minutes: 60, max_advance_minutes: 60, previous_day_time: "", readiness_hint: "", source: "manual" });
}
function removeReminder(index: number) { preview.value?.reminders.splice(index, 1); }
function addReminder() { preview.value?.reminders.push({ text: "", advance_minutes: 0 }); }

async function submit() {
  if (!preview.value) return void uni.showToast({ title: "请先解析检查说明", icon: "none" });
  const perform = async () => {
    saving.value = true;
    try {
      const result = await guidanceApi.configureExaminationItem({
        action: isNew.value ? "create" : "update", item_id: itemId.value,
        owner_department_id: departmentId.value, item_name: name.value.trim(),
        estimated_duration_minutes: estimatedDurationMinutes.value, expected_item_version: itemVersion.value,
        description: description.value.trim(),
        precedence_rules: draftRules.value.map(({ candidateName: _candidateName, direction: _direction, ...value }) => value),
        preparation_rules: preview.value?.preparation_rules ?? [], reminders: preview.value?.reminders ?? [],
        expected_configuration_version: configurationVersion.value, operation_id: operation.value,
      });
      itemVersion.value = result.item_version;
      configurationVersion.value = result.configuration_version;
      uni.showToast({ title: "项目与规则已保存" });
      setTimeout(() => uni.redirectTo({ url: `/pages/admin/appointment/item-detail?department_id=${encodeURIComponent(departmentId.value)}&item_id=${encodeURIComponent(result.item_id)}&department_label=${encodeURIComponent(departmentLabel.value)}` }), 700);
    } catch (cause) {
      uni.showToast({ title: messageOf(cause, "完整配置保存失败"), icon: "none" });
    } finally { saving.value = false; }
  };
  if (!preview.value.preparation_rules.length && !preview.value.reminders.length) {
    uni.showModal({ title: "确认无准备规则", content: "该项目没有结构化准备状态和患者提醒，确认后仍可发布。", confirmText: "确认发布", success: ({ confirm }) => { if (confirm) void perform(); } });
  } else void perform();
}
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
        <button class="secondary" :disabled="saving" @tap="parseDescription">{{ saving ? '解析中…' : '智能解析说明' }}</button>
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
        <view class="actions"><button @tap="step = 2">上一步</button><button class="primary" :disabled="saving || !preview" @tap="submit">{{ saving ? '保存中…' : '一次性保存并发布' }}</button></view>
      </view>
    </template>
  </view>
</template>

<style scoped>
button::after{display:none}.page{min-height:100vh;padding:24rpx 24rpx calc(28rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#f3f6f9}.context{display:flex;justify-content:space-between;padding:20rpx 24rpx;color:#8b97a7;font-size:20rpx;background:#fff;border:1rpx solid #e5eaf0;border-radius:20rpx}.context text:last-child{color:#354257;font-weight:650}.steps{display:flex;align-items:center;margin:28rpx 22rpx 18rpx}.step{display:flex;flex:1;align-items:center;color:#9aa5b4}.step:last-child{flex:0}.step text{display:flex;width:42rpx;height:42rpx;align-items:center;justify-content:center;color:#8793a5;font-size:20rpx;background:#e5eaf0;border-radius:50%}.step view{width:100%;height:3rpx;background:#e1e6ec}.step--active text,.step--done text{color:#fff;background:#168bd7}.step--done view{background:#168bd7}.heading{margin:0 4rpx 18rpx}.heading__eyebrow,.heading__title{display:block}.heading__eyebrow{color:#168bd7;font-size:20rpx}.heading__title{margin-top:7rpx;color:#273449;font-size:32rpx;font-weight:750}.panel{padding:26rpx;background:#fff;border:1rpx solid #e4eaf0;border-radius:24rpx}.label{display:block;margin-top:22rpx;color:#647186;font-size:21rpx}.label--first{margin-top:0}.input,.picker,.textarea{width:100%;margin-top:9rpx;padding:0 20rpx;box-sizing:border-box;color:#2d3a4e;font-size:23rpx;background:#f5f8fa;border:1rpx solid #e3e9ef;border-radius:16rpx}.input,.picker{height:72rpx}.picker{display:flex;align-items:center;justify-content:space-between}.picker text{color:#8c98a8;font-size:30rpx}.textarea{height:220rpx;padding-top:18rpx}.hint{display:block;margin-top:18rpx;color:#8995a6;font-size:20rpx;line-height:1.6}.hint--top{margin-top:0}.primary,.secondary{width:100%;margin:24rpx 0 0;font-size:23rpx;line-height:70rpx;border-radius:35rpx}.primary{color:#fff;background:#168bd7}.secondary{color:#167fc2;background:#eaf5fc}.actions{display:grid;grid-template-columns:1fr 2fr;gap:14rpx;margin-top:24rpx}.actions button{margin:0;line-height:70rpx;border-radius:35rpx}.actions .primary{margin:0}.rule-card{display:flex;align-items:flex-start;justify-content:space-between;gap:16rpx;margin-top:14rpx;padding:20rpx;background:#f3f8fc;border-left:5rpx solid #168bd7;border-radius:16rpx}.rule-card__title,.rule-card__reason,.rule-card__patient{display:block}.rule-card__title{color:#304057;font-size:22rpx;font-weight:680}.rule-card__reason{margin-top:7rpx;color:#6f7d90;font-size:19rpx}.rule-card__patient{margin-top:5rpx;color:#2380b8;font-size:18rpx}.rule-card button{width:auto;margin:0;padding:0 12rpx;color:#c34f61;font-size:19rpx;line-height:46rpx;background:#fbecef;border-radius:23rpx}.rule-editor{margin-top:20rpx;padding:20rpx;background:#f8fafc;border-radius:18rpx}.preview{margin-top:18rpx;padding:20rpx;background:#f7fafc;border-radius:18rpx}.preview__title{display:block;color:#344157;font-size:23rpx;font-weight:700}.preview__row{display:flex;align-items:center;gap:12rpx;margin-top:13rpx;color:#59687d;font-size:20rpx}.preview__tag{padding:5rpx 11rpx;color:#147e62;background:#e2f7ef;border-radius:12rpx}.preview__reminder{margin-top:12rpx;color:#765f42;font-size:20rpx;line-height:1.5}.preview__empty{display:block;margin-top:14rpx;color:#8995a6;font-size:20rpx}.state{padding:70rpx 20rpx;color:#8491a4;font-size:22rpx;text-align:center;background:#fff;border-radius:22rpx}.state--error{color:#c44f61}.primary[disabled],.secondary[disabled]{opacity:.55}
.preview__warning{display:block;margin-top:14rpx;padding:14rpx;color:#8a6530;font-size:19rpx;line-height:1.5;background:#fff6e5;border-radius:12rpx}.preview__subtitle{display:block;margin-top:24rpx;color:#344157;font-size:22rpx;font-weight:700}.rule-result{margin-top:16rpx;padding:18rpx;background:#fff;border:1rpx solid #e4eaf0;border-radius:16rpx}.rule-result__heading{display:flex;align-items:center;gap:12rpx}.rule-result__heading>text{flex:1;color:#8793a3;font-size:18rpx}.rule-result__heading button,.reminder-editor>button{width:auto;margin:0;padding:0 12rpx;color:#c34f61;font-size:18rpx;line-height:44rpx;background:#fbecef;border-radius:22rpx}.picker--compact{height:62rpx;font-size:20rpx}.range-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8rpx;margin-top:14rpx}.range-grid>view{padding:12rpx 8rpx;text-align:center;background:#f5f8fa;border-radius:12rpx}.range-grid text{display:block;color:#8a96a6;font-size:17rpx}.range-grid picker view{margin-top:6rpx;color:#354358;font-size:18rpx;font-weight:650}.secondary--small{margin-top:14rpx;font-size:20rpx;line-height:58rpx}.reminder-editor{display:flex;align-items:center;gap:10rpx;margin-top:10rpx}.reminder-editor>view{min-width:0;flex:1}.reminder-editor .input{margin-top:0}.reminder-editor__time{display:flex;align-items:center;justify-content:space-between;margin-top:8rpx;padding:0 16rpx;color:#69778b;font-size:18rpx;line-height:54rpx;background:#f5f8fa;border-radius:12rpx}.reminder-editor__time text{font-size:26rpx}
</style>
