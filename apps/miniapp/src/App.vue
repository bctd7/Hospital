<script setup lang="ts">
import { onLaunch, onShow } from "@dcloudio/uni-app";
import { watch } from "vue";

import { initializeCloudBase } from "@/platform/cloudbase";
import {
  initializeDevelopmentSession,
  restoreSession,
  sessionState,
} from "@/stores/session";
import { applyAppVariantNavigation } from "@/utils/appShell";

onLaunch(() => {
  initializeCloudBase();
  restoreSession();
  initializeDevelopmentSession();
  applyAppVariantNavigation(sessionState.appVariant);
});

onShow(() => {
  applyAppVariantNavigation(sessionState.appVariant);
});

watch(
  () => sessionState.appVariant,
  (variant) => applyAppVariantNavigation(variant),
);
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
