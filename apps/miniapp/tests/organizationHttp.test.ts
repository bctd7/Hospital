import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("@/api/client", () => ({
  request: requestMock,
}));

describe("organization HTTP adapters", () => {
  beforeEach(() => {
    requestMock.mockReset();
  });

  it("loads and maps the organization context", async () => {
    requestMock.mockResolvedValueOnce({
      hospital: {
        hospital_id: "hospital-a",
        code: "HOSPITAL",
        name: "测试医院",
        version: 1,
      },
      campuses: [
        {
          campus_id: "campus-a",
          hospital_id: "hospital-a",
          code: "CAMPUS-A",
          name: "本部院区",
          department_count: 0,
          status: "active",
          version: 1,
        },
      ],
    });

    const { httpOrganizationDirectoryApi } = await import(
      "@/api/management/organizationDirectory.http"
    );
    const context = await httpOrganizationDirectoryApi.getOrganizationContext();

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/directory/organization-context",
      authenticated: false,
    });
    expect(context.hospital.hospitalId).toBe("hospital-a");
    expect(context.campuses[0]?.campusId).toBe("campus-a");
  });

  it("loads the public directory for the selected campus", async () => {
    requestMock.mockResolvedValueOnce({
      items: [
        {
          department_id: "department-a",
          campus_id: "campus-a",
          code: "DEP-A",
          name: "内科",
          status: "active",
          version: 1,
        },
      ],
    });

    const { httpOrganizationDirectoryApi } = await import(
      "@/api/management/organizationDirectory.http"
    );
    const departments = await httpOrganizationDirectoryApi.listDepartments("campus-a");

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/directory/departments?campus_id=campus-a",
      authenticated: false,
    });
    expect(departments[0]?.parentId).toBe("campus-a");
  });

  it("maps a doctor's stable account id from the public directory", async () => {
    requestMock.mockResolvedValueOnce({
      items: [
        {
          account_id: "account-a",
          display_name: "测试医生",
          department_id: "department-a",
          version: 2,
        },
      ],
      page: 1,
      page_size: 100,
      total: 1,
    });

    const { httpOrganizationDirectoryApi } = await import(
      "@/api/management/organizationDirectory.http"
    );
    const doctors = await httpOrganizationDirectoryApi.listDoctors("department-a");

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/directory/departments/department-a/doctors?page=1&page_size=100",
      authenticated: false,
    });
    expect(doctors[0]?.accountId).toBe("account-a");
  });

  it("loads all department statuses through the protected admin route", async () => {
    requestMock.mockResolvedValueOnce({ items: [] });

    const { httpOrganizationAdminApi } = await import(
      "@/api/management/organizationAdmin.http"
    );
    await httpOrganizationAdminApi.listDepartments("campus-a", true);

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/admin/identity/organization-units?unit_type=department&parent_id=campus-a&status=all",
      authenticated: true,
    });
  });

  it("creates a campus with the hospital parent and a UUID operation id", async () => {
    requestMock.mockResolvedValueOnce({
      unit_id: "campus-a",
      parent_id: "hospital-a",
      code: "CAMPUS-A",
      name: "本部院区",
      status: "active",
      version: 1,
    });

    const { httpOrganizationAdminApi } = await import(
      "@/api/management/organizationAdmin.http"
    );
    await httpOrganizationAdminApi.createCampus({
      name: "本部院区",
      hospitalId: "hospital-a",
    });

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        path: "/api/v1/admin/identity/organization-units",
        method: "POST",
        authenticated: true,
        data: expect.objectContaining({
          unit_type: "campus",
          parent_id: "hospital-a",
          operation_id: expect.stringMatching(
            /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
          ),
        }),
      }),
    );
  });

  it("uses a POST action route when disabling an organization unit", async () => {
    requestMock.mockResolvedValueOnce({
      unit_id: "department-a",
      parent_id: "campus-a",
      code: "DEP-A",
      name: "内科",
      status: "disabled",
      version: 2,
    });

    const { httpOrganizationAdminApi } = await import(
      "@/api/management/organizationAdmin.http"
    );
    await httpOrganizationAdminApi.setDepartmentEnabled("department-a", false, 1);

    expect(requestMock).toHaveBeenCalledWith(
      expect.objectContaining({
        path: "/api/v1/admin/identity/organization-units/department-a/disable",
        method: "POST",
      }),
    );
  });
});
