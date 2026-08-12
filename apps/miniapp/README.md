# 微信小程序

本目录是 Hospital 唯一客户端实现，技术栈为 uni-app、Vue 3、TypeScript 和 Vite。

README 只说明当前源码和运行方式。页面业务规则见 `plan/frontend/`，HTTP 字段见 `contracts/api/` 和
`docs/api/openapi.json`。

## 当前实现

- 首页、挂号、消息、我的四个 Tab；
- 手机号 + 阿里云 PNVS 验证码登录；
- Hospital Access/Refresh Token、单任务刷新和退出；
- 患者/工作人员应用版本切换；
- 医院院区上下文、科室和医生公共目录；
- 超级管理员院区/科室管理；
- 超级管理员账号查询、医生开通、资料编辑、调岗、撤销和账号启停；
- 本人展示昵称同步；
- 管理功能使用真实 HTTP，运行时 Mock 和 Mock 缓存已经删除。

预约、消息、就诊人等尚未接入的业务只显示真实空状态，不伪造后端数据。

## 源码结构

```text
apps/miniapp/
├── src/
│   ├── api/management/   # Identity/Organization HTTP Adapter
│   ├── api/              # 登录、会话和共享 HTTP Client
│   ├── services/         # 页面可复用的业务数据编排
│   ├── components/       # 可复用组件
│   ├── pages/            # 页面
│   ├── stores/           # 会话与应用状态
│   ├── types/            # 前端业务类型
│   └── utils/            # 无状态工具
├── tests/
├── package.json
└── vite.config.ts
```

页面不直接调用 `uni.request`。请求统一经过 API Client/Adapter，Token 刷新和错误转换只维护一份。

### 分层约束

- `api/client.ts` 只负责传输、Bearer Token、统一错误和单次刷新；
- `api/management/` 按公共组织目录、组织管理、账号管理拆分契约与 HTTP Adapter，不再提供聚合全部能力的兼容接口；
- `services/` 负责跨页面查询编排、短期缓存和失效，不包含页面跳转或弹窗；
- `stores/` 保存会话等跨页面状态；授权变化后的清理、回登录页和并发保护集中在 Session Store；
- `pages/` 组合用例和交互，组件通过 props/emit 复用，不根据名称反查业务主键；
- 园区、科室和账号写操作始终使用稳定 ID。同名科室通过“园区 / 科室”展示区分，提交仍使用 `department_id`。

## 本地运行

```powershell
Set-Location apps/miniapp
npm install
npm run dev:mp-weixin
```

微信开发者工具导入：

```text
apps/miniapp/dist/dev/mp-weixin
```

正式体验版构建：

```powershell
npm run build:mp-weixin
```

产物目录：

```text
apps/miniapp/dist/build/mp-weixin
```

正式构建固定读取 `.env.production` 并强制使用 CloudBase AnyService，不继承 `.env.local`、
`.env.production.local` 或当前 Shell 中的 `VITE_*`。构建结束会检查产物；发现局域网 API 地址、
缺失 CloudBase 环境或没有编译为 AnyService 调用时直接失败。不要绕过该脚本直接执行 `uni build`。

## API 地址

开发者工具模拟器可以访问 `127.0.0.1`；真机中的 `127.0.0.1` 指向手机自身，必须配置电脑局域网地址：

```dotenv
VITE_API_BASE_URL=http://192.168.x.x:8888
```

前端环境文件只保存公开连接信息，不能放 AppSecret、阿里云 AccessKey、JWT 私钥或手机号 HMAC Key。
本地真机调试只使用 `.env.local`；不要创建 `.env.production.local`。

## 登录联调

```text
POST /api/v1/auth/phone/code
  -> 用户收到验证码
POST /api/v1/auth/phone/login
  -> 保存 Hospital TokenPair
GET /api/v1/auth/me
  -> 恢复 Principal 和应用版本
```

微信登录接口保留兼容能力，但当前主入口使用手机号认证。接口字段和 Bearer 调试方式统一查看
[`docs/api/README.md`](../../docs/api/README.md)。

## 验证

```powershell
npm run test
npm run type-check
npm run build:mp-weixin
```

根目录 `scripts/check.ps1` 会运行上述检查。

## 安全约束

- 不记录手机号、验证码、Token 和微信临时 code；
- 不在 Storage 保存不必要的医疗敏感数据；
- 菜单隐藏不是权限校验，后端必须重新授权；
- 不将开发服务暴露到公网；
- 不执行可能破坏 DCloud 依赖兼容性的无评估强制升级。
