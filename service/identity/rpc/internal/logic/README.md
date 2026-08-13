# Identity RPC Logic 导航

`logic` 目录保持扁平，因为 goctl 按“一个 RPC 方法一个文件”生成 Server 和 Logic。这里是协议适配层：解析请求、取得调用者、调用对应 Manager，并把业务错误与模型转换成 protobuf 响应。

业务规则、事务和幂等约束不能写进 Logic，也不能从 Logic 直接访问 Repository。

## 登录与会话

| 功能 | Logic | 业务入口 |
| --- | --- | --- |
| 发送手机验证码 | `send_phone_login_code_logic.go` | `Managers.PhoneLogin` |
| 手机号登录 | `phone_login_logic.go` | `Managers.PhoneLogin` |
| 微信登录 | `we_chat_login_logic.go` | `Managers.Authentication` |
| 绑定本人手机号 | `set_my_phone_logic.go` | `Managers.Authentication` |
| 刷新访问令牌 | `refresh_access_token_logic.go` | `Managers.Session` |
| 撤销刷新令牌 | `revoke_refresh_token_logic.go` | `Managers.Session` |

微信、阿里云手机号和本地测试实现位于 `authentication/provider/`。Provider 只适配外部登录渠道；账号解析由 `authentication` 完成，Token 与 Refresh Session 生命周期由 `session` 完成。

公共登录错误转换位于 `account_helpers.go`，Token 响应转换位于 `session_helpers.go`。

## 授权上下文

`get_authorization_context_logic.go` 调用只读的 `Managers.Authorization`，返回当前账号的 Principal。账号状态、医生身份、科室和权限变更不在授权读取模块写入，而由账号管理模块完成。

`authorization_helpers.go` 负责：

- 从请求上下文取得调用者；
- 传递合法的 `request_id`；
- 将内部 Principal 转为 protobuf；
- 将授权读取错误转为稳定的 gRPC 状态码。

## 账号、医生与本人资料

| 功能 | Logic |
| --- | --- |
| 读取/更新本人昵称 | `get_account_display_profile_logic.go` / `update_account_display_profile_logic.go` |
| 管理账号列表、详情和手机号搜索 | `list_admin_accounts_logic.go` / `get_admin_account_logic.go` / `search_admin_account_by_phone_logic.go` |
| 开通、更新、调科和撤销医生 | `promote_doctor_logic.go` / `update_doctor_logic.go` / `change_doctor_department_logic.go` / `revoke_doctor_logic.go` |
| 停用/恢复账号 | `disable_account_logic.go` / `enable_account_logic.go` |

这些 Logic 统一调用 `Managers.Account`，公共模型转换与错误映射位于 `account_management_helpers.go`。

## 组织目录与管理

公共目录读取调用 `Managers.OrganizationDirectory`，组织管理 CRUD 调用 `Managers.OrganizationUnit`：

| 功能 | Logic |
| --- | --- |
| 医院、院区上下文 | `get_organization_context_logic.go` |
| 院区科室、科室医生目录 | `list_departments_logic.go` / `list_doctors_by_department_logic.go` |
| 管理端组织节点查询 | `list_organization_units_logic.go` / `get_organization_unit_logic.go` |
| 创建、修改、停用、恢复组织节点 | `create_organization_unit_logic.go` / `update_organization_unit_logic.go` / `disable_organization_unit_logic.go` / `enable_organization_unit_logic.go` |

组织模型转换与错误映射位于 `organization_helpers.go`。

## 固定调用顺序

```text
protobuf Request
  -> 调用者/权限检查
  -> Manager Command 或 Filter
  -> Manager
  -> 业务错误映射
  -> protobuf Response
```

数据库集成测试位于 `repository/mysqlstore/`；Logic 目录只保留针对协议适配和错误映射的测试。
