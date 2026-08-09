<script setup lang="ts">
import { onLaunch } from "@dcloudio/uni-app";

onLaunch(() => {
  // 应用启动阶段只初始化前端基础能力，不阻塞一级页面展示。
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
