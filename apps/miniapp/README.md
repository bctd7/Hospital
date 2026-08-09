# Mini App

这里用于放置微信小程序源码。当前尚未绑定正式 AppID，前端技术栈为 uni-app + Vue 3 + TypeScript + Vite。

规划文档集中维护在 [`plan/miniapp/`](../../plan/miniapp/)，不在源码目录中混放需求和设计文档。

## 当前实现

- 首页、挂号、消息、我的四个独立一级页面；
- 使用 `pages.json` 配置的原生 TabBar；
- 页面共享统一的应用外壳和空状态组件；
- 当前不发起任何后端业务请求。

## 目录结构

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

尚未接入的目录会在对应业务开始时创建，不使用空目录或无意义的占位依赖提前填充工程。

## 本地运行

安装依赖：

```powershell
cd apps/miniapp
npm install
```

启动微信小程序开发构建：

```powershell
npm run dev:mp-weixin
```

命令会持续监听源码变化。日常开发可以直接在微信开发者工具中导入小程序工程目录：

```text
apps/miniapp
```

根目录的 `project.config.json` 会把 `dist/build/mp-weixin` 识别为小程序根目录。首次导入前需要执行一次 `npm run build:mp-weixin`，确保其中已经生成 `app.json`。请勿把 `apps/miniapp` 的上一级目录或 `src` 目录直接作为小程序根目录。

首次导入时可以使用测试号；取得正式 AppID 后，在微信开发者工具或 `src/manifest.json` 的 `mp-weixin.appid` 中配置。开发环境访问本地后端时，可以在微信开发者工具中暂时关闭合法域名校验；真机联调仍需要手机可访问的 HTTPS 地址或局域网地址。

生产构建：

```powershell
npm run type-check
npm run build:mp-weixin
```

生产构建输出到：

```text
apps/miniapp/dist/build/mp-weixin
```

## 依赖安全说明

当前依赖版本来自 DCloud 官方 Vue 3/Vite TypeScript 模板，并由 `package-lock.json` 锁定。初始化时 `npm audit` 报告的问题来自 DCloud 编译器及其 Babel、Vite、国际化、压缩和图片处理等传递依赖；`npm audit fix --force` 会把 DCloud 包替换为不兼容版本，因此不能直接执行。

在 DCloud 发布兼容修复前：

- 不使用本地开发服务编译来源不可信的源码、样式或图片；
- 不把开发服务暴露到公网；
- 更新 DCloud 编译器前同时执行类型检查、微信构建和开发者工具回归；
- 生成的小程序产物不包含 Node.js 编译器本身，但仍需要持续跟踪工具链公告。

小程序初始化后，应保证：

- 微信临时 `code` 只发送给后端；
- AppSecret 不进入小程序代码；
- API Client 统一处理 Token、超时和错误码；
- 不在本地长期保存不必要的敏感数据；
- 请求只访问 HTTPS 合法域名。
