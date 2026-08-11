import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  DepartmentSummary,
  OrganizationContext,
} from "@/types/staffManagement";

const apiMocks = vi.hoisted(() => ({
  getOrganizationContext: vi.fn(),
  listDirectoryDepartments: vi.fn(),
  listDoctors: vi.fn(),
  listCampuses: vi.fn(),
  listManagedDepartments: vi.fn(),
}));

vi.mock("@/api/staffManagement", () => ({
  organizationDirectoryApi: {
    getOrganizationContext: apiMocks.getOrganizationContext,
    listDepartments: apiMocks.listDirectoryDepartments,
    listDoctors: apiMocks.listDoctors,
  },
  organizationAdminApi: {
    listCampuses: apiMocks.listCampuses,
    listDepartments: apiMocks.listManagedDepartments,
  },
}));

const organizationContext: OrganizationContext = {
  hospital: {
    hospitalId: "hospital-a",
    code: "H-A",
    name: "Hospital A",
    version: 1,
  },
  campuses: [
    {
      campusId: "campus-a",
      hospitalId: "hospital-a",
      code: "C-A",
      name: "Campus A",
      departmentCount: 1,
      status: "active",
      version: 1,
    },
  ],
};

const departments: DepartmentSummary[] = [
  {
    departmentId: "department-a",
    parentId: "campus-a",
    code: "D-A",
    name: "Department A",
    doctorCount: 1,
    status: "active",
    version: 1,
  },
];

describe("organization service", () => {
  beforeEach(() => {
    vi.resetModules();
    Object.values(apiMocks).forEach((mock) => mock.mockReset());
  });

  it("deduplicates concurrent organization context requests and returns safe copies", async () => {
    let resolveRequest!: (value: OrganizationContext) => void;
    apiMocks.getOrganizationContext.mockReturnValue(
      new Promise<OrganizationContext>((resolve) => {
        resolveRequest = resolve;
      }),
    );
    const service = await import("@/services/organization");

    const firstRequest = service.loadOrganizationContext();
    const secondRequest = service.loadOrganizationContext();
    expect(apiMocks.getOrganizationContext).toHaveBeenCalledTimes(1);

    resolveRequest(organizationContext);
    const [first, second] = await Promise.all([firstRequest, secondRequest]);
    first.hospital.name = "locally changed";

    expect(second.hospital.name).toBe("Hospital A");
    expect((await service.loadOrganizationContext()).hospital.name).toBe("Hospital A");
    expect(apiMocks.getOrganizationContext).toHaveBeenCalledTimes(1);
  });

  it("separates public and managed department caches and supports precise invalidation", async () => {
    apiMocks.listDirectoryDepartments.mockResolvedValue(departments);
    apiMocks.listManagedDepartments.mockResolvedValue([
      ...departments,
      { ...departments[0], departmentId: "department-disabled", status: "disabled" },
    ]);
    const service = await import("@/services/organization");

    await service.loadDepartments("campus-a");
    await service.loadDepartments("campus-a");
    await service.loadDepartments("campus-a", true);
    expect(apiMocks.listDirectoryDepartments).toHaveBeenCalledTimes(1);
    expect(apiMocks.listManagedDepartments).toHaveBeenCalledTimes(1);

    service.invalidateDepartments("campus-a");
    await service.loadDepartments("campus-a");
    expect(apiMocks.listDirectoryDepartments).toHaveBeenCalledTimes(2);
  });
});
