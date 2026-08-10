# Identity 身份与访问控制模块

> 状态：手机号认证、Token 会话和首版角色授权已实现；管理页面待后续开发

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

账号可以拥有用户自行确认的可选昵称，用于界面展示和管理员备选搜索。昵称允许重名，不参与登录、
不作为实名依据，也不能替代 `account_id`。当前小程序昵称仍是本机数据，服务端同步和检索属于
[组织、科室、医生与用户管理方案](./04-organization-staff-and-seed-data.md)的待实现内容。

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
- `POST /api/v1/admin/identity/doctors/promote`。

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

- Access Token 是短期 JWT，各服务本地验签；
- Refresh Token 每次使用都会轮换，Redis 保存 Refresh Session；
- 并发刷新由前端合并为一次；
- 退出撤销 Refresh Session，前端立即删除本地 Token；
- 角色、部门或账号状态变化提升授权版本；统一拦截器发现旧 Access Token 的版本不一致时返回 `401`，
  客户端通过全局 `refreshOnce()` 无感轮换 Token，并使用最新 Principal 将原请求最多重试一次；
- 只有 Refresh Session 已过期、被撤销，或者账号已被禁用导致刷新失败时，客户端才清理本地会话并
  回到登录流程；
- `common/authn` 提供可注入的授权版本校验接口，HTTP Middleware 和 gRPC Interceptor 在 JWT 本地验签后，
  统一比较 Token 中的 `authorization_version` 与服务端当前版本；
- 授权版本的 Identity 实现优先读取 Redis，Redis 未命中时回源 Identity/MySQL 并回填；角色、部门或账号
  状态变更在 MySQL 事务中递增版本，提交后更新 Redis，并通过 Outbox 事件补偿失败或遗漏的缓存更新；
- 版本不一致时统一拒绝旧 Access Token，不允许每个业务 Manager 再分别查询操作者的角色和权限；
- Redis 校验器不可用且无法回源时，受保护写接口采用 fail-closed，不能因为缓存故障放行高风险操作；
- `common/authn` 只定义 Principal、Token、上下文传递、拦截器和版本校验抽象，不依赖 Identity Repository，
  也不承载账号、角色、组织等领域业务；
- 业务 Manager 接收拦截器已经验证的 `authn.Principal`，使用 `common/authz` 判断当前操作所需 permission，
  Repository 只查询和修改本业务资源；
- `authorization.Manager` 可以据此移除对“操作者最新权限”的重复查询，但仍负责目标账号前后状态、
  角色变更、幂等、审计、Outbox 和授权版本递增。

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
  store.go                     可被所有服务复用的 Redis VersionReader

service/identity
  authorization.Manager       角色、科室、账号状态变更与版本递增
  session.Manager             Refresh Session 与新 Token 签发
  repository                  Identity MySQL 事实与事务实现
```

`common/authn` 不导入 `service/identity/rpc/internal`；反过来由各服务的 `ServiceContext` 创建同一个
`versionredis.Store` 和 `AuthorizationVersionValidator`，再注入公共 HTTP/gRPC 拦截器。Appointment、
Planning、Report、Navigation 等服务只增加配置和依赖装配，不复制 Redis 查询、版本比较或错误映射代码。

授权版本遵守“Identity 单写、其他服务只读”：

- Identity 在登录、刷新、角色变更、调岗和账号状态变更后维护 Redis 当前版本；
- 其他服务只通过公共 `VersionReader` 读取同一 Redis 命名空间，不得自行修改授权版本；
- Redis 未命中时通过可注入 Loader 回源 Identity/MySQL 并回填，公共包不直接依赖 Identity RPC；
- Identity/MySQL 是授权事实源，Redis 是跨服务共享的快速版本投影；
- MySQL 事务提交后更新 Redis，Outbox 消费者负责补偿失败或遗漏的投影更新；
- 每个可直接接收受保护请求的服务都注册公共拦截器，不能只依赖前端隐藏菜单或上游服务口头保证。

请求阶段统一为：本地 JWT 验签 → 公共 VersionValidator 读取 Redis/必要时回源 → Principal 写入 Context
→ 业务 Manager 使用 `common/authz` 判断 permission。Identity 业务 Manager 不再查询操作者权限，但仍查询
并锁定目标账号、组织或医生的最新业务事实。

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

## 8. 当前未实现

- 工作人员业务页面和超级管理员身份录入页面；
- 医院、院区、科室以及工作人员管理，具体方案见
  [Identity 组织、科室、医生与用户管理](./04-organization-staff-and-seed-data.md)；
- 医生资质线上审批；
- 手机号换绑、号码回收争议和账号合并；
- 医院统一身份系统同步；
- 家庭成员授权关系。

这些能力进入实施前分别补充业务规则和契约，不在 Identity 中提前创建无用途字段。
