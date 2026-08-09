import { beforeEach, describe, expect, it } from "vitest";

import {
  mockStaffManagementApi,
  resetMockStaffManagement,
} from "@/api/staffManagement.mock";

describe("staff management mock contract", () => {
  beforeEach(() => resetMockStaffManagement());

  it("keeps disabled departments out of the public directory", async () => {
    const publicDepartments = await mockStaffManagementApi.listDepartments(false);
    const adminDepartments = await mockStaffManagementApi.listDepartments(true);

    expect(publicDepartments.every((item) => item.status === "active")).toBe(true);
    expect(adminDepartments.some((item) => item.status === "disabled")).toBe(true);
  });

  it("uses full phone for exact lookup and names for alternative search", async () => {
    const byPhone = await mockStaffManagementApi.searchAccountByPhone("138 0013 8001");
    const byName = await mockStaffManagementApi.listAccounts({ nickname: "林医生" });

    expect(byPhone?.accountId).toBe("account-doctor-lin");
    expect(byName.items.map((item) => item.accountId)).toContain("account-doctor-lin");
  });

  it("promotes an account by stable account id and exposes it in the department", async () => {
    const account = await mockStaffManagementApi.getAccount("account-patient-wang");
    const promoted = await mockStaffManagementApi.promoteDoctor(
      account.accountId,
      "dept-radiology",
      { displayName: "王医生", staffNo: "D004" },
      account.managementVersion,
    );
    const doctors = await mockStaffManagementApi.listDoctors("dept-radiology");

    expect(promoted.identityType).toBe("doctor");
    expect(doctors.map((doctor) => doctor.doctorId)).toContain(account.accountId);
  });

  it("keeps doctor identity while disabled and restores directory visibility", async () => {
    const account = await mockStaffManagementApi.getAccount("account-doctor-lin");
    const disabled = await mockStaffManagementApi.setAccountEnabled(
      account.accountId,
      false,
      account.managementVersion,
    );
    expect(disabled.identityType).toBe("doctor");
    expect(await mockStaffManagementApi.listDoctors("dept-radiology")).toEqual([]);

    const restored = await mockStaffManagementApi.setAccountEnabled(
      disabled.accountId,
      true,
      disabled.managementVersion,
    );
    expect(restored.identityType).toBe("doctor");
    expect((await mockStaffManagementApi.listDoctors("dept-radiology")).length).toBe(1);
  });
});
