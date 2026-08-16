import { beforeEach, describe, expect, it, vi } from "vitest";

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }));

vi.mock("@/api/client", () => ({
  request: requestMock,
}));

describe("guidance HTTP adapter", () => {
  beforeEach(() => requestMock.mockReset());

  it("posts two unified location points to the authenticated walking route endpoint", async () => {
    requestMock.mockResolvedValueOnce({
      origin: { name: "A", address: "", latitude: 31.2, longitude: 121.4 },
      destination: { name: "B", address: "", latitude: 31.21, longitude: 121.41 },
      distance_meters: 820,
      duration_seconds: 640,
      polyline: [],
      steps: [],
      provider: "baidu",
    });
    const { guidanceApi } = await import("@/api/guidance");
    const input = {
      origin: { name: "A", address: "", latitude: 31.2, longitude: 121.4 },
      destination: { name: "B", address: "", latitude: 31.21, longitude: 121.41 },
    };

    await expect(guidanceApi.calculateWalkingRoute(input)).resolves.toMatchObject({
      provider: "baidu",
      distance_meters: 820,
    });
    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/guidance/routes/walking",
      method: "POST",
      data: input,
      authenticated: true,
    });
  });
});
