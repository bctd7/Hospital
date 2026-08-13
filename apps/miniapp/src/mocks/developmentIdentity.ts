import type { DevAuthRole } from "@/config/environment";
import type { CurrentIdentityResponse } from "@/types/auth";

const APPOINTMENT_PERMISSIONS = [
  "appointment.read",
  "appointment.create",
  "appointment.update",
  "appointment.cancel",
  "appointment.reschedule",
];

const CLINICAL_PERMISSIONS = [
  "planning.read",
  "planning.create",
  "planning.adjust",
  "report.read",
  "report.publish",
  "report.correct",
  "report.download",
  "navigation.read",
];

export function developmentPrincipal(role: DevAuthRole): CurrentIdentityResponse {
  if (role === "admin") {
    return {
      account_id: "00000000-0000-4000-8000-000000000001",
      account_type: "staff",
      status: "active",
      roles: ["super_admin"],
      permissions: [
        "identity.authorization.manage",
        "identity.account.manage",
        "identity.department.manage",
        ...APPOINTMENT_PERMISSIONS,
        ...CLINICAL_PERMISSIONS,
      ],
      authorization_version: 1,
    };
  }

  if (role === "doctor") {
    return {
      account_id: "00000000-0000-4000-8000-000000000002",
      account_type: "staff",
      status: "active",
      roles: ["department_doctor"],
      department_id: "00000000-0000-4000-8000-000000000101",
      permissions: [...APPOINTMENT_PERMISSIONS, ...CLINICAL_PERMISSIONS],
      authorization_version: 1,
    };
  }

  return {
    account_id: "00000000-0000-4000-8000-000000000003",
    account_type: "patient",
    status: "active",
    roles: [],
    permissions: [],
    authorization_version: 1,
  };
}
