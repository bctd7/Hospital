# Go 服务

当前有四个可独立构建和运行的服务：

| 服务 | 入口 | 端口 | 职责 |
|---|---|---:|---|
| App API | `service/app/api/app.go` | 8888 | 小程序 HTTP、Token 中间件、协议转换与统一错误 |
| Identity RPC | `service/identity/rpc/identity.go` | 8080 | 登录、会话、账号、权限、组织和医生 |
| Appointment RPC | `service/appointment/rpc/appointment.go` | 8081 | 检查资源、预约、容量、报到、候检叫号、检查、报告和消息 |
| Guidance RPC | `service/guidance/rpc/guidance.go` | 8082 | 完整项目配置协调、检查规则、智能预约、当日顺序、地点检索和两点路线 |

```text
Miniapp -> App API -> Identity RPC
                   -> Appointment RPC
                   -> Guidance RPC -> Appointment RPC（项目事实、配置 TCC、整组预约）
                                   -> 高德 Web 服务
```

App API 不拥有业务数据库。需要业务数据时调用对应 RPC；Identity、Appointment 和 Guidance 分别拥有
`hospital_identity`、`hospital_appointment` 和 `hospital_guidance`，不得跨库查询。Guidance 需要检查项目事实时
调用 Appointment RPC，不通过前端传递，也不直接读取 Appointment 数据库。

## 服务内部约束

```text
server -> logic -> manager -> store -> repository
```

- Server/Logic：协议适配、Principal 读取和错误转换；
- Manager：权限、校验、状态机、幂等和事务；
- Store：由领域包声明的最小持久化端口；
- Repository：MySQL/Redis 的具体实现；
- ServiceContext：资源和 Worker 的装配与关闭。

详细导航：

- [App API](./app/README.md)；
- [Identity](./identity/README.md)；
- [Identity 内部结构](./identity/rpc/internal/README.md)；
- [Appointment](./appointment/README.md)；
- [Appointment 内部结构](./appointment/rpc/internal/README.md)；
- [Guidance](./guidance/README.md)。

## 文档分工

- `plan/backend` 记录业务目标、服务边界、稳定规则、已实现状态和后续提案；
- `service/*/README.md` 说明当前代码如何承载这些规则，包括调用关系、目录导航、配置、运行和验证；
- `contracts/api` 与 `contracts/proto` 是接口、字段和错误契约的唯一事实来源；
- Migration 是数据库结构的唯一事实来源，README 和 Plan 不复制完整表定义。

服务 README 可以概括主要业务能力，便于从代码入口理解服务，但不维护逐接口清单；模块 Plan 可以引用当前代码
结构说明落地位置，但不替代代码导航。两类文档都从服务职责和业务边界出发，不按页面临时需求或数据库表拆主题。

## 契约与生成

- HTTP：`contracts/api/app.api`；
- RPC：`contracts/proto/identity/v1/identity.proto`、`contracts/proto/appointment/v1/appointment.proto`、
  `contracts/proto/guidance/v1/guidance.proto`；
- 生成代码：`contracts/gen/`、各 RPC Client、Handler、Types 和 Server 骨架。

```powershell
.\scripts\contracts\generate-protobuf.ps1 -Check
goctl api validate -api contracts/api/app.api
goctl api go -api contracts/api/app.api -dir service/app/api --style go_zero
```

## 启动与检查

```powershell
.\scripts\development\start-backend.ps1 -Restart
.\scripts\quality\verify-repository.ps1
```

只有同时具备明确数据所有权、独立发布边界和真实调用方时才增加新服务；检查报告当前属于 Appointment，
不会仅为拆目录而单独建立 Report 服务。
