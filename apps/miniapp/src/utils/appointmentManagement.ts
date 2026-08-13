import type { AppointmentSession } from "@/types/appointment";

interface AppointmentPrincipal {
  readonly roles: readonly string[];
  readonly permissions: readonly string[];
}

export const WEEKDAY_LABELS = ["周一", "周二", "周三", "周四", "周五", "周六", "周日"];
export const SESSION_LABELS: Record<AppointmentSession, string> = {
  morning: "上午",
  afternoon: "下午",
};

export function hasRole(principal: AppointmentPrincipal | null, role: string): boolean {
  return principal?.roles.includes(role) ?? false;
}

export function hasPermission(principal: AppointmentPrincipal | null, permission: string): boolean {
  return principal?.permissions.includes("*") || principal?.permissions.includes(permission) || false;
}

export function canReadAppointmentManagement(principal: AppointmentPrincipal | null): boolean {
  return (
    hasPermission(principal, "appointment.read") &&
    (hasRole(principal, "department_doctor") || hasRole(principal, "super_admin"))
  );
}

export function validClock(value: string): boolean {
  return /^(?:[01]\d|2[0-3]):[0-5]\d$/.test(value);
}

export function clockMinutes(value: string): number {
  if (!validClock(value)) return -1;
  const [hour, minute] = value.split(":").map(Number);
  return hour * 60 + minute;
}

export function itemWindowTimeValid(start: string, cutoff: string, end: string): boolean {
  const values = [clockMinutes(start), clockMinutes(cutoff), clockMinutes(end)];
  return values.every((value) => value >= 0) && values[0] <= values[1] && values[1] < values[2];
}

export function roomWindowTimeValid(start: string, end: string): boolean {
  const startMinutes = clockMinutes(start);
  const endMinutes = clockMinutes(end);
  return startMinutes >= 0 && endMinutes > startMinutes;
}

export function messageOf(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback;
}
