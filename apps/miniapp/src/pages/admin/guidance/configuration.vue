<script setup lang="ts">
import { onLoad } from "@dcloudio/uni-app";
import { computed, ref } from "vue";

import { guidanceApi } from "@/api/guidance";
import { operationId } from "@/api/management/httpShared";
import { loadExaminationItems } from "@/services/appointment";
import { loadDepartmentOptions } from "@/services/organization";
import { sessionState } from "@/stores/session";
import type { ExaminationItem } from "@/types/appointment";
import type { ConfiguredPrecedenceRule, PreparationRulePreview } from "@/types/guidance";
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

const isNew = computed(() => !itemVersion.value);
const canEdit = computed(() => hasPermission(sessionState.principal, "rule.edit"));
const durationIndex = computed(() => Math.max(0, durationOptions.indexOf(estimatedDurationMinutes.value)));
const durationLabels = durationOptions.map(formatEstimatedDuration);
const candidateLabels = computed(() => candidates.value.map((value) => `${value.item.name} · ${value.departmentLabel}`));
const stepTitle = computed(() => ["项目基本信息", "项目先后关系", "检查准备规则"][step.value - 1]);

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
      preview.value = { description: configuration.description, preparation_rules: configuration.preparation_rules, reminders: configuration.reminders, unresolved_fragments: [] };
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
        <text class="hint hint--top">这里只配置直接关系。其他科室项目可作为只读参照，不会修改对方项目。</text>
        <view v-for="(rule, index) in draftRules" :key="`${rule.predecessor_item_id}-${rule.successor_item_id}`" class="rule-card">
          <view><text class="rule-card__title">{{ rule.direction === 'before' ? `${rule.candidateName} 建议先于当前项目` : `当前项目建议先于 ${rule.candidateName}` }}</text><text class="rule-card__reason">{{ rule.staff_reason }}</text><text v-if="rule.patient_message" class="rule-card__patient">患者提示：{{ rule.patient_message }}</text></view>
          <button @tap="removeRule(index)">删除</button>
        </view>
        <view class="rule-editor">
          <text class="label">关联项目</text><picker :value="candidateIndex" :range="candidateLabels" @change="selectCandidate"><view class="picker">{{ candidateLabels[candidateIndex] || '暂无可选项目' }}<text>›</text></view></picker>
          <text class="label">直接顺序</text><picker :value="directionIndex" :range="['关联项目建议先做', '当前项目建议先做']" @change="selectDirection"><view class="picker">{{ directionIndex === 0 ? '关联项目建议先做' : '当前项目建议先做' }}<text>›</text></view></picker>
          <text class="label">工作人员设置理由</text><input v-model="staffReason" class="input" maxlength="512" placeholder="例如：抽血结果用于增强 CT 评估" />
          <text class="label">患者可见提示（可选）</text><input v-model="patientMessage" class="input" maxlength="512" placeholder="例如：建议先完成抽血" />
          <button class="secondary" :disabled="!candidateLabels.length" @tap="addRule">＋ 添加直接关系</button>
        </view>
        <view class="actions"><button @tap="step = 1">上一步</button><button class="primary" @tap="nextFromRules">下一步：解析准备规则</button></view>
      </view>

      <view v-else class="panel">
        <text class="label label--first">患者可见检查说明</text><textarea v-model="description" class="textarea" maxlength="4000" placeholder="填写空腹、禁水、憋尿、提前用药或妊娠提醒等说明" />
        <button class="secondary" :disabled="saving" @tap="parseDescription">{{ saving ? '解析中…' : '智能解析说明' }}</button>
        <view v-if="preview" class="preview">
          <text class="preview__title">结构化结果</text>
          <view v-for="rule in preview.preparation_rules" :key="rule.rule_type" class="preview__row"><text class="preview__tag">{{ ruleLabel(rule.rule_type) }}</text><text>{{ rule.start_mode === 'previous_day_time' ? `前一天 ${rule.previous_day_time} 起` : `提前 ${rule.min_advance_minutes / 60}–${rule.max_advance_minutes / 60} 小时` }}</text></view>
          <view v-for="reminder in preview.reminders" :key="reminder.text" class="preview__reminder">提醒：{{ reminder.text }}</view>
          <text v-if="!preview.preparation_rules.length && !preview.reminders.length" class="preview__empty">未识别到准备状态或提醒；确认后可以按无规则发布。</text>
        </view>
        <view class="actions"><button @tap="step = 2">上一步</button><button class="primary" :disabled="saving || !preview" @tap="submit">{{ saving ? '保存中…' : '一次性保存并发布' }}</button></view>
      </view>
    </template>
  </view>
</template>

<style scoped>
button::after{display:none}.page{min-height:100vh;padding:24rpx 24rpx calc(28rpx + env(safe-area-inset-bottom));box-sizing:border-box;background:#f3f6f9}.context{display:flex;justify-content:space-between;padding:20rpx 24rpx;color:#8b97a7;font-size:20rpx;background:#fff;border:1rpx solid #e5eaf0;border-radius:20rpx}.context text:last-child{color:#354257;font-weight:650}.steps{display:flex;align-items:center;margin:28rpx 22rpx 18rpx}.step{display:flex;flex:1;align-items:center;color:#9aa5b4}.step:last-child{flex:0}.step text{display:flex;width:42rpx;height:42rpx;align-items:center;justify-content:center;color:#8793a5;font-size:20rpx;background:#e5eaf0;border-radius:50%}.step view{width:100%;height:3rpx;background:#e1e6ec}.step--active text,.step--done text{color:#fff;background:#168bd7}.step--done view{background:#168bd7}.heading{margin:0 4rpx 18rpx}.heading__eyebrow,.heading__title{display:block}.heading__eyebrow{color:#168bd7;font-size:20rpx}.heading__title{margin-top:7rpx;color:#273449;font-size:32rpx;font-weight:750}.panel{padding:26rpx;background:#fff;border:1rpx solid #e4eaf0;border-radius:24rpx}.label{display:block;margin-top:22rpx;color:#647186;font-size:21rpx}.label--first{margin-top:0}.input,.picker,.textarea{width:100%;margin-top:9rpx;padding:0 20rpx;box-sizing:border-box;color:#2d3a4e;font-size:23rpx;background:#f5f8fa;border:1rpx solid #e3e9ef;border-radius:16rpx}.input,.picker{height:72rpx}.picker{display:flex;align-items:center;justify-content:space-between}.picker text{color:#8c98a8;font-size:30rpx}.textarea{height:220rpx;padding-top:18rpx}.hint{display:block;margin-top:18rpx;color:#8995a6;font-size:20rpx;line-height:1.6}.hint--top{margin-top:0}.primary,.secondary{width:100%;margin:24rpx 0 0;font-size:23rpx;line-height:70rpx;border-radius:35rpx}.primary{color:#fff;background:#168bd7}.secondary{color:#167fc2;background:#eaf5fc}.actions{display:grid;grid-template-columns:1fr 2fr;gap:14rpx;margin-top:24rpx}.actions button{margin:0;line-height:70rpx;border-radius:35rpx}.actions .primary{margin:0}.rule-card{display:flex;align-items:flex-start;justify-content:space-between;gap:16rpx;margin-top:14rpx;padding:20rpx;background:#f3f8fc;border-left:5rpx solid #168bd7;border-radius:16rpx}.rule-card__title,.rule-card__reason,.rule-card__patient{display:block}.rule-card__title{color:#304057;font-size:22rpx;font-weight:680}.rule-card__reason{margin-top:7rpx;color:#6f7d90;font-size:19rpx}.rule-card__patient{margin-top:5rpx;color:#2380b8;font-size:18rpx}.rule-card button{width:auto;margin:0;padding:0 12rpx;color:#c34f61;font-size:19rpx;line-height:46rpx;background:#fbecef;border-radius:23rpx}.rule-editor{margin-top:20rpx;padding:20rpx;background:#f8fafc;border-radius:18rpx}.preview{margin-top:18rpx;padding:20rpx;background:#f7fafc;border-radius:18rpx}.preview__title{display:block;color:#344157;font-size:23rpx;font-weight:700}.preview__row{display:flex;align-items:center;gap:12rpx;margin-top:13rpx;color:#59687d;font-size:20rpx}.preview__tag{padding:5rpx 11rpx;color:#147e62;background:#e2f7ef;border-radius:12rpx}.preview__reminder{margin-top:12rpx;color:#765f42;font-size:20rpx;line-height:1.5}.preview__empty{display:block;margin-top:14rpx;color:#8995a6;font-size:20rpx}.state{padding:70rpx 20rpx;color:#8491a4;font-size:22rpx;text-align:center;background:#fff;border-radius:22rpx}.state--error{color:#c44f61}.primary[disabled],.secondary[disabled]{opacity:.55}
</style>
