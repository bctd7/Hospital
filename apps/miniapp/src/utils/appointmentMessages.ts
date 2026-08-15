import type { AppointmentMessage, AppointmentMessageType } from "@/types/appointment";
import { formatEstimatedDuration } from "@/utils/appointmentManagement";

const chinaOffsetMilliseconds = 8 * 60 * 60 * 1000;

export function appointmentMessageTitle(
  type: AppointmentMessageType,
  staff: boolean,
): string {
  return ({
    booking_created: staff ? "收到新的检查预约" : "预约成功",
    arrival_60m: "距离最晚到院时间还有 1 小时",
    arrival_30m: "距离最晚到院时间还有 30 分钟",
    booking_canceled: "预约已取消",
    booking_no_show: staff ? "患者未到场" : "本次预约已记为未到场",
    booking_called: "已叫到您的候检号",
    booking_deferred: "本次叫号已顺延",
    report_due: "检查报告待完成",
    report_overdue: "检查报告仍未完成",
    report_published: "检查报告已发布",
    report_corrected: "检查报告已更正",
  } as Record<AppointmentMessageType, string>)[type];
}

export function appointmentMessageDetail(
  value: AppointmentMessage,
  staff: boolean,
): string {
  const booking = value.booking;
  if (staff) {
    const patient = booking.patientPhoneMasked
      ? `${booking.patientDisplayName || "患者"}（${booking.patientPhoneMasked}）`
      : booking.patientDisplayName || "患者";
    return `${patient} · ${booking.itemName}`;
  }
  return `${booking.itemName} · ${booking.departmentName || "检查科室"}`;
}

export function appointmentMessageSchedule(value: AppointmentMessage): string {
  const booking = value.booking;
  const location = [booking.campusName, booking.roomDisplayName]
    .filter((current) => current?.trim())
    .join(" · ");
  return `${booking.serviceDate} ${clock(booking.itemStartTime)}–${clock(booking.itemEndTime)} · ${formatEstimatedDuration(booking.estimatedDurationMinutes)}${location ? ` · ${location}` : ""}`;
}

export function appointmentMessageTime(value: string): string {
  const trimmed = value.trim();
  const timezoneAware = /(?:Z|[+-]\d{2}:?\d{2})$/i.test(trimmed);
  if (timezoneAware) {
    const instant = Date.parse(trimmed);
    if (!Number.isNaN(instant)) {
      const china = new Date(instant + chinaOffsetMilliseconds);
      return `${china.getUTCFullYear()}-${pad(china.getUTCMonth() + 1)}-${pad(china.getUTCDate())} ${pad(china.getUTCHours())}:${pad(china.getUTCMinutes())}`;
    }
  }
  const match = trimmed.match(/^(\d{4}-\d{2}-\d{2})[T\s](\d{2}:\d{2})/);
  return match ? `${match[1]} ${match[2]}` : trimmed;
}

export function appointmentMessageTarget(
  value: AppointmentMessage,
  staff: boolean,
): string {
  const bookingID = encodeURIComponent(value.booking.bookingId);
  if (staff) {
    return `/pages/admin/appointment/booking-detail?booking_id=${bookingID}`;
  }
  if (value.messageType === "report_published" || value.messageType === "report_corrected") {
    return `/pages/profile/reports/detail?booking_id=${bookingID}`;
  }
  const completed = ["completed", "no_show", "canceled"].includes(value.booking.status);
  return `/pages/profile/appointments/index?booking_id=${bookingID}${completed ? "&view=completed" : ""}`;
}

function clock(value: string): string {
  return value.slice(0, 5);
}

function pad(value: number): string {
  return String(value).padStart(2, "0");
}
