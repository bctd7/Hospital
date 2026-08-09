<script setup lang="ts">
import { onLaunch, onShow } from "@dcloudio/uni-app";

import { restoreSession, sessionState } from "@/stores/session";
import { applyAppModeNavigation } from "@/utils/appMode";

onLaunch(() => {
  restoreSession();
  applyAppModeNavigation(sessionState.activeMode);
});

onShow(() => {
  applyAppModeNavigation(sessionState.activeMode);
});
</script>

<style>
page {
  min-height: 100%;
  background: #f4f7fb;
  color: #172033;
  font-family:
    -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB",
    "Microsoft YaHei", sans-serif;
}

view,
text {
  box-sizing: border-box;
}
</style>
