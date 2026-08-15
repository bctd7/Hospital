import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("@/api/client", () => ({ request: requestMock }));

describe("appointment HTTP adapter", () => {
  beforeEach(() => requestMock.mockReset());

  it("passes the stable department id when listing examination items", async () => {
    requestMock.mockResolvedValueOnce({ items: [], page: 1, page_size: 50, total: 0 });
    const { appointmentManagementApi } = await import("@/api/appointment");

    await appointmentManagementApi.listItems("department-a", "active");

    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/admin/appointment/examination-items?owner_department_id=department-a&status=active&page=1&page_size=50",
      authenticated: true,
    });
  });

  it("normalizes null empty collections returned by Go JSON", async () => {
    requestMock
      .mockResolvedValueOnce({ rooms: null, page: 1, page_size: 50, total: 0 })
      .mockResolvedValueOnce({ relations: null, page: 1, page_size: 100, total: 0 });
    const { appointmentManagementApi } = await import("@/api/appointment");

    const rooms = await appointmentManagementApi.listRooms("department-empty");
    const relations = await appointmentManagementApi.listRoomItems("room-empty", "active");

    expect(rooms.items).toEqual([]);
    expect(relations.items).toEqual([]);
  });

  it("creates an examination item with a fresh idempotency id", async () => {
    requestMock.mockResolvedValueOnce({
      item_id: "item-a",
      owner_department_id: "department-a",
      name: "胸部 CT",
      description: "说明",
      status: "active",
      version: 1,
      created_at: "now",
      updated_at: "now",
    });
    const { appointmentManagementApi } = await import("@/api/appointment");

    const created = await appointmentManagementApi.createItem("department-a", "胸部 CT", "说明");

    expect(created.ownerDepartmentId).toBe("department-a");
    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({
      path: "/api/v1/admin/appointment/examination-items",
      method: "POST",
      authenticated: true,
      data: expect.objectContaining({
        owner_department_id: "department-a",
        operation_id: expect.stringMatching(/^[0-9a-f-]{36}$/),
      }),
    }));
  });

  it("uses the relation id and optimistic version when disabling a room item", async () => {
    requestMock.mockResolvedValueOnce({
      relation_id: "relation-a",
      room_id: "room-a",
      item_id: "item-a",
      room_display_name: "门诊楼 · 3层 · 301室",
      campus_id: "00000000-0000-4000-8000-000000000001",
      building: "门诊楼",
      floor_number: 3,
      room_number: "301",
      item_name: "胸部 CT",
      status: "disabled",
      version: 3,
      created_at: "now",
      updated_at: "now",
    });
    const { appointmentManagementApi } = await import("@/api/appointment");

    await appointmentManagementApi.setRoomItemEnabled({
      relationId: "relation-a",
      roomId: "room-a",
      itemId: "item-a",
      roomDisplayName: "门诊楼 · 3层 · 301室",
      campusId: "00000000-0000-4000-8000-000000000001",
      building: "门诊楼",
      floorNumber: 3,
      roomNumber: "301",
      itemName: "胸部 CT",
      status: "active",
      version: 2,
      createdAt: "now",
      updatedAt: "now",
    }, false);

    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({
      path: "/api/v1/admin/appointment/room-examination-items/relation-a/disable",
      method: "POST",
      data: expect.objectContaining({ resource_id: "relation-a", expected_version: 2 }),
    }));
  });

  it("keeps room and item weekly window payloads independent", async () => {
    requestMock.mockResolvedValueOnce({
      window_id: "window-a", room_id: "room-a", weekday: 1, session: "morning",
      open_time: "08:00", close_time: "12:00", active_capacity: 20,
      status: "active", version: 1, created_at: "now", updated_at: "now",
    });
    const { appointmentManagementApi } = await import("@/api/appointment");

    await appointmentManagementApi.saveRoomWindow("room-a", {
      weekday: 1,
      session: "morning",
      openTime: "08:00",
      closeTime: "12:00",
      activeCapacity: 20,
      expectedVersion: 0,
    });

    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({
      data: expect.objectContaining({
        open_time: "08:00",
        close_time: "12:00",
        active_capacity: 20,
      }),
    }));
    const payload = requestMock.mock.calls[0]?.[0]?.data;
    expect(payload).not.toHaveProperty("booking_cutoff_time");
  });

  it("reuses the operation id when an ambiguous write is retried", async () => {
    const response = {
      room_id: "room-retry", department_id: "department-a",
      campus_id: "00000000-0000-4000-8000-000000000001",
      building: "门诊楼", floor_number: 3, room_number: "301",
      display_name: "门诊楼 · 3层 · 301室",
      version: 1, created_at: "now", updated_at: "now",
    };
    requestMock.mockRejectedValueOnce(new Error("network timeout")).mockResolvedValueOnce(response);
    const { appointmentManagementApi } = await import("@/api/appointment");

    const location = {
      campusId: "00000000-0000-4000-8000-000000000001",
      building: "门诊楼",
      floorNumber: 3,
      roomNumber: "301",
    };
    await expect(appointmentManagementApi.createRoom("department-a", location)).rejects.toThrow();
    await appointmentManagementApi.createRoom("department-a", location);

    expect(requestMock.mock.calls[0]?.[0]?.data.operation_id).toBe(
      requestMock.mock.calls[1]?.[0]?.data.operation_id,
    );
  });

  it("maps patient messages and normalizes optional collections", async () => {
    requestMock.mockResolvedValueOnce({
      messages: null,
      page: 1,
      page_size: 50,
      total: 0,
      unread_count: 3,
      department_unread_counts: null,
    });
    const { appointmentMessageApi } = await import("@/api/appointment");

    const result = await appointmentMessageApi.listMine();

    expect(result.items).toEqual([]);
    expect(result.departmentUnreadCounts).toEqual([]);
    expect(result.unreadCount).toBe(3);
    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/appointment/messages?page=1&page_size=50",
      authenticated: true,
    });
  });

  it("requests the administrator unread overview without inventing a department", async () => {
    requestMock.mockResolvedValueOnce({
      messages: [], page: 1, page_size: 1, total: 0, unread_count: 7,
      department_unread_counts: [{ department_id: "department-a", unread_count: 7 }],
    });
    const { appointmentMessageApi } = await import("@/api/appointment");

    const result = await appointmentMessageApi.listDepartment("", 1, 1);

    expect(result.departmentUnreadCounts).toEqual([{ departmentId: "department-a", unreadCount: 7 }]);
    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/admin/appointment/messages?page=1&page_size=1",
      authenticated: true,
    });
  });
});
