# Go 服务

当前有四个可独立构建和运行的服务：

| 服务 | 入口 | 端口 | 职责 |
|---|---|---:|---|
| App API | `service/app/api/app.go` | 8888 | 小程序 HTTP、Token 中间件、协议转换与页面聚合 |
| Identity RPC | `service/identity/rpc/identity.go` | 8080 | 登录、会话、账号、权限、组织和医生 |
| Appointment RPC | `service/appointment/rpc/appointment.go` | 8081 | 检查资源、预约、容量、报到、候检叫号、检查、报告和消息 |
| Guidance RPC | `service/guidance/rpc/guidance.go` | 8082 | 检查项目先后规则、地点检索和两点步行路线；后续承载智能预约与当日路线 |

```text
Miniapp -> App API -> Identity RPC
                   -> Appointment RPC
                   -> Guidance RPC -> Appointment RPC（只读项目事实）
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

- [Identity](./identity/README.md)；
- [Identity 内部结构](./identity/rpc/internal/README.md)；
- [Appointment](./appointment/README.md)；
- [Appointment 内部结构](./appointment/rpc/internal/README.md)；
- [Guidance](./guidance/README.md)。

## 契约与生成

- HTTP：`contracts/api/app.api`；
- RPC：`contracts/proto/identity/v1/identity.proto`、`contracts/proto/appointment/v1/appointment.proto`、
  `contracts/proto/guidance/v1/guidance.proto`；
- 生成代码：`contracts/gen/`、各 RPC Client、Handler、Types 和 Server 骨架。

```powershell
goctl api validate -api contracts/api/app.api
goctl api go -api contracts/api/app.api -dir service/app/api --style go_zero
```

## 启动与检查

```powershell
.\scripts\start-backend.ps1 -Restart
.\scripts\check.ps1
```

只有同时具备明确数据所有权、独立发布边界和真实调用方时才增加新服务；检查报告当前属于 Appointment，
不会仅为拆目录而单独建立 Report 服务。
