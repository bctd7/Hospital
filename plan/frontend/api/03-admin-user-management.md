# 超级管理员用户接口

> 状态：待补充后端契约并联调

## 1. 调用边界

本组接口只服务[超级管理员用户管理页面](../pages/05-admin-user-management.md)，统一位于
`/api/v1/admin/identity`。所有请求要求 Hospital Access Token，`app-api` 和 Identity Service
分别完成 permission 校验；隐藏前端入口不能替代后端鉴权。

当前已经存在：

```http
POST /api/v1/admin/identity/accounts/search-by-phone
POST /api/v1/admin/identity/doctors/promote
```

用户分页、详情、调岗、撤销医生身份和账号禁用/恢复仍是待新增契约。

## 2. 用户查询

```http
GET /api/v1/admin/identity/accounts?page=1&page_size=20&nickname=&identity_type=&status=&department_id=
GET /api/v1/admin/identity/accounts/:accountId
POST /api/v1/admin/identity/accounts/search-by-phone
```

分页列表建议返回：

```text
AdminAccountSummary
  account_id
  nickname?
  display_name?
  avatar_url?
  masked_phone?
  account_status
  identity_type = patient / doctor / super_admin
  department_id?
  department_name?
  management_version
```

`identity_type` 由账号、角色和有效医生档案动态计算，不新增重复保存字段。`nickname` 是用户自行
维护的可选账号展示资料，`display_name` 是医生档案名称；两者都可能为空，前端使用脱敏手机号兜底。

详情在列表字段之上返回手机号验证状态、账号创建/更新时间、角色、可选工号、医生公开资料、医生
状态、`authorization_version` 和服务端计算的 `available_actions`，但不返回 OpenID、Token、
验证码、完整审计记录或内部密钥。完整手机号仅作为管理员主动输入的精确查询条件；当前数据库没有
可恢复的手机号明文，分页和详情只返回脱敏号码。

后端必须分页并限制最大 `page_size`。数据库应针对账号状态、工作人员身份和部门筛选建立必要索引，
不能先取出全部账号再在内存中过滤。

搜索栏采用两条明确链路：

```text
完整手机号
  -> POST /accounts/search-by-phone
  -> 规范化号码并计算 HMAC 指纹
  -> 精确返回一个账号或未找到

昵称或医生显示名称
  -> GET /accounts?nickname=<text>
  -> 服务端前缀匹配、分页返回候选
  -> 允许重名，管理员必须进入详情再次确认
```

当前本机 Storage 中的昵称不能被后端搜索。昵称查询落地前，需要新增账号展示资料同步能力，由用户在
就诊人管理页面确认后写入 Identity；昵称不唯一、不作为登录凭据，也不能证明真实姓名。

## 3. 身份和账号操作

以下所有写接口都接收稳定 `account_id`，不接收手机号或昵称作为目标。手机号和昵称只负责在管理员
搜索栏定位候选账号；这样即使手机号将来换绑或昵称修改，也不会操作错误的数据库记录。

### 3.1 开通医生

```http
POST /api/v1/admin/identity/doctors/promote
```

请求在现有账号 ID 之外补充工作人员显示名称、可选工号、唯一 `department_id`、头像 URL、擅长
描述、`operation_id` 和账号 `management_version`。后端原子创建医生资料、授予角色并提升管理版本
和授权版本。

曾经被撤销医生身份的普通账号可以再次开通。后端必须重新激活原有 `revoked` 医生档案并更新科室和
公开资料，不能插入第二条档案；已禁用账号必须先恢复，开通医生操作不能顺便恢复登录能力。

### 3.2 编辑资料与调岗

```http
PATCH /api/v1/admin/identity/doctors/:accountId
PUT   /api/v1/admin/identity/doctors/:accountId/department
```

- `PATCH` 维护工作人员显示名称、工号和公开医生资料；
- `PUT .../department` 原子替换首版唯一所属科室；
- 调岗成功后响应包含原科室和目标科室 ID，供前端准确失效缓存；
- 不允许通过清空 `department_id` 产生无科室医生。

### 3.3 撤销医生身份

```http
DELETE /api/v1/admin/identity/doctors/:accountId
```

撤销医生资料和医生角色，保留账号、普通用户能力和历史业务记录。后端提升授权版本、撤销 Refresh
Session、记录审计与 Outbox；前端刷新详情及原科室目录。

### 3.4 禁用和恢复账号

```http
POST /api/v1/admin/identity/accounts/:accountId/disable
POST /api/v1/admin/identity/accounts/:accountId/enable
```

- 禁用要求 `identity.account.manage`，使账号所有登录入口不可用并撤销全部 Refresh Session；
- 管理员通常先通过完整手机号精确查询账号，核对详情后再按 `account_id` 调用禁用接口；
- 禁用医生账号保留医生角色和档案；恢复后仍按原医生身份使用；
- 已经先行撤销医生身份的账号，恢复登录能力时不会自动重新开通医生；
- 首版拒绝通过本接口禁用任何 `super_admin`，避免自锁和最后管理员问题；
- 禁用和恢复不物理删除账号及历史业务记录。

## 4. 权限矩阵

| 接口能力 | permission |
|---|---|
| 用户列表、详情及精确手机号查询 | `identity.authorization.manage` 或 `identity.account.manage` |
| 开通医生、编辑医生、调岗、撤销医生身份 | `identity.authorization.manage` |
| 禁用和恢复账号 | `identity.account.manage` |

账号详情的 `available_actions` 由后端根据目标状态和操作者最新权限计算，可包含
`promote_doctor`、`edit_doctor`、`change_department`、`revoke_doctor`、`disable_account` 和
`enable_account`。前端取交集显示按钮，不能自行推导服务端没有返回的高风险操作。

## 5. 幂等、并发与错误

- 所有写请求携带 UUID `operation_id`；重复提交返回原操作结果；
- 请求携带详情 `management_version` 做乐观锁，冲突时返回 `409` 并要求重新读取；
- 只修改医生头像或擅长描述不会让 Access Token 失效；角色、科室或账号状态变化才提升授权版本；
- 目标账号已调岗、已撤销、已禁用或已恢复时返回稳定业务码；
- `401` 进入统一 Token 刷新，写请求只在明确可安全重试时重放；
- `403` 刷新 Principal 并退出用户管理页；
- 完整手机号、Token、验证码和敏感资料不能进入普通日志或 Toast；
- 查询敏感用户详情、开通、调岗、撤销、禁用和恢复都写审计记录；普通分页列表不逐行写业务审计。

## 6. 缓存与联调

- 用户列表按筛选条件和页码短期缓存，进入详情始终校验版本；
- 写入成功后失效用户详情、相关列表页和受影响科室医生目录；
- 页面返回列表时只在收到失效标记或缓存过期后重新请求；
- 医生和普通用户调用任一接口均返回 `403`；
- 超级管理员缺少相应 permission 时不能通过手工构造请求扩大能力；
- 列表、详情和操作结果全部来自后端动态数据，不使用前端静态用户清单；
- 并发管理员修改、重复点击和网络重试不会产生重复角色或半完成调岗。
