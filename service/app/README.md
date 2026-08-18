# App API

App API 是小程序唯一直接访问的后端 HTTP 入口，也是 Identity、Appointment 和 Guidance RPC 的无状态适配层。
它负责认证中间件、HTTP/RPC 协议转换、统一错误和请求链路，不拥有业务数据库，也不重新实现三个领域服务的
状态机。后端整体边界见
[后端架构与数据边界](../../plan/backend/01-architecture-and-data-boundaries.md)，业务规则分别以对应模块 Plan 和
服务 README 为准。

## 当前职责

- 暴露小程序需要的认证、组织目录、账号管理、检查预约和智能导诊 HTTP 能力；
- 校验 Access Token 的签名、签发方、受众、有效期和 Redis 授权版本；
- 将原始 Access Token、请求 ID 和调用截止时间传递给下游 RPC；
- 把 HTTP 请求转换为 RPC 命令，把 RPC 结果转换为稳定的小程序响应；
- 把领域错误统一映射成 HTTP 状态、业务错误码和可展示消息；
- 对健康检查、访问日志、Tracing、Metrics、请求大小和超时使用统一中间件；
- 对手机号、验证码、Refresh Token 和检查报告正文等敏感 RPC 禁止记录请求或响应正文。

App API 可以进行协议级字段整形和页面所需的轻量响应组合，但不能决定权限、预约容量、检查状态或推荐顺序。
需要跨服务完成的业务流程由明确的领域协调者承担，例如 Guidance 协调完整项目配置和智能预约，App API 只调用
Guidance 的公开 RPC。

## 调用关系

```text
Miniapp
  -> App API Handler
  -> App API Logic
      -> Identity RPC
      -> Appointment RPC
      -> Guidance RPC
```

- Identity 负责登录、会话、账号、权限、医院组织和医生档案；
- Appointment 负责检查资源、预约、队列、检查、报告和消息；
- Guidance 负责项目规则、智能预约、当日顺序和地图路线；
- App API 不跨库查询，也不缓存领域业务数据。

受保护请求先由 App API 校验 Token 和授权版本，再把同一 Token 作为 gRPC Metadata 转发。下游 RPC 的拦截器会
再次校验并将 Principal 写入 Context，真正的角色、科室和资源权限仍由领域 Manager 判断。公开登录接口不伪造
Principal，也不经过受保护路由中间件。

## 代码结构

```text
service/app/api/
├─ app.go                         配置加载、HTTP Server、中间件和依赖生命周期
├─ etc/app-api.yaml               本地服务、RPC、Redis 和 Token 配置
└─ internal/
   ├─ handler/                    HTTP 路由入口；由 goctl 生成骨架并按能力分包
   ├─ logic/                      HTTP/RPC DTO 转换、Metadata 传递和错误返回
   ├─ middleware/                 Access Token 与授权版本校验
   ├─ httperror/                  统一 HTTP 错误映射
   ├─ svc/                        RPC Client、Redis 和中间件装配、关闭
   ├─ config/                     配置结构
   └─ types/                      由 API 契约生成的 HTTP 类型
```

`handler` 与 `logic` 目录按照对外能力分组，例如 `auth`、`organizationdirectory`、`appointmentbookings`、
`guidanceplanning`。这些分组只是 HTTP 适配导航，不是新的领域包，也不拥有 Store、Repository 或业务事务。

阅读一条请求时按 `routes → handler → logic → RPC Client → 对应服务 Logic/Manager` 追踪。若业务判断出现在 App API
Logic，应优先判断它是否应该下沉到真正的数据拥有者，而不是继续在网关堆条件。

## 契约与生成代码

HTTP 路由、请求和响应字段的唯一事实来源是 `contracts/api/app.api`。本 README 只说明服务职责，不维护接口清单。
修改契约后统一重新生成 Handler、Types 和路由骨架，再在已有分包中恢复必要的手写适配；生成物不得成为第二份
业务规则来源。

```powershell
goctl api validate -api contracts/api/app.api
goctl api go -api contracts/api/app.api -dir service/app/api --style go_zero
```

RPC 字段分别以 `contracts/proto/identity/v1/identity.proto`、`appointment/v1/appointment.proto` 和
`guidance/v1/guidance.proto` 为准。App API 不复制这些 Proto 模型作为内部持久化对象。

## 配置与运行

`app-api.yaml` 只保存结构和环境变量占位。运行时需要三个 RPC 地址、Identity Access Token 公钥和授权版本 Redis。
App API 没有 MySQL DSN；如果新增代码需要直接连接业务数据库，说明服务边界已经被破坏。

```powershell
.\scripts\development\start-backend.ps1 -Restart
go test ./service/app/api/...
goctl api validate -api contracts/api/app.api
.\scripts\quality\verify-repository.ps1
```

Guidance 的模型解析和路线调用链比普通 RPC 更长，当前 App API 与 Guidance Client 预留 30 秒超时；其他超时仍由
统一服务配置和请求 Context 控制，不能在单个 Handler 中无限延长。

## 新增能力规则

1. 先在对应模块 Plan 确认业务拥有者、权限、状态和失败语义；
2. 在 `contracts/` 修改 HTTP/RPC 契约并生成代码；
3. App API 只增加协议适配，下游 Manager 实现业务规则；
4. 敏感字段必须加入访问日志或 RPC 正文屏蔽；
5. 补充 Handler/Logic 测试和至少一条真实 HTTP → RPC 链路验证；
6. 同步对应服务 README，不在 App API README 复制领域规则全文。

## 当前不包含

- 业务数据库和 Repository；
- Appointment 或 Guidance 热点业务缓存；
- 预约、报告、组织或导诊的第二套权限判断；
- 通过前端中转服务间业务事实；
- 为页面临时拼接而长期保存跨服务副本。

若未来需要稳定的跨服务查询模型，应先在 Plan 中明确数据所有权、同步方式和失效语义，再决定由现有领域服务
提供聚合 RPC，还是新增明确的数据产品；不能把 App API 变成隐式业务数据库。
