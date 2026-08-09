<script setup lang="ts">
import { onLaunch } from "@dcloudio/uni-app";

import { restoreSession } from "@/stores/session";

onLaunch(() => {
  restoreSession();
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
