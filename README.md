# Hospital

Hospital 是面向医院检查预约与组织管理的微信小程序项目。后端使用 Go 与 go-zero，客户端使用 uni-app、
Vue 3 和 TypeScript。

## 当前能力

Identity 与 Appointment 当前规划范围均已落地：

- 手机号验证码登录、Access/Refresh Token 和授权版本失效；
- 医院、院区、科室目录，账号与医生管理；
- 检查项目、严格地址房间、房间—项目关系和独立周窗口；
- 本周预约、共享容量、防重复预约、每周额度和未到场处理；
- 患者检查报到、房间候检队列、一分钟叫号、过号顺延和现场检查；
- 检查结束、报告待完成、报告模板、草稿、发布与不可覆盖的更正版本；
- 患者与科室消息、逐账号已读状态；
- 患者端与工作人员端小程序页面及真实 HTTP 接入。

完整状态见 [规划索引](./plan/README.md)。

## 架构

```text
微信小程序
  -> app-api :8888
       -> identity-rpc :8080
       -> appointment-rpc :8081

Identity -> MySQL / Redis / Aliyun PNVS / Kafka
Appointment -> MySQL / Redis
```

| 目录 | 职责 |
|---|---|
| `apps/miniapp/` | 微信小程序页面、组件、服务层和 HTTP Client |
| `contracts/api/` | 对外 HTTP 契约源文件 |
| `contracts/proto/` | 内部 gRPC 契约源文件 |
| `contracts/events/` | Outbox/Kafka 事件契约 |
| `service/app/api/` | 面向小程序的 App API |
| `service/identity/rpc/` | 认证、账号、权限、组织和医生领域 |
| `service/appointment/rpc/` | 检查资源、预约、容量、报告和消息领域 |
| `migrations/` | Identity 与 Appointment 数据库版本事实 |
| `common/` | 认证、授权与可观测性等跨服务技术能力 |
| `plan/` | 当前有效设计、已实现归档与后续提案 |

## 本地启动

```powershell
Copy-Item .env.example .env
.\scripts\db-bootstrap-local.ps1
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service appointment -Direction up
.\scripts\start-backend.ps1 -Restart
```

健康检查：

```text
GET http://127.0.0.1:8888/api/v1/health
```

小程序开发：

```powershell
Set-Location apps/miniapp
npm install
npm run dev:mp-weixin
```

微信开发者工具导入 `apps/miniapp/dist/dev/mp-weixin`。体验版构建和终端上传见
[小程序 README](./apps/miniapp/README.md)。

## 契约

HTTP 契约入口是 `contracts/api/app.api`，内部 RPC 契约位于 `contracts/proto/`。仓库当前不提交生成式
OpenAPI 快照，路径和字段直接以契约源文件为准。

```powershell
goctl api validate -api contracts/api/app.api
```

生成代码中标记为 `DO NOT EDIT` 的文件不得单独修改。业务规则进入 Manager，数据库访问进入 Repository，
依赖创建进入 ServiceContext。

## 测试

```powershell
.\scripts\check.ps1
```

该脚本校验契约、事件 Schema、Compose、Go Test/Vet、小程序测试、TypeScript，以及开发版和发布版微信小程序
构建。

## 安全约束

- `.env`、AccessKey、Token 密钥和生产凭据不得提交；
- 完整手机号、验证码、Token 和检查报告正文不得进入日志或测试快照；
- 小程序只调用 App API，服务不得直接读写其他服务数据库；
- 生产环境禁止本地固定验证码；
- Redis 只用于会话、授权版本与缓存，不能替代 MySQL 的业务裁决。
