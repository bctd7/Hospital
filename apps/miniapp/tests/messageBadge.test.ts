import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  listMine: vi.fn(),
  listDepartment: vi.fn(),
  rememberedDepartmentId: "",
  sessionState: {
    status: "authenticated",
    appVariant: "patient",
    principal: null as null | { roles: string[]; department_id?: string },
  },
}));

vi.mock("@/api/appointment", () => ({
  appointmentMessageApi: {
    listMine: mocks.listMine,
    listDepartment: mocks.listDepartment,
  },
}));
vi.mock("@/stores/session", () => ({ sessionState: mocks.sessionState }));
vi.mock("@/utils/appointmentManagement", () => ({
  hasRole: (principal: { roles?: string[] } | null, role: string) => principal?.roles?.includes(role) ?? false,
}));
vi.mock("@/utils/staffDepartmentContext", () => ({
  currentStaffDepartmentId: () => mocks.rememberedDepartmentId,
}));

describe("message tab badge", () => {
  beforeEach(() => {
    mocks.listMine.mockReset();
    mocks.listDepartment.mockReset();
    mocks.rememberedDepartmentId = "";
    mocks.sessionState.status = "authenticated";
    mocks.sessionState.appVariant = "patient";
    mocks.sessionState.principal = null;
    vi.stubGlobal("uni", {
      setTabBarBadge: vi.fn(),
      removeTabBarBadge: vi.fn(),
    });
  });

  it("loads the patient unread count", async () => {
    mocks.listMine.mockResolvedValue({ unreadCount: 4 });
    const { refreshMessageBadge } = await import("@/services/messageBadge");

    await refreshMessageBadge();

    expect(mocks.listMine).toHaveBeenCalledWith(1, 1);
    expect(uni.setTabBarBadge).toHaveBeenCalledWith({ index: 2, text: "4" });
  });

  it("locks a doctor to the principal department", async () => {
    mocks.sessionState.appVariant = "staff";
    mocks.sessionState.principal = { roles: ["department_doctor"], department_id: "department-doctor" };
    mocks.listDepartment.mockResolvedValue({ unreadCount: 2 });
    const { refreshMessageBadge } = await import("@/services/messageBadge");

    await refreshMessageBadge();

    expect(mocks.listDepartment).toHaveBeenCalledWith("department-doctor", 1, 1);
  });

  it("uses the administrator overview before a department is selected", async () => {
    mocks.sessionState.appVariant = "staff";
    mocks.sessionState.principal = { roles: ["super_admin"] };
    mocks.listDepartment.mockResolvedValue({ unreadCount: 12 });
    const { refreshMessageBadge } = await import("@/services/messageBadge");

    await refreshMessageBadge();

    expect(mocks.listDepartment).toHaveBeenCalledWith("", 1, 1);
    expect(uni.setTabBarBadge).toHaveBeenCalledWith({ index: 2, text: "12" });
  });
});
