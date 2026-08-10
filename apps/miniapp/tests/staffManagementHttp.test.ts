import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("@/api/client", () => ({
  request: requestMock,
}));

describe("HTTP staff management organization adapter", () => {
  beforeEach(() => {
    requestMock.mockReset();
  });

  it("loads the public department directory for every campus", async () => {
    requestMock
      .mockResolvedValueOnce({ campuses: [{ campus_id: "campus-a" }, { campus_id: "campus-b" }] })
      .mockResolvedValueOnce({
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
      })
      .mockResolvedValueOnce({ items: [] });

    const { httpStaffManagementApi } = await import("@/api/staffManagement.http");
    const departments = await httpStaffManagementApi.listDepartments(false);

    expect(requestMock).toHaveBeenNthCalledWith(2, {
      path: "/api/v1/directory/departments?campus_id=campus-a",
      authenticated: false,
    });
    expect(requestMock).toHaveBeenNthCalledWith(3, {
      path: "/api/v1/directory/departments?campus_id=campus-b",
      authenticated: false,
    });
    expect(departments[0]?.parentId).toBe("campus-a");
  });

  it("loads all department statuses through the protected admin route", async () => {
    requestMock
      .mockResolvedValueOnce({ campuses: [{ campus_id: "campus-a" }] })
      .mockResolvedValueOnce({ items: [] });

    const { httpStaffManagementApi } = await import("@/api/staffManagement.http");
    await httpStaffManagementApi.listDepartments(true);

    expect(requestMock).toHaveBeenNthCalledWith(2, {
      path: "/api/v1/admin/identity/organization-units?unit_type=department&parent_id=campus-a&status=all",
      authenticated: true,
    });
  });
});
