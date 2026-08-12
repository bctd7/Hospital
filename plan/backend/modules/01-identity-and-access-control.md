# Identity 身份与访问控制模块

> 状态：手机号认证、JWT、Refresh Session、首版角色授权、Redis 授权版本校验、权限变化后强制重新登录，
> 管理员账号/医生管理，以及 Outbox → Kafka → Redis 授权版本自动补偿链路均已实现。

## 1. 模块职责

Identity 保存身份事实并签发 Token：

- Hospital 账号与账号状态；
- PNVS 已验证手机号和兼容的微信外部身份；
- `super_admin`、`department_doctor` 角色；
- 工作人员当前部门；
- Access Token、Refresh Session 和授权版本；
- 角色、部门、账号状态变更的审计与 Outbox 事件。

Identity 不远程裁决每一条预约、方案或报告。业务服务本地验签，读取 Token Principal，再结合
资源所属部门和业务状态完成最终授权。

## 2. 登录与账号创建

当前主流程：

```text
手机号 -> PNVS 发送/校验验证码
       -> phone_fingerprint 查找账号
       -> 不存在则创建 patient 账号
       -> 已禁用账号拒绝签发 Token
       -> Redis Refresh Session
       -> Hospital Access Token + Refresh Token
```

手机号是登录标识，不是数据库主键。数据库使用 UUID `account_id`，手机号只保存 HMAC-SHA256
指纹、脱敏值和 `verified/sms` 状态。一个手机号指纹只能绑定一个账号。

验证码发送和账号状态分开：发送接口维持统一响应，避免在验证前暴露账号状态；验证码校验成功后，
禁用账号必须返回稳定 `ACCOUNT_DISABLED` 错误且不能签发 Token。禁止特定号码收码属于独立短信风控，
不由普通账号禁用状态隐式承担。

账号可以拥有用户自行确认的可选昵称，用于界面展示和管理员候选查询。昵称允许重名，不参与登录、
不作为实名依据，也不能替代 `account_id`。昵称已经由 Identity 保存，小程序本地 Storage 只作为界面缓存。

微信登录接口和 `identity_external_identities` 作为兼容能力保留。OpenID 不能证明手机号、真实
姓名、患者身份或医生资格；微信 code、OpenID、`session_key` 不写业务日志。

## 3. 角色和工作人员开通

普通注册只能创建患者账号，用户不能在登录页自行选择医生或管理员身份。

```text
super_admin 按完整手机号精确查找，或按昵称查询候选账号
  -> 核对脱敏手机号、身份、状态和科室
  -> 选中稳定 account_id
  -> 线下确认人员和医生资格
  -> promote doctor(account_id, department_id, offline_verified)
  -> account_type = staff
  -> staff profile + department
  -> role = department_doctor
  -> authorization_version + 1
```

相关接口：

- `POST /api/v1/admin/identity/accounts/search-by-phone`；
- `POST /api/v1/admin/identity/accounts/:accountId/promote-doctor`。

完整手机号只用于请求时计算指纹，不记录日志、不提供模糊手机号搜索或账号枚举。昵称查询允许重名，
只负责返回分页候选。开通医生、调岗、撤销身份、禁用和恢复账号都在用户详情按 `account_id` 执行，
不能直接以手机号或昵称作为写接口目标。

首位超级管理员通过部署阶段工具初始化：

```powershell
go run ./tools/identity-bootstrap-admin --account-id <account-id>
```

日常授权不得绕过 Identity 业务操作直接修改数据库。

## 4. 权限模型

首期角色：

| 角色 | 主要范围 |
|---|---|
| `super_admin` | 组织、账号、授权以及跨部门管理 |
| `department_doctor` | 本部门业务操作；跨部门能力按具体 permission 和数据政策控制 |
| patient（无工作人员角色） | 本人患者业务 |

Access Token 包含：`account_id`、`account_type`、角色、部门、permissions、授权版本和短期有效期。
各业务服务使用 `common/authn` 验证 Token，使用 `common/authz` 复用部门范围规则，并自行判断资源
归属和业务状态。

权限不能绕过业务状态机：已发布报告、已执行检查、已确认预约和已生效规则必须遵守各自版本与
状态约束。

## 5. 部门和数据归属

医生首期只有一个当前部门。调岗只改变当前工作人员档案，不迁移预约、方案、报告等历史业务
数据。业务数据归属于产生业务的部门，医生 ID 只用于创建者、处理者和审计。

部门被业务引用后不得物理删除；停用部门仍保留稳定 ID 和历史快照。当前业务规则明确限制一个医生
只有一个所属科室；如果未来规则发生变化，必须重新设计和评审，不能提前写入多科室关系。

## 6. 会话与权限变更

- Access Token 是短期 JWT，只有 Identity 持有私钥并签发；其他服务只持有公钥并使用
  `common/authn` 本地验签；
- Refresh Token 每次使用都会轮换，Redis 保存 Refresh Session；
- 并发刷新由前端合并为一次；
- 退出撤销 Refresh Session，前端立即删除本地 Token；
- 角色、当前科室归属或账号状态变化提升账号 `authorization_version`；昵称、医生展示资料和医院、院区、
  科室组织 CRUD 不提升个人授权版本；
- 登录时以 MySQL 最新 Principal 签发 Access Token、创建携带相同授权版本的 Refresh Session，并写入
  Redis 账号当前授权版本投影；
- `common/authn` 提供可注入的授权版本校验接口，HTTP Middleware 和 gRPC Interceptor 在 JWT 本地验签后，
  统一比较 Token 中的 `authorization_version` 与服务端当前版本；
- Redis 版本缺失、读取失败或与 Token 不一致时，所有受保护接口采用 fail-closed 并返回 `401`；普通请求
  不从拦截器回源 Identity/MySQL，避免跨服务递归依赖；
- 客户端收到 `401` 后只调用一次全局 `refreshOnce()`；Refresh RPC 不依赖旧 Access Token，因此可以直接
  查询 Refresh Session 和 MySQL 最新 Principal；
- Refresh Session 版本与 MySQL 当前版本一致时，允许无感轮换 Token、回填 Redis 版本并将原请求最多重试
  一次；两者不一致表示个人授权已经变化，必须拒绝刷新、撤销当前 Refresh Session、清理客户端会话并
  重新登录；
- 账号已禁用、Refresh Session 已过期、被撤销或发生 Token 重放时同样拒绝刷新；账号禁用时重新登录也
  必须失败；
- 版本一致时不允许每个业务 Manager 再分别查询操作者的角色和权限；Manager 使用已经验证的 Principal
  判断 permission，只在事务中查询并锁定目标账号或本业务资源；
- `common/authn` 只定义 Principal、Token、上下文传递、拦截器和版本校验抽象，不依赖 Identity Repository，
  也不承载账号、角色、组织等领域业务；
- 角色、当前科室或账号状态变更在 MySQL 事务中递增授权版本并写入 Outbox；Publisher 轮询未发布事件并
  在 Kafka ACK 后标记 `published_at`，Consumer 将新版本单调、幂等地写入 Redis，成功后才提交 Offset；
  发布、投影或提交 Offset 失败均会重试，旧事件不能覆盖更高版本。MySQL、Kafka 与 Redis 不是同一事务，
  因此这是最终一致投影，不能承诺数据库提交后的绝对下一次请求必然已经看到新版本。

### 6.1 多服务复用边界

统一拦截器是“所有服务注册同一份公共实现”，不是每个服务各写一套权限查询。代码边界固定为：

```text
common/authn
  principal.go                 Principal 与 Context 传递
  token.go                     JWT 签发与本地验签
  authorization_version.go     VersionReader/Validator 抽象与版本比较
  http.go                      统一 HTTP Middleware
  grpc.go                      统一 gRPC Interceptor

common/authn/versionredis
  store.go                     可复用 Redis Store，实现 Reader 与 Writer 能力

service/identity
  authorization.Manager       只读取授权上下文
  identityadmin.Manager       账号、医生、角色和账号状态管理
  organization.Manager        组织管理与公共目录
  session.Manager             Refresh Session 版本比较与新 Token 签发
  outbox.Publisher            发布 MySQL 待处理事件并记录发布结果
  authorizationprojection     消费 Kafka 并单调推进 Redis 版本
  messaging                   Kafka Producer/Consumer 适配
  repository                  Identity MySQL 事实与事务实现
```

`common/authn` 不导入 `service/identity/rpc/internal`；反过来由各服务的 `ServiceContext` 创建同一个
`versionredis.Store` 和 `AuthorizationVersionValidator`，再注入公共 HTTP/gRPC 拦截器。Appointment、
Planning、Report、Navigation 等服务只增加配置和依赖装配，不复制 Redis 查询、版本比较或错误映射代码。

Access Token 能力统一放在 `common/authn`，但签发权不共享：JWT Claims、签名算法、验签、Principal、
Context 和拦截器是公共代码；JWT 私钥、Access Token 签发、Refresh Token 和 Refresh Session 只属于
Identity。其他服务只配置公钥，不能签发 Hospital Access Token。

授权版本遵守“Identity 单写、其他服务只读”：

- `common/authn` 定义 `AuthorizationVersionReader` 和 Validator；`versionredis.Store` 可以同时实现 Reader
  和 Writer 接口；
- app-api、Appointment、Planning、Report、Navigation 等服务只注入 Reader；Identity 注入 Reader 和
  Writer，并且是唯一允许写入授权版本的服务；
- Identity 在登录、正常刷新、角色变更、调岗和账号状态变更后维护 Redis 当前版本；
- Redis 未命中不由普通拦截器回源；拦截器返回 `401`，再由不依赖旧 Access Token 的 Refresh 或登录流程
  查询 MySQL 并回填；
- Identity/MySQL 是授权事实源，Redis 是跨服务共享的快速版本投影；
- 管理写事务不直接写 Redis；Outbox 保存版本事实并通过 Kafka 投影。登录和版本一致的正常刷新可以按
  MySQL Principal 回填 Redis，用于初始化或修复缺失投影；
- 每个可直接接收受保护请求的服务都注册公共拦截器，不能只依赖前端隐藏菜单或上游服务口头保证。

请求阶段统一为：本地 JWT 验签 → 公共 VersionValidator 读取 Redis → Principal 写入 Context → 业务
Manager 使用 `common/authz` 判断 permission。Identity 业务 Manager 不再查询操作者权限，但仍查询并锁定
目标账号、组织或医生的最新业务事实。app-api 和下游 RPC 各自在自己的安全边界执行同一公共校验，即使
一次用户请求读取两次 Redis，也不通过信任上游口头保证来绕过下游服务鉴权。

### 6.2 当前实现与待实现

当前已经实现：

- Access Token Claims 已包含角色、permissions、当前科室和 `authorization_version`；
- `common/authn` 已提供 `AuthorizationVersionReader`、Validator 和 HTTP/gRPC 统一校验，
  `common/authn/versionredis` 已实现统一账号级 Key：`identity:authorization-version:{account_id}`；
- app-api HTTP Middleware 与 Identity gRPC Interceptor 已完成 JWT 本地验签、Redis 版本校验和 Principal
  Context 注入，两个服务的 `ServiceContext` 已分别装配 Reader 与 Identity Writer；
- Refresh Session 已保存创建或轮换时的 `authorization_version`；
- 登录和正常刷新会回填 Redis 当前版本；Refresh 会重新查询 MySQL 最新 Principal，Session 版本不一致时
  拒绝轮换并撤销会话；前端已有并发合并的 `refreshOnce()`、最多一次请求重试和失败后清理会话；
- `AuthorizationManager` 已收缩为授权上下文读取；管理员账号和医生写操作统一进入
  `IdentityAdminManager`，组织写操作统一进入 `OrganizationManager`；Manager 接收拦截器验证过的
  Principal，不重复查询操作者权限；
- 授权变化与审计、幂等结果在同一 MySQL 事务写入 Outbox；Publisher 每秒轮询未发布事件，使用
  `account_id` 作为 Kafka Key；Consumer 投影 Redis 成功后才提交 Offset，Redis Lua 脚本只接受更高版本；
- 发布失败会保留 `published_at IS NULL` 并累计 `attempts`，Kafka/Redis/Offset 提交失败均有重试；重复发布、
  重复消费和乱序旧事件不会降低 Redis 中的授权版本。

下一阶段待实现：

- 将真实 MySQL、Kafka、Redis 的授权投影故障恢复、进程重启和并发压力场景纳入可重复执行的自动化回归；
- Appointment、Planning、Report、Navigation 等服务创建并直接接收受保护请求时，复用相同 Reader 和
  gRPC Interceptor 装配；
- Appointment、Planning、Report、Navigation 等新服务建立时的跨服务授权版本端到端回归。

## 7. 数据与安全边界

主要数据职责以 `migrations/identity/README.md` 为准，包括账号、手机号、外部身份、工作人员档案、
角色权限、Refresh Session、审计与 Outbox。

必须遵守：

- PNVS AccessKey、手机号 HMAC Key、JWT 私钥、微信 AppSecret 仅从运行环境注入；
- 验证码、完整手机号、OpenID、Token 和 Secret 不写普通日志；
- 登录失败不暴露账号是否存在；
- 短信发送具有手机号/IP/日级限流和费用防护；
- 权限管理操作记录操作者、目标账号、operation ID、结果和时间；
- 生产环境不存在按账号 ID 直接领取 Token 的开发后门。

## 8. 后续能力

- 医生资质线上审批；
- 手机号换绑、号码回收争议和账号合并；
- 医院统一身份系统同步；
- 家庭成员授权关系。

这些能力进入实施前分别补充业务规则和契约，不在 Identity 中提前创建无用途字段。
