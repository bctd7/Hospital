# Mini App

这里用于放置微信小程序源码。当前尚未绑定 AppID，前端技术方向确定为 uni-app + Vue 3 + TypeScript + Vite。

规划文档集中维护在 [`plan/miniapp/`](../../plan/miniapp/)，不在源码目录中混放需求和设计文档。

项目至少分为：

```text
miniapp/
├── src/
│   ├── api/          # 统一 HTTP Client
│   ├── components/   # 可复用组件
│   ├── pages/        # 页面
│   ├── stores/       # 状态管理
│   ├── styles/       # 主题变量和全局样式
│   ├── types/        # 前端类型
│   └── utils/        # 无业务状态的工具
├── package.json
└── vite.config.ts
```

小程序初始化后，应保证：

- 微信临时 `code` 只发送给后端；
- AppSecret 不进入小程序代码；
- API Client 统一处理 Token、超时和错误码；
- 不在本地长期保存不必要的敏感数据；
- 请求只访问 HTTPS 合法域名。
