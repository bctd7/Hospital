import { computed, onMounted, ref } from "vue";

import { patientAppointmentApi } from "@/api/appointment";
import { ApiError } from "@/api/client";
import { guidanceApi } from "@/api/guidance";
import { distinctSmartAppointmentPlans, visiblePlanKey } from "@/features/guidance/contracts";
import { createLatestRequestGuard } from "@/features/shared/latestRequest";
import { loadDepartmentOptions, type DepartmentOption } from "@/services/organization";
import type { ExaminationItem } from "@/types/appointment";
import type { SmartAppointmentPlan } from "@/types/guidance";

type SessionChoice = "all" | "morning" | "afternoon";

export function useSmartAppointmentPlanning() {
  const departments = ref<DepartmentOption[]>([]);
  const activeDepartmentId = ref("");
  const itemsByDepartment = ref<Record<string, ExaminationItem[]>>({});
  const selectedItemIds = ref<string[]>([]);
  const sessionChoices: ReadonlyArray<readonly [SessionChoice, string]> = [
    ["all", "全天"], ["morning", "上午"], ["afternoon", "下午"],
  ];
  const selectedAvailability = ref<Record<string, SessionChoice>>({});
  const plans = ref<SmartAppointmentPlan[]>([]);
  const loading = ref(false);
  const loadingItems = ref(false);
  const generating = ref(false);
  const confirmingPlanId = ref("");
  const errorMessage = ref("");
  const initialized = ref(false);
  const departmentRequest = createLatestRequestGuard();
  const planRequest = createLatestRequestGuard();

  const currentItems = computed(() => itemsByDepartment.value[activeDepartmentId.value] ?? []);
  const currentDepartment = computed(() => departments.value.find((value) => value.department.departmentId === activeDepartmentId.value));
  const selectedItems = computed(() => Object.values(itemsByDepartment.value).flat().filter((item) => selectedItemIds.value.includes(item.itemId)));
  const candidateDates = computed(() => remainingWeekDates());
  const selectedDates = computed(() => Object.keys(selectedAvailability.value).sort());
  const canGenerate = computed(() => initialized.value && !loadingItems.value && selectedItemIds.value.length > 0 && selectedDates.value.length > 0 && !generating.value);

  onMounted(async () => {
    loading.value = true;
    try {
      departments.value = await loadDepartmentOptions();
      const first = departments.value[0];
      if (first) await chooseDepartment(first.department.departmentId);
    } catch (error) {
      errorMessage.value = messageOf(error, "检查项目加载失败，请稍后重试");
    } finally {
      loading.value = false;
      initialized.value = true;
    }
  });

  async function chooseDepartment(departmentId: string) {
    activeDepartmentId.value = departmentId;
    if (itemsByDepartment.value[departmentId]) return;
    const token = departmentRequest.begin();
    loadingItems.value = true;
    try {
      const items = await patientAppointmentApi.listItems(departmentId);
      if (!departmentRequest.isCurrent(token)) return;
      itemsByDepartment.value = { ...itemsByDepartment.value, [departmentId]: items };
    } catch (error) {
      if (!departmentRequest.isCurrent(token)) return;
      errorMessage.value = messageOf(error, "当前科室检查项目加载失败");
    } finally {
      if (departmentRequest.isCurrent(token)) loadingItems.value = false;
    }
  }

  function resetGeneratedPlans() {
    planRequest.cancel();
    generating.value = false;
    plans.value = [];
    errorMessage.value = "";
  }

  function toggleItem(itemId: string) {
    selectedItemIds.value = selectedItemIds.value.includes(itemId)
      ? selectedItemIds.value.filter((value) => value !== itemId)
      : [...selectedItemIds.value, itemId];
    resetGeneratedPlans();
  }

  function toggleDate(value: string) {
    const next = { ...selectedAvailability.value };
    if (next[value]) delete next[value];
    else next[value] = "all";
    selectedAvailability.value = next;
    resetGeneratedPlans();
  }

  function chooseSession(date: string, choice: SessionChoice) {
    selectedAvailability.value = { ...selectedAvailability.value, [date]: choice };
    resetGeneratedPlans();
  }

  async function generatePlans() {
    if (!canGenerate.value) return;
    const token = planRequest.begin();
    const requestedItemIds = [...selectedItemIds.value];
    const requestedDates = [...selectedDates.value];
    generating.value = true;
    plans.value = [];
    errorMessage.value = "";
    try {
      const availability = requestedDates.map((serviceDate) => ({
        service_date: serviceDate,
        sessions: selectedAvailability.value[serviceDate] === "morning"
          ? ["morning" as const]
          : selectedAvailability.value[serviceDate] === "afternoon"
            ? ["afternoon" as const]
            : ["morning" as const, "afternoon" as const],
      }));
      const result = await guidanceApi.generateSmartAppointmentPlans(requestedItemIds, availability);
      if (!planRequest.isCurrent(token)) return;
      const nextPlans = distinctSmartAppointmentPlans(result.plans);
      plans.value = nextPlans;
      if (!nextPlans.length) errorMessage.value = "当前选择无法生成预约方案，请检查已有预约，或重新选择日期和时段";
    } catch (error) {
      if (!planRequest.isCurrent(token)) return;
      plans.value = [];
      errorMessage.value = error instanceof ApiError && error.code === "NO_SMART_APPOINTMENT_PLAN"
        ? "当前选择无法生成预约方案，请检查已有预约，或重新选择日期和时段"
        : messageOf(error, "方案生成失败，请稍后重试");
    } finally {
      if (planRequest.isCurrent(token)) generating.value = false;
    }
  }

  async function confirmPlan(plan: SmartAppointmentPlan) {
    if (confirmingPlanId.value) return;
    const accepted = await new Promise<boolean>((resolve) => {
      uni.showModal({
        title: "确认智能预约",
        content: `将一次性预约“${plan.title}”中的 ${plan.items.length} 个检查项目。`,
        success: (result) => resolve(result.confirm), fail: () => resolve(false),
      });
    });
    if (!accepted) return;
    confirmingPlanId.value = plan.plan_id;
    try {
      await guidanceApi.confirmSmartAppointmentPlan(plan.plan_id);
      uni.showToast({ title: "预约成功", icon: "success" });
      setTimeout(() => void uni.navigateTo({ url: "/pages/profile/appointments/index" }), 500);
    } catch (error) {
      uni.showModal({ title: "预约失败", content: messageOf(error, "方案可能已过期，请重新生成"), showCancel: false });
    } finally {
      confirmingPlanId.value = "";
    }
  }

  return {
    departments, activeDepartmentId, selectedItemIds, sessionChoices, selectedAvailability,
    plans, loading, loadingItems, generating, confirmingPlanId, errorMessage,
    currentItems, currentDepartment, selectedItems, candidateDates, selectedDates, canGenerate,
    chooseDepartment, toggleItem, toggleDate, chooseSession, generatePlans, confirmPlan,
    openTodayGuidance, sessionLabel, floorLabel, planKey: visiblePlanKey,
  };
}

function remainingWeekDates() {
  const result: Array<{ value: string; weekday: string; label: string }> = [];
  const now = new Date();
  const remaining = 7 - (now.getDay() || 7);
  for (let offset = 0; offset <= remaining; offset += 1) {
    const value = new Date(now.getFullYear(), now.getMonth(), now.getDate() + offset);
    result.push({
      value: localDate(value),
      weekday: ["周日", "周一", "周二", "周三", "周四", "周五", "周六"][value.getDay()]!,
      label: `${value.getMonth() + 1}/${value.getDate()}`,
    });
  }
  return result;
}

function localDate(value: Date) {
  return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, "0")}-${String(value.getDate()).padStart(2, "0")}`;
}
function openTodayGuidance() { void uni.redirectTo({ url: "/pages/guidance/today/index" }); }
function sessionLabel(value: string) { return value === "morning" ? "上午" : "下午"; }
function floorLabel(value: number) { return value < 0 ? `地下${Math.abs(value)}层` : `${value}层`; }
function messageOf(error: unknown, fallback: string) { return error instanceof Error && error.message.trim() ? error.message : fallback; }
