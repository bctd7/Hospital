import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { createStableNativeMapReveal } from "../src/features/guidance/stableNativeMap";

describe("原生地图稳定揭示", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("等待最后一次 updated 安静后再显示地图", () => {
    let ready = true;
    const controller = createStableNativeMapReveal((value) => { ready = value; }, 1500, 3500);

    controller.prepare();
    controller.updated();
    vi.advanceTimersByTime(1200);
    controller.updated();
    vi.advanceTimersByTime(1499);
    expect(ready).toBe(false);

    vi.advanceTimersByTime(1);
    expect(ready).toBe(true);
  });

  it("没有 updated 时使用兜底计时结束加载态", () => {
    let ready = true;
    const controller = createStableNativeMapReveal((value) => { ready = value; }, 1500, 3500);

    controller.prepare();
    vi.advanceTimersByTime(3499);
    expect(ready).toBe(false);
    vi.advanceTimersByTime(1);
    expect(ready).toBe(true);
  });

  it("新一轮地图或销毁会取消上一轮计时", () => {
    let ready = true;
    const controller = createStableNativeMapReveal((value) => { ready = value; }, 1500, 3500);

    controller.prepare();
    controller.updated();
    controller.prepare();
    vi.advanceTimersByTime(1500);
    expect(ready).toBe(false);

    controller.dispose();
    vi.advanceTimersByTime(3500);
    expect(ready).toBe(false);
  });
});
