# Hospital

Hospital 是面向微信小程序的医院检查业务平台。项目采用 Monorepo 管理客户端、Go 后端、接口契约、数据库迁移、基础设施和工程文档；后端以 go-zero 为基础，首期使用 MySQL、Redis，并按业务需要接入 Kafka。

当前项目处于工程基线阶段：已建立 API 服务骨架、健康检查契约和本地基础设施配置，业务流程、数据模型、微信登录和小程序页面尚待实现。

## 系统组成

```text
微信小程序
    -> HTTPS API（service/app/api）
        -> Handler：解析请求和输出响应
        -> Logic：业务用例、规则和权限校验
        -> Model：MySQL 数据访问
        -> Redis：缓存、限流和幂等
        -> Kafka：异步事件（按需接入）
```

首期以一个可部署的 API 服务完成核心业务闭环。只有在业务边界、数据所有权、独立发布或扩缩容需求明确后，才拆分 RPC 服务或异步 Consumer。

## 目录与开发位置

| 目录 | 职责 | 主要开发内容 |
|---|---|---|
| `apps/miniapp/` | 微信小程序客户端 | 页面、组件、API Client、状态管理和前端类型 |
| `contracts/api/` | 对外 HTTP 契约 | 在 `.api` 中定义路由、请求和响应，再用 goctl 生成代码 |
| `service/app/api/internal/logic/` | 后端业务层 | 业务规则、用例编排、权限判断和事务边界 |
| `service/app/api/internal/handler/` | HTTP 适配层 | 保持轻量；生成后通常只做必要的协议适配 |
| `service/app/api/internal/svc/` | 依赖装配 | 初始化并注入数据库、Redis、Kafka、RPC Client 等共享依赖 |
| `service/app/api/internal/config/` | 服务配置 | 声明 YAML 和环境变量对应的配置结构 |
| `service/app/api/internal/types/` | API 类型 | 由 `.api` 契约生成，不直接维护 |
| `service/app/api/internal/model/` | 数据访问层（待建立） | 数据表确定后放置 goctl 生成或手写的 Model 与查询代码 |
| `migrations/` | 数据库版本 | 按业务域维护可执行、可回滚的 SQL 迁移 |
| `common/` | 跨服务技术能力 | 稳定复用的认证、错误码、中间件、可观测性和事件基础设施 |
| `contracts/proto/` | 内部 RPC 契约 | 服务拆分后维护 Protobuf 定义 |
| `contracts/events/` | 事件契约 | Kafka 事件 Schema、版本和兼容性约定 |
| `deploy/` | 环境与部署 | Compose、镜像、代理和可观测性配置 |
| `tests/` | 跨模块测试 | 契约测试、API 测试和基础设施集成测试 |
| `plan/` | 架构文档 | 技术规划、业务边界、数据设计和实施路线 |

## 开发流程

一个后端功能按以下顺序交付：

1. 在 `plan/` 明确业务流程、角色、数据归属和验收标准。
2. 在 `contracts/api/app.api` 修改 HTTP 契约，并执行契约校验。
3. 使用 goctl 更新服务骨架；生成的路由和类型文件不手工修改。
4. 在 `internal/logic/` 实现业务，在 `internal/svc/` 装配依赖，在 Model 中实现数据访问。
5. 在 `migrations/` 增加数据库迁移，并为 Logic、API 和数据链路补充测试。
6. 在 `apps/miniapp/` 实现页面和 API 调用，完成端到端联调。
7. 通过构建、测试、契约校验和 Compose 配置校验后提交。

go-zero API 的主要调用链如下：

```text
app.go -> routes.go -> handler -> logic -> model / infrastructure
```

## 本地运行

准备本地配置：

```powershell
Copy-Item .env.example .env
```

启动 MySQL 和 Redis：

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  up -d mysql redis
```

需要验证异步事件链路时启动 Kafka：

```powershell
docker compose `
  --env-file .env `
  -f deploy/compose/docker-compose.yml `
  --profile messaging `
  up -d kafka
```

校验契约并构建：

```powershell
goctl api validate -api contracts/api/app.api
go mod tidy
go build ./...
go test ./...
```

启动 API 服务：

```powershell
go run ./service/app/api `
  -f service/app/api/etc/app-api.yaml
```

健康检查地址：`GET http://localhost:8888/api/v1/health`。

也可以执行基础校验脚本：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/check.ps1
```

## 工程约束

- `.api`、`.proto` 和事件 Schema 是契约源文件，契约变更后再生成代码。
- Handler 不承载复杂业务；主要业务代码放在 Logic。
- 数据库、Redis、Kafka 和 RPC Client 统一通过 `ServiceContext` 注入。
- 每项业务数据只有一个明确拥有者，禁止跨服务直接读写其他服务的数据表。
- Redis 只保存可过期或可重建的数据，不作为业务事实库。
- 新增基础组件时同步设计超时、错误处理、测试、监控和降级策略。
- `.env`、微信 AppSecret、Token 密钥和生产凭据不得提交到仓库。

详细方案见 [总体技术规划](./plan/00-overall-technical-plan.md)。

## Identity 服务请求链路

Identity 服务的一次完整请求按以下链路执行：

```text
前端
  -> HTTP / gRPC 请求
  -> API 服务
  -> Identity gRPC 服务
  -> gRPC 鉴权拦截器（验证 Token，将 Principal 放入 context）
  -> Server（接收 gRPC 请求并调用对应 Logic）
  -> Logic（取请求参数和 Principal，调用 Manager，转换响应和错误）
  -> Manager（权限校验、参数校验、业务规则和事务编排）
  -> Store（定义数据访问能力）
  -> mysqlstore（执行具体 SQL）
  -> MySQL
```

以“管理员查询组织单元详情”为例：

1. 前端传入 `unit_id`。
2. 请求到达 `IdentityService.GetOrganizationUnit`。
3. 鉴权拦截器验证 Token，并把当前用户 `Principal` 放入 `context`。
4. Server 创建 `GetOrganizationUnitLogic` 并将请求交给它。
5. Logic 从 `context` 中取出 `Principal`，调用 `OrganizationManager.GetManagedUnit`。
6. Manager 检查用户是否拥有组织管理权限，然后调用 `Store.GetUnit`。
7. mysqlstore 执行 SQL 查询 MySQL。
8. 查询结果逐层返回，Logic 将 `organization.Unit` 转换为 protobuf `OrganizationUnit`。
9. gRPC 响应返回 API 服务，最终返回前端。

各层职责与 Java 常见分层的对照：

| 项目分层 | Java 类比 | 主要职责 |
|---|---|---|
| Interceptor | Filter / Interceptor | 统一鉴权，将登录用户写入上下文 |
| Server | Controller 入口 | 接收 gRPC 请求并转交给 Logic |
| Logic | Controller / Service 适配层 | 拆解请求、获取用户、调用 Manager、组装响应 |
| Manager | Service | 实现权限检查和核心业务规则 |
| Store 接口 | Mapper 接口 | 定义业务所需的数据访问能力 |
| mysqlstore | Mapper 实现 | 编写并执行具体 SQL |

简单记忆：**Server 接请求，Logic 接线翻译，Manager 处理业务，Store 操作数据库，Interceptor 统一把守入口。**
