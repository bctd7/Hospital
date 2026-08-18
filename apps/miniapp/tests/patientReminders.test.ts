import { describe, expect, it } from "vitest";

import { patientReminderModalContent } from "@/features/guidance/patientReminders";

describe("patientReminderModalContent", () => {
  it("只输出患者提醒文本", () => {
    expect(patientReminderModalContent([
      { text: "请确认您今天不在月经期。", advance_minutes: 0 },
      { text: "如有怀孕可能，请先告知工作人员。", advance_minutes: 0 },
    ])).toBe("1. 请确认您今天不在月经期。\n2. 如有怀孕可能，请先告知工作人员。");
  });

  it("忽略空白提醒", () => {
    expect(patientReminderModalContent([{ text: "  ", advance_minutes: 0 }])).toBe("");
  });
});
