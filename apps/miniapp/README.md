# Mini App

这里用于放置微信小程序源码。当前尚未绑定 AppID，也没有提前选择脚手架。

首期建议使用原生微信小程序 + TypeScript，并至少分为：

```text
miniapp/
├── api/          # 统一 HTTP Client
├── components/   # 可复用组件
├── pages/        # 页面
├── stores/       # 状态管理
├── types/        # 前端类型
└── utils/        # 无业务状态的工具
```

小程序初始化后，应保证：

- 微信临时 `code` 只发送给后端；
- AppSecret 不进入小程序代码；
- API Client 统一处理 Token、超时和错误码；
- 不在本地长期保存不必要的敏感数据；
- 请求只访问 HTTPS 合法域名。
