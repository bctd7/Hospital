# Mini App

这里用于放置微信小程序源码。当前使用测试 AppID，前端技术栈为 uni-app + Vue 3 + TypeScript + Vite。

前端接口契约和页面设计集中维护在 [`plan/frontend/`](../../plan/frontend/)，不在源码目录中混放需求和设计文档。

## 当前实现

- 患者端保留首页、挂号、消息、我的四个一级页面；
- 工作人员端第二页为可复用的部门管理页：医生只读，超级管理员可维护部门并进入用户管理；
- 超级管理员可按完整手机号精确搜索，也可按昵称或医生名称筛选账号；用户详情承载开通医生、编辑资料、调岗、撤销医生身份及账号启停；
- 使用 `pages.json` 配置的原生 TabBar；
- 启动时展示可跳过的微信头像和昵称填写页；
- “我的”页面展示资料头部及就诊人管理、预约、关注、设置和消息管理入口；
- 页面共享统一的应用外壳和空状态组件；
- 已接入微信 code 登录、Hospital Token、当前身份、单任务刷新和退出会话；
- 登录失败时以访客身份进入，不在前端接收、保存或展示 OpenID。

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

API 默认地址为 `http://127.0.0.1:8888`。需要覆盖时，在 `apps/miniapp/.env.local` 中配置：

```text
VITE_API_BASE_URL=http://电脑的局域网地址:8888
VITE_STAFF_DATA_SOURCE=mock
```

`VITE_STAFF_DATA_SOURCE=mock` 在开发服务中生效。部门和用户演示数据全部隔离在
`src/api/staffManagement.mock.ts`，页面只调用 `StaffManagementApi`，不直接引用静态数据。
后端接口完成后把本地配置改为 `http` 即可联调，普通生产构建无条件使用 HTTP。

后端尚未完成、需要从 `apps/miniapp` 直接导入微信开发者工具查看演示数据时，先执行：

```powershell
npm run build:mp-weixin:mock
```

该命令显式生成 Mock 预览包到 `dist/build/mp-weixin`。发布前必须改用
`npm run build:mp-weixin`；普通生产构建会剔除演示账号数据。

后端联调完成后的 Mock 清理清单：

1. 对照 `plan/frontend/api/` 完成真实接口回归；
2. 删除 `staffManagement.mock.ts` 及对应 Mock 测试；
3. 移除 `staffManagement.ts` 中的 Mock 选择和页面中的 Mock 提示；
4. 删除 `VITE_STAFF_DATA_SOURCE` 配置，只保留 HTTP 实现。

微信开发者工具模拟器可以访问本机 `127.0.0.1`；真机中的 `127.0.0.1` 指向手机自身，必须改为手机可访问的局域网或 HTTPS 地址。API Base URL 是公开的前端配置，不得在这里放 AppSecret。

启动微信小程序开发构建：

```powershell
npm run dev:mp-weixin
```

命令会持续监听源码变化，热更新模式下微信开发者工具需要导入：

```text
apps/miniapp/dist/dev/mp-weixin
```

如果使用 `npm run build:mp-weixin` 进行普通构建，也可以直接导入 `apps/miniapp`。根目录的 `project.config.json` 会把 `dist/build/mp-weixin` 识别为小程序根目录。首次导入前需要完成一次构建，确保其中已经生成 `app.json`。请勿把 `apps/miniapp` 的上一级目录或 `src` 目录直接作为小程序根目录。

如果开发者工具提示某个 `components/*.js`“已被代码依赖分析忽略”，请确认使用最新构建产物并重新
编译；工程已同时关闭开发期的 `ignoreDevUnusedFiles` 和上传期的 `ignoreUploadUnusedFiles`，避免
uni-app 生成的局部组件被误判。开发者工具生成的 `project.private.config.json` 优先级更高；旧工程
仍报错时，在“详情 → 本地设置”关闭“过滤无依赖文件”，或者删除该私有配置后重新导入
`apps/miniapp`。

当前测试 AppID 同时配置在 `src/manifest.json` 和根目录 `project.config.json`。开发环境访问本地后端时，可以在微信开发者工具中暂时关闭合法域名校验；真机联调仍需要手机可访问的 HTTPS 地址或局域网地址。

## 登录联调

入口页点击“进入小程序”后的调用链：

```text
uni.login
  -> POST /api/v1/auth/wechat/login
  -> 保存 Hospital Access/Refresh Token
  -> GET /api/v1/auth/me
  -> 成功进入 authenticated，失败进入 guest
```

小程序只把一次性 `code` 作为 `login_code` 发给 App API。OpenID 和 `session_key` 由 Identity 服务向微信换取并留在后端；App API 返回的是 Hospital Token，不返回 OpenID。

本地真实登录前需要启动 App API、Identity RPC、MySQL 和 Redis，并在后端环境中配置 `WECHAT_MINIAPP_APP_ID`、`WECHAT_MINIAPP_APP_SECRET` 和签名密钥。不要把这些后端密钥复制到 `apps/miniapp`。

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
