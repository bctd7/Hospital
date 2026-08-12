# Hospital

Hospital 是面向医院检查预约、院内流程规划和组织管理的微信小程序项目。后端使用 Go 与 go-zero，客户端使用
uni-app、Vue 3 和 TypeScript。

## 当前阶段

Identity 与组织管理已经完成首期闭环：

- 阿里云 PNVS 手机验证码登录、Access/Refresh Token 和授权版本校验；
- 医院、院区、科室公共目录及管理员组织 CRUD；
- 账号列表、详情、手机号精确搜索、停用与恢复；
- 医生开通、资料编辑、调岗和撤销；
- `operation_id` 幂等、乐观锁、授权审计，以及 Outbox → Kafka → Redis 授权版本投影；
- 小程序真实 HTTP 接入，运行时 Mock 已移除；
- HTTP → App API → Identity RPC → Manager → Repository → MySQL 全链路测试。

下一阶段进入预约检查服务设计与最小业务闭环，参见
[预约检查服务计划](./plan/backend/modules/05-appointment-and-examination-booking.md)。

## 架构与目录

```text
微信小程序
  -> app-api :8888               对外 HTTP、Token 中间件和页面聚合
  -> identity-rpc :8080          认证、账号、权限、组织和医生领域
  -> MySQL                      业务事实、审计和 Outbox
  -> Kafka -> Redis             授权事件传递与版本投影
```

| 目录 | 职责 |
|---|---|
| `apps/miniapp/` | 微信小程序页面、组件、服务层和 HTTP Client |
| `contracts/api/` | go-zero HTTP 契约源文件 |
| `contracts/proto/` | 内部 gRPC 契约源文件 |
| `contracts/events/` | Outbox/Kafka 事件契约 |
| `service/app/api/` | 面向客户端的 App API |
| `service/identity/rpc/` | Identity RPC 与领域实现 |
| `migrations/identity/` | Identity 数据库迁移 |
| `common/` | 认证、授权和可观测性等跨服务基础能力 |
| `docs/api/` | 生成后的 Swagger/OpenAPI 文档及预览说明 |
| `plan/` | 当前有效的业务、架构和交付计划 |

服务内部的细节分别见 [Identity README](./service/identity/README.md)、
[契约 README](./contracts/README.md) 和 [规划索引](./plan/README.md)。

## 本地启动

准备 `.env` 并启动基础设施：

```powershell
Copy-Item .env.example .env
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  up -d mysql redis kafka
```

首次使用或迁移升级：

```powershell
.\scripts\db-bootstrap-local.ps1
.\scripts\migrate.ps1 -Service identity -Direction up
```

统一启动后端：

```powershell
.\scripts\start-backend.ps1 -Restart
```

该脚本按 UTF-8 加载 `.env`，构建并启动 `identity-rpc` 与 `app-api`。健康检查：

```text
GET http://127.0.0.1:8888/api/v1/health
```

小程序开发：

```powershell
Set-Location apps/miniapp
npm install
npm run dev:mp-weixin
```

## 接口文档

HTTP 契约入口是 `contracts/api/app.api`，Swagger 快照位于
[`docs/api/openapi.json`](./docs/api/openapi.json)。生成、预览和 Bearer Token 调试方式见
[HTTP 接口文档](./docs/api/README.md)。

```powershell
goctl api validate -api contracts/api/app.api
goctl api swagger --api contracts/api/app.api --dir docs/api --filename openapi
```

`.api` 是接口事实来源，Swagger 是生成物；不要直接修改 `openapi.json`。

## 后端新增功能

1. 在 `plan/` 明确业务边界、权限、状态机、数据归属和验收标准；
2. 先修改 HTTP/Proto/事件契约，再生成代码；
3. Handler 只做协议适配，Logic 负责用例编排，Manager 负责领域规则；
4. Store/Repository 负责数据库访问，事务由 Manager 定义边界；
5. 写操作统一考虑幂等、乐观锁、审计、Outbox 和权限版本；
6. 补齐单元、数据库集成和 HTTP 全链路测试；
7. 更新 Swagger 与相关 README，运行全量检查后提交。

详细规则见 [后端开发指南](./plan/backend/README.md)。

## 验证

```powershell
.\scripts\check.ps1
```

该脚本校验 API 契约、事件 Schema、Compose、Go 测试与 Vet、小程序测试、类型检查和微信小程序构建。

## 安全约束

- `.env`、AccessKey、AppSecret、Token 密钥和生产凭据不得提交；
- 完整手机号、验证码、Access/Refresh Token 不得进入日志、Swagger 示例或测试快照；
- 小程序只调用 App API，禁止直接访问 RPC 或业务数据库；
- 生产环境禁止使用本地固定验证码 Provider；
- 跨服务不得直接读写其他服务的数据表。
