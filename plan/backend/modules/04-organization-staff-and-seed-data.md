# Identity 组织、科室、医生与用户管理

> 状态：首期实现完成。组织、账号、医生 CRUD 已完成真实 HTTP → RPC → Manager → MySQL 全链路验证；
> 运行时 Mock 和重叠的旧授权写链路已经删除。

本文件只保留仍然有效的领域规则与扩展边界。HTTP 字段以 `contracts/api/` 和
`docs/api/openapi.json` 为准，RPC 字段以 `contracts/proto/identity/v1/identity.proto` 为准。

## 1. 完成范围

### 1.1 组织

- 单医院下管理院区和科室；
- 查询医院/院区上下文、院区下科室和科室下医生；
- 管理员创建、列表、详情、修改、停用和恢复院区/科室；
- 院区存在启用科室时禁止停用；
- 科室存在启用医生时禁止停用；
- 所有组织删除语义均为状态变化，不物理删除。

### 1.2 账号

- 管理员分页查询账号和读取详情；
- 按完整手机号精确搜索，服务端只返回脱敏号码；
- 停用和恢复账号；
- 本人读取、更新展示昵称；
- 管理响应返回服务端计算的 `available_actions`。

### 1.3 医生

- 将已有患者账号开通为指定科室医生；
- 编辑医生展示名、工号、头像和说明；
- 调整当前所属科室；
- 撤销医生身份但保留账号及历史业务引用；
- 公共目录只返回启用账号、有效医生档案和有效科室中的医生。

### 1.4 工程能力

- Manager 定义权限、状态机和事务边界；
- MySQL Store 与 Transaction Store；
- `operation_id` 幂等与请求摘要冲突保护；
- 组织 `version` 和账号 `management_version` 乐观锁；
- 授权变化递增 `authorization_version`；
- 本地事务同时写主数据、审计和 Outbox；
- MySQL 集成测试、HTTP 全链路测试和旧路由回归测试；
- 小程序使用真实 HTTP Adapter，不保留运行时 Mock。

## 2. 统一术语

| 名称 | 含义 |
|---|---|
| `department` | 业务含义统一为“科室”；“部门管理”只保留为页面标题 |
| `organization_unit` | Identity 内部通用组织模型，只允许 `hospital/campus/department` |
| `account_id` | 账号和医生资源的稳定主键；医生不另设 `doctor_id` |
| `account_type` | 认证主体类型：`patient/staff/service` |
| `identity_type` | 管理端展示身份：`patient/doctor/super_admin`，由账号、角色和医生档案计算 |
| `version` | 组织单元乐观锁版本 |
| `management_version` | 账号/医生管理写操作乐观锁版本 |
| `authorization_version` | Token 权限快照版本 |
| `operation_id` | 写操作幂等键，不等同于任何版本号 |

`department_doctor` 仍是内部角色代码；对外资源和动作使用 `Doctor`、`Account`、`AdminAccount`，不再暴露
`ManagedDoctor`、`ManagedAccount` 等第二套作用域命名。

## 3. 组织模型

```text
Hospital
  -> Campus
       -> Department
            -> Doctor profile
```

首期只有一个医院根节点。父子规则：

- `campus.parent_id` 必须是医院；
- `department.parent_id` 必须是院区；
- 医生档案只能引用有效科室；
- 调整科室归属不修改历史预约、检查或报告；
- 已被业务引用的组织 ID 永久稳定。

组织 `code` 是后端生成的稳定业务编码，用于数据识别、导入和日志关联，不允许前端把名称当作编码，也不
允许管理员在普通编辑接口中修改编码。

## 4. 账号、角色和医生档案

账号、角色和医生档案是三个不同概念：

- 账号决定能否登录和技术主体类型；
- 角色决定权限集合，例如 `super_admin`、`department_doctor`；
- 医生档案保存公开展示资料和当前科室。

开通医生流程：

```text
按完整手机号精确找到账号
  -> 线下核验人员与资格
  -> 按 account_id 提交 promote-doctor
  -> account_type = staff
  -> 创建/恢复医生档案与当前科室
  -> 分配 department_doctor 角色
  -> management_version + 1
  -> authorization_version + 1
  -> audit + outbox
```

撤销医生身份时保留账号、手机号绑定和历史业务记录，移除当前有效医生档案/角色关系，并提升管理版本和
授权版本。禁止使用物理删除实现撤销。

账号停用后不能登录、刷新 Token 或出现在公共医生目录。恢复账号不会自动恢复已经撤销的医生身份。

## 5. 权限边界

公共目录允许未登录访问，只返回展示所需字段，不返回手机号、角色明细、审计信息和管理版本。

管理员接口必须同时满足：

1. App API Access Token Middleware 校验 JWT 与 Redis 授权版本；
2. Identity gRPC Interceptor 再次校验 JWT 与授权版本；
3. Logic 从 Context 取得 Principal；
4. Manager 校验所需 permission 和资源范围；
5. Store 只接收已经通过领域规则的持久化命令。

`AuthorizationManager` 只读取授权上下文。账号和医生管理写操作统一进入 `IdentityAdminManager`，组织写操作
统一进入 `OrganizationManager`。不得重新增加第二套授权写 API。

## 6. HTTP 契约

完整字段和示例见 [Swagger 文档](../../../docs/api/README.md)。主要路径：

| 能力 | 方法与路径 |
|---|---|
| 组织上下文 | `GET /api/v1/directory/organization-context` |
| 院区科室 | `GET /api/v1/directory/departments` |
| 科室医生 | `GET /api/v1/directory/departments/:departmentId/doctors` |
| 组织 CRUD | `/api/v1/admin/identity/organization-units` |
| 账号列表/详情 | `/api/v1/admin/identity/accounts` |
| 手机号精确搜索 | `POST /api/v1/admin/identity/accounts/search-by-phone` |
| 开通医生 | `POST /api/v1/admin/identity/accounts/:accountId/promote-doctor` |
| 医生资料/调岗 | `PUT /api/v1/admin/identity/doctors/:accountId[/department]` |
| 撤销医生 | `POST /api/v1/admin/identity/doctors/:accountId/revoke` |
| 账号停用/恢复 | `POST /api/v1/admin/identity/accounts/:accountId/disable|enable` |

状态变化统一使用动作型 POST：

```text
POST /:id/disable
POST /:id/enable
POST /:id/revoke
```

旧的 `DELETE` 状态变更路由和旧 RPC 方法已经删除，并有回归测试防止重新出现。

## 7. 幂等、并发和事务

每个管理员写请求携带：

- `operation_id`：客户端每次用户操作生成的新 UUID；网络重试复用同一个值；
- `version` 或 `management_version`：客户端上次读取到的资源版本。

固定处理顺序：

```text
校验 operation_id
  -> 已存在：核对动作、目标和请求摘要并返回原结果
  -> 不存在：开始事务并锁定目标
  -> 校验 expected version
  -> 校验状态、层级和权限
  -> 更新主数据与版本
  -> 写审计
  -> 写 Outbox
  -> 保存幂等结果
  -> 提交事务
  -> 发布 Redis authorization_version（如涉及授权）
```

同一个 `operation_id` 携带不同目标或请求内容必须返回冲突。旧版本写请求返回 `409`，前端重新读取详情后由
用户确认，不静默覆盖。

## 8. 隐私、审计和日志

- 数据库不保存明文手机号，只保存 HMAC 指纹和脱敏展示值；
- 手机号精确搜索的 App API Client 日志和 Identity RPC 统计日志都必须忽略正文；
- 验证码、完整手机号、Token、OpenID、AccessKey 不进入普通日志；
- 查询敏感账号详情、手机号搜索和所有管理员写操作记录业务审计；
- 普通分页列表不逐行写审计，避免噪声和审计表膨胀；
- 审计记录与主数据在同一 MySQL 事务中提交。

## 9. 测试与验收

阶段收尾已经验证：

- Go 单元测试与 Vet；
- MySQL 组织生命周期集成测试；
- MySQL 账号/医生生命周期集成测试；
- 登录、管理员初始化、组织 CRUD、账号查询、医生 CRUD 的真实 HTTP 全链路；
- 重复 `operation_id`、过期乐观锁版本、层级停用保护；
- 医生调岗和撤销后公共目录实时变化；
- 旧 DELETE 路由返回 `405`；
- 小程序测试、TypeScript 检查和微信小程序构建。

## 10. 后续能力

以下内容不属于本期 Identity CRUD：

- 多医院租户；
- 一名医生多科室任职；
- 医生排班、号源和预约容量；
- 资质线上审批；
- 账号合并、注销和手机号换绑争议处理；
- 医院统一身份平台同步。

排班、号源和预约能力进入独立 Appointment 服务，参见
[预约检查服务](./05-appointment-and-examination-booking.md)。这些能力不得继续堆入 Identity Manager。
