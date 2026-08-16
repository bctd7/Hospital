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
      provider: "amap",
    });
    const { guidanceApi } = await import("@/api/guidance");
    const input = {
      origin: { name: "A", address: "", latitude: 31.2, longitude: 121.4 },
      destination: { name: "B", address: "", latitude: 31.21, longitude: 121.41 },
    };

    await expect(guidanceApi.calculateWalkingRoute(input)).resolves.toMatchObject({
      provider: "amap",
      distance_meters: 820,
    });
    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/guidance/routes/walking",
      method: "POST",
      data: input,
      authenticated: true,
    });
  });

  it("searches AMap places through the authenticated Guidance endpoint", async () => {
    requestMock.mockResolvedValueOnce({
      places: [{ name: "123号楼", address: "上海市宝山区", latitude: 31.4, longitude: 121.48, provider_place_id: "B001" }],
    });
    const { guidanceApi } = await import("@/api/guidance");
    const input = { keyword: "123号楼", city: "上海市", limit: 15 };

    await expect(guidanceApi.searchPlaces(input)).resolves.toMatchObject({
      places: [{ provider_place_id: "B001" }],
    });
    expect(requestMock).toHaveBeenCalledWith({
      path: "/api/v1/guidance/places/search",
      method: "GET",
      data: input,
      authenticated: true,
    });
  });
});
