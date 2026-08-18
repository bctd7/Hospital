export const NATIVE_MAP_SETTLE_DELAY_MS = 1500;
export const NATIVE_MAP_FALLBACK_DELAY_MS = 3500;

/**
 * 微信原生 map 的 updated 只表示视野数据已提交，并不代表地图瓦片和文字已经全部绘制。
 * 该控制器把多次 updated 合并成一次稳定揭示，避免用户看到空底、道路和 POI 逐层闪现。
 */
export function createStableNativeMapReveal(
  setReady: (ready: boolean) => void,
  settleDelayMs = NATIVE_MAP_SETTLE_DELAY_MS,
  fallbackDelayMs = NATIVE_MAP_FALLBACK_DELAY_MS,
) {
  let generation = 0;
  let revealTimer: ReturnType<typeof setTimeout> | undefined;
  let fallbackTimer: ReturnType<typeof setTimeout> | undefined;

  function clearTimers() {
    if (revealTimer) clearTimeout(revealTimer);
    if (fallbackTimer) clearTimeout(fallbackTimer);
    revealTimer = undefined;
    fallbackTimer = undefined;
  }

  function prepare() {
    generation += 1;
    const currentGeneration = generation;
    clearTimers();
    setReady(false);
    // 极少数基础库不会触发 updated，兜底避免地图永久停留在加载态。
    fallbackTimer = setTimeout(() => {
      if (currentGeneration === generation) setReady(true);
    }, fallbackDelayMs);
  }

  function updated() {
    const currentGeneration = generation;
    if (revealTimer) clearTimeout(revealTimer);
    revealTimer = setTimeout(() => {
      if (currentGeneration !== generation) return;
      if (fallbackTimer) clearTimeout(fallbackTimer);
      fallbackTimer = undefined;
      setReady(true);
    }, settleDelayMs);
  }

  function dispose() {
    generation += 1;
    clearTimers();
    setReady(false);
  }

  return { prepare, updated, dispose };
}
