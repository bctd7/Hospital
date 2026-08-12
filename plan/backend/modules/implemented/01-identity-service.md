# Identity Service 首期实施归档

> 状态：首期实现完成并通过阶段验收。
>
> 本文合并原“Identity 身份与访问控制”和“Identity 组织、科室、医生与用户管理”两个 Plan，统一描述
> 同一个 Identity Service 的当前有效边界。接口字段以 `contracts/api/`、
> `contracts/proto/identity/v1/identity.proto` 和生成的 `docs/api/openapi.json` 为准。

## 1. 服务职责与完成范围

Identity Service 是身份、账号、工作人员角色和组织主数据的事实源，已经完成：

- 阿里云 PNVS 手机号验证码登录和兼容的微信外部身份；
- Access Token、Refresh Session、Token 轮换、退出和重放防护；
- `super_admin`、`department_doctor` 角色和 permission 快照；
- 授权版本的 MySQL 事实、Outbox、Kafka 发布和 Redis 单调投影；
- 单医院下的医院、院区和科室管理；
- 账号列表、详情、手机号精确搜索、停用和恢复；
- 医生开通、资料维护、调岗和撤销；
- 公共组织、科室和医生目录；
- `operation_id` 幂等、乐观锁、审计和事务 Outbox；
- App API → Identity RPC → Manager → MySQL 的真实 HTTP 全链路；
- 小程序真实 HTTP Adapter，运行时 Mock 和重叠旧写链路已经删除。

Identity 不保存预约、号源、医生排班、患者医疗档案、检查方案、地图或报告，也不远程裁决每个业务请求。
Appointment、Navigation、Report 等业务服务本地验证 Token 和授权版本，再结合自己拥有的资源和状态完成最终授权。

## 2. 账号与登录

当前主流程：

```text
手机号
  -> PNVS 发送并校验验证码
  -> phone_fingerprint 查找账号
  -> 不存在则创建 patient 账号
  -> 已禁用账号拒绝签发 Token
  -> 创建 Redis Refresh Session
  -> 签发 Hospital Access Token + Refresh Token
```

- 数据库使用稳定 UUID `account_id`；手机号不是业务主键；
- 手机号只保存 HMAC-SHA256 指纹、脱敏值和验证状态，不保存明文；
- 一个手机号指纹只能绑定一个账号；
- 发送验证码阶段维持统一响应，不在验证前暴露账号是否存在或被禁用；
- 验证成功后，禁用账号返回稳定 `ACCOUNT_DISABLED` 且不能获得 Token；
- 账号可维护重名的展示昵称，昵称不参与登录、不证明实名身份，也不能替代 `account_id`；
- OpenID 不能证明手机号、患者身份或医生资格，微信 code、OpenID 和 `session_key` 不进入业务日志。

## 3. 会话与授权版本

- Access Token 是短期 JWT，包含 `account_id`、账号类型、角色、permissions、当前科室、授权版本和有效期；
- 只有 Identity 持有私钥并签发 Token，其他服务只配置公钥；
- Refresh Token 每次使用都轮换，Redis 保存 Refresh Session 和创建时的授权版本；
- 退出撤销 Refresh Session，客户端立即清除本地 Token；
- 角色、当前科室和账号状态变化递增 `authorization_version`；昵称、医生展示资料和普通组织 CRUD 不递增
  个人授权版本；
- 登录和版本一致的正常刷新按 MySQL 最新 Principal 回填 Redis；
- Refresh Session 版本与 MySQL 不一致时拒绝刷新、撤销会话并要求重新登录；
- 账号禁用、Session 过期、撤销或 Token 重放同样拒绝刷新；
- Redis 版本缺失、读取失败或与 Token 不一致时，受保护请求 fail-closed 返回 `401`；
- 普通请求不从拦截器同步回源 Identity/MySQL，避免跨服务递归依赖。

授权变化在同一个 MySQL 事务中更新事实、审计和 Outbox。Publisher 在 Kafka ACK 后标记发布完成；Consumer
成功将更高版本写入 Redis 后才提交 Offset。发布、消费或提交失败可以重试，重复或乱序旧事件不能降低
Redis 版本。MySQL、Kafka 与 Redis 采用最终一致投影，不承诺数据库提交后的绝对下一次请求已经看到新版本。

## 4. 公共认证与授权边界

跨服务公共实现固定在：

```text
common/authn
  Principal、JWT、Context、HTTP/gRPC Interceptor、授权版本校验抽象

common/authn/versionredis
  Redis 授权版本 Reader/Writer

common/authz
  permission、角色和部门范围的通用判断
```

Identity 注入授权版本 Reader 和 Writer；App API、Appointment、Navigation、Report 等只注入
Reader。业务请求统一经过：

```text
JWT 本地验签
  -> Redis authorization_version 校验
  -> Principal 写入 Context
  -> Manager 判断 permission、部门范围、资源所有权和业务状态
```

App API HTTP Middleware 和下游 gRPC Interceptor 分别执行自己的安全边界，不能因为上游已经鉴权而跳过
下游校验。`common/authn` 不依赖 Identity Repository，也不允许其他服务获得 Token 签发权。

## 5. 角色、账号类型和工作人员开通

| 概念 | 当前定义 |
|---|---|
| `account_type` | `patient/staff/service` 技术主体类型 |
| `super_admin` | 组织、账号、授权及跨部门管理 |
| `department_doctor` | 当前所属科室内的业务工作人员 |
| patient | 没有工作人员角色，只操作本人患者业务 |

普通注册只能创建患者账号，用户不能自行选择医生或管理员身份。医生开通固定为：

```text
管理员按完整手机号精确查找账号
  -> 核对脱敏手机号和账号状态
  -> 线下确认人员与医生资格
  -> 按稳定 account_id 开通医生
  -> account_type = staff
  -> 创建或恢复医生档案并设置当前科室
  -> 授予 department_doctor
  -> management_version + 1
  -> authorization_version + 1
  -> audit + outbox
```

完整手机号只在请求内计算指纹，不提供模糊手机号搜索或账号枚举。昵称查询允许重名，仅用于返回分页候选；
所有写操作都以 `account_id` 为目标。

初始超级管理员由部署工具按服务器 Secret 中的手机号列表幂等初始化。手机号原文不得进入 Git；初始化后，
日常授权必须通过 Identity 业务操作，不能直接修改数据库。

## 6. 组织模型

首期组织树固定为：

```text
Hospital
  -> Campus
       -> Department
            -> Doctor profile
```

- 首期只有一个医院根节点，可以管理多个院区和科室；
- `campus.parent_id` 必须是医院，`department.parent_id` 必须是院区；
- 医生只有一个当前所属科室，医生档案只能引用有效科室；
- 院区存在启用科室时不能停用，科室存在启用医生时不能停用；
- 停用和恢复是状态变化，组织单元不物理删除；
- 已被业务引用的组织 ID 永久稳定；调岗和改名不修改预约、方案、报告等历史事实；
- 组织 `code` 由后端生成，用于稳定识别、导入和日志关联，普通编辑不能修改；
- 一名医生多科室任职不属于当前规则，不能提前建立未使用关系。

公共目录可以未登录访问，但只返回有效院区、科室、医生和展示资料，不返回手机号、角色明细、管理版本或
审计信息。

## 7. 账号和医生管理

账号、角色和医生档案相互独立：

- 账号决定能否登录和技术主体类型；
- 角色决定 permission 集合；
- 医生档案保存公开资料和当前科室；医生资源复用 `account_id`，不另设 `doctor_id`。

管理员已经可以分页查询账号、读取详情、按完整手机号精确搜索、停用和恢复账号。管理响应由服务端根据
最新状态计算 `available_actions`，客户端不能自行推断可执行写操作。

医生管理已经支持开通、编辑展示名/工号/头像/说明、调岗和撤销。撤销医生身份时保留账号、手机号绑定和
历史业务引用，移除当前有效医生档案和角色关系并提升管理与授权版本。账号停用后不能登录、刷新 Token
或出现在公共目录；恢复账号不会自动恢复已撤销的医生身份。

管理员安全边界固定为：App API 校验 → Identity RPC 再校验 → Logic 取得 Principal → Manager 校验
permission 和状态 → Store 持久化。账号/医生写入统一由 `IdentityAdminManager` 管理，组织写入统一由
`OrganizationManager` 管理，`AuthorizationManager` 只读取授权上下文，不得恢复旧的第二套授权写接口。

## 8. 幂等、并发与事务

所有管理员写请求携带：

- UUID `operation_id`：一次用户操作一个值，网络重试复用；
- `version` 或 `management_version`：客户端最近读取的资源版本。

处理顺序固定为：

```text
校验 operation_id
  -> 已存在：核对动作、目标和请求摘要并返回原结果
  -> 不存在：开始事务并锁定目标
  -> 校验 expected version
  -> 校验权限、层级和状态
  -> 更新主数据和版本
  -> 写审计与 Outbox
  -> 保存幂等结果
  -> 提交事务
```

同一 `operation_id` 携带不同动作、目标或请求摘要返回冲突。过期版本返回 `409`，客户端重新读取并由用户
确认，不能静默覆盖。主数据、授权版本、审计、Outbox 和幂等结果必须在同一 MySQL 事务内保持一致。

## 9. 数据、隐私、审计与日志

Identity 数据职责以 `migrations/identity/README.md` 为准，包括账号、手机号绑定、外部身份、工作人员档案、
组织、RBAC、Refresh Session、授权/组织审计和 Outbox。

- PNVS AccessKey、手机号 HMAC Key、JWT 私钥、微信 AppSecret 只从运行环境注入；
- 验证码、完整手机号、Token、OpenID、AccessKey 和请求 Secret 不进入普通日志；
- 手机号精确搜索链路屏蔽 HTTP/RPC 正文日志；
- 登录失败不暴露账号是否存在；
- 短信发送具有手机号/IP/日级限流和费用保护；
- 管理写、敏感账号详情和手机号搜索记录业务审计；
- 普通分页列表不逐行写审计；
- 审计记录只追加，包含操作者、目标、动作、operation/request ID、结果和必要脱敏摘要；
- 生产环境不存在按账号 ID 直接领取 Token 的开发后门。

## 10. 契约和实现事实源

- HTTP：`contracts/api/identity-*.api` 和 `docs/api/openapi.json`；
- RPC：`contracts/proto/identity/v1/identity.proto`；
- 事件：`contracts/events/identity-authorization-changed-v1.payload.schema.json`；
- 数据库：`migrations/identity/`；
- 服务实现：`service/identity/rpc/`；
- App API 聚合：`service/app/api/`；
- 本地和部署说明：`service/identity/README.md`、根 README 与 `deploy/`。

本文只保留业务规则和已实施边界，不复制会从契约生成的字段清单，也不作为启动手册。

## 11. 阶段验收结果

阶段收尾已经验证：

- Go 单元测试、Vet 和 MySQL 集成测试；
- 手机号登录、Refresh、退出、禁用账号和授权变化重新登录；
- 多个初始超级管理员幂等初始化；
- 组织 CRUD、层级停用保护和公共目录；
- 账号查询、手机号精确搜索、医生开通、资料编辑、调岗、撤销、账号停用和恢复；
- 重复 `operation_id`、请求摘要冲突和过期乐观锁版本；
- 授权 Outbox → Kafka → Redis 的重复、乱序和失败重试；
- 真实 HTTP → App API → Identity RPC → Manager → MySQL 全链路；
- 旧 DELETE/旧 RPC/重叠 Manager 回归保护；
- 小程序测试、TypeScript 检查、微信小程序构建和生产部署配置。

## 12. 后续边界

以下能力不继续堆入 Identity：

- 医生排班、检查项目、号源、容量和预约生命周期；
- Patient 医疗业务档案和家庭成员代理授权；
- Navigation 地点路线和 Report 报告；
- 多医院租户和医生多科室任职；
- 医生资质线上审批；
- 手机号换绑、号码回收争议、账号合并和注销审批；
- 医院统一身份平台同步。

Appointment 等新服务建立时直接复用公共认证和授权版本 Reader，并增加跨服务端到端回归；Identity 只在
出现阻塞新服务接入的缺陷时进入维护开发。
