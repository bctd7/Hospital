import type { GuidancePatientReminder } from "@/types/guidance";

// patientReminderModalContent 只展示项目第三步配置的“患者提醒”。
export function patientReminderModalContent(reminders: GuidancePatientReminder[]) {
  const texts = reminders.map((value) => value.text.trim()).filter(Boolean);
  if (texts.length <= 1) return texts[0] ?? "";
  return texts.map((value, index) => `${index + 1}. ${value}`).join("\n");
}

export function confirmPatientReminders(reminders: GuidancePatientReminder[]) {
  const content = patientReminderModalContent(reminders);
  if (!content) return Promise.resolve(true);
  return new Promise<boolean>((resolve) => {
    uni.showModal({
      title: "检查前请确认",
      content,
      confirmText: "确认并报到",
      cancelText: "暂不报到",
      success: ({ confirm }) => resolve(confirm),
      fail: () => resolve(false),
    });
  });
}
