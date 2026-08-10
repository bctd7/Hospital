import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("@/api/client", () => ({
  request: requestMock,
}));

describe("HTTP staff management organization adapter", () => {
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

    const { httpStaffManagementApi } = await import("@/api/staffManagement.http");
    const context = await httpStaffManagementApi.getOrganizationContext();

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/directory/organization-context",
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

    const { httpStaffManagementApi } = await import("@/api/staffManagement.http");
    const departments = await httpStaffManagementApi.listDepartments("campus-a", false);

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/directory/departments?campus_id=campus-a",
      authenticated: false,
    });
    expect(departments[0]?.parentId).toBe("campus-a");
  });

  it("loads all department statuses through the protected admin route", async () => {
    requestMock.mockResolvedValueOnce({ items: [] });

    const { httpStaffManagementApi } = await import("@/api/staffManagement.http");
    await httpStaffManagementApi.listDepartments("campus-a", true);

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

    const { httpStaffManagementApi } = await import("@/api/staffManagement.http");
    await httpStaffManagementApi.createCampus({
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
});
