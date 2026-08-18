import { onLoad } from "@dcloudio/uni-app";
import { computed, ref, watch } from "vue";

import { guidanceApi } from "@/api/guidance";
import { operationId } from "@/api/management/httpShared";
import { normalizePreparationPreview } from "@/features/guidance/contracts";
import { createLatestRequestGuard } from "@/features/shared/latestRequest";
import { loadExaminationItems } from "@/services/appointment";
import { loadDepartmentOptions } from "@/services/organization";
import { sessionState } from "@/stores/session";
import type { ExaminationItem } from "@/types/appointment";
import type { ConfiguredPrecedenceRule, GuidancePreparationRule, PreparationRulePreview, PreparationRuleType } from "@/types/guidance";
import { formatEstimatedDuration, hasPermission, messageOf } from "@/utils/appointmentManagement";

interface Candidate { item: ExaminationItem; departmentLabel: string }
interface DraftRule extends ConfiguredPrecedenceRule { candidateName: string; direction: "before" | "after" }

export function useGuidanceConfiguration() {
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
  const parsing = ref(false);
  const saving = ref(false);
  const error = ref("");
  const operation = ref(operationId());
  const parsedDescription = ref("");
  const parseRequest = createLatestRequestGuard();
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

  watch(description, (value) => {
    if (!preview.value || value.trim() === parsedDescription.value) return;
    parseRequest.cancel();
    parsing.value = false;
    preview.value = undefined;
  });

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
        preview.value = normalizePreparationPreview({
          description: configuration.description, preparation_rules: configuration.preparation_rules,
          reminders: configuration.reminders, unresolved_fragments: [], parser_mode: "saved", warning: "",
        });
        parsedDescription.value = configuration.description.trim();
      }
    } catch (cause) {
      error.value = messageOf(cause, "导诊配置加载失败，请重试");
    } finally {
      loading.value = false;
    }
  }

  function selectDuration(event: { detail: { value: string | number } }) { estimatedDurationMinutes.value = durationOptions[Number(event.detail.value)] ?? 30; }
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
    if (draftRules.value.length) return void (step.value = 3);
    uni.showModal({ title: "确认无先后规则", content: "该项目不配置与其他项目的直接先后关系，是否继续？", confirmText: "继续", success: ({ confirm }) => { if (confirm) step.value = 3; } });
  }

  async function parseDescription() {
    if (parsing.value || saving.value) return;
    const source = description.value.trim();
    const token = parseRequest.begin();
    parsing.value = true;
    try {
      const result = await guidanceApi.previewPreparationRules(source);
      if (!parseRequest.isCurrent(token) || description.value.trim() !== source) return;
      preview.value = normalizePreparationPreview(result);
      parsedDescription.value = source;
      if (!preview.value.preparation_rules.length && !preview.value.reminders.length) {
        uni.showToast({ title: "未识别到结构化规则，可确认无规则后保存", icon: "none" });
      }
    } catch (cause) {
      if (parseRequest.isCurrent(token)) uni.showToast({ title: messageOf(cause, "准备规则解析失败"), icon: "none" });
    } finally {
      if (parseRequest.isCurrent(token)) parsing.value = false;
    }
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
  function setPreviousDayTime(rule: GuidancePreparationRule, event: { detail: { value: string | number } }) { rule.previous_day_time = String(event.detail.value); rule.source = "manual"; }
  function advanceIndex(value: number) { return Math.max(0, advanceMinuteOptions.indexOf(value)); }
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
    if (description.value.trim() !== parsedDescription.value) return void uni.showToast({ title: "检查说明已经变化，请重新解析后确认", icon: "none" });
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

  return {
    departmentLabel, name, step, stepTitle, error, loading, durationIndex, durationLabels,
    draftRules, candidateIndex, candidateLabels, directionIndex, directionLabels, staffReason, patientMessage,
    description, parsing, saving, preview, ruleTypes, ruleTypeLabels, startModeLabels,
    advanceLabels, reminderAdvanceOptions, reminderAdvanceLabels,
    selectDuration, nextFromBasics, removeRule, selectCandidate, selectDirection, addRule, nextFromRules,
    parseDescription, ruleLabel, sourceLabel, setRuleType, setStartMode, setAdvance, setPreviousDayTime,
    advanceIndex, setReminderAdvance, removePreparationRule, addPreparationRule, removeReminder, addReminder, submit,
  };
}

function ruleLabel(type: string) { return ({ fasting: "禁食", no_water: "禁水", drink_water: "提前饮水" } as Record<string, string>)[type] ?? type; }
function sourceLabel(source: string) { return ({ explicit: "根据原文时间", default: "医院默认补全", model: "模型解析", manual: "人工调整" } as Record<string, string>)[source] ?? "待确认"; }
function advanceLabel(value: number) {
  if (value < 60) return `${value}分钟`;
  const hours = Math.floor(value / 60);
  const minutes = value % 60;
  return minutes ? `${hours}小时${minutes}分钟` : `${hours}小时`;
}
