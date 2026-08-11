# Identity RPC Logic 导航

`logic` 保持扁平目录，是因为 goctl 默认按“一个 RPC 方法一个文件”生成 Server 和 Logic。这里不通过移动文件分包，避免重新生成 RPC 后出现重复文件或导入路径失效。

## 推荐查找方式

按业务查找时只看对应表格中的文件，不需要顺序浏览整个目录。

## 账号与登录

| RPC Logic | 文件 | 状态 |
| --- | --- | --- |
| 发送手机验证码 | `send_phone_login_code_logic.go` | 已实现 |
| 手机号登录 | `phone_login_logic.go` | 已实现 |
| 微信登录 | `we_chat_login_logic.go` | 已实现 |
| 绑定本人手机号 | `set_my_phone_logic.go` | 已实现 |
| 按手机号查询账号 | `find_account_by_phone_logic.go` | 已实现 |

公共错误转换在 `account_helpers.go`；登录成功后的 Token 响应转换目前在 `session_helpers.go`。

## 授权上下文

| RPC Logic | 文件 | 状态 |
| --- | --- | --- |
| 查询授权上下文 | `get_authorization_context_logic.go` | 已实现 |

授权模块只负责读取 `Principal`。医生开通、调岗、撤销和账号启停统一通过下方管理员 CRUD Logic
进入 `IdentityAdminManager`，不再由授权模块直接写账号数据。

`authorization_helpers.go` 包含四类公共能力：

- `authorizationRequestContext`：把合法的 `request_id` 放入请求上下文。
- `authorizationOperator`：取得当前登录用户，未登录时返回 `Unauthenticated`。
- `authorizationContextResponse`：把内部 `Principal` 转成 protobuf 响应。
- `authorizationRPCError`：把授权业务错误转成 gRPC 状态码。

## 会话

| RPC Logic | 文件 | 状态 |
| --- | --- | --- |
| 刷新访问令牌 | `refresh_access_token_logic.go` | 已实现 |
| 撤销刷新令牌 | `revoke_refresh_token_logic.go` | 已实现 |

公共响应和错误转换在 `session_helpers.go`。

## 本人资料、账号与医生管理

| RPC Logic | 文件 |
| --- | --- |
| 读取/更新本人昵称 | `get_account_display_profile_logic.go` / `update_account_display_profile_logic.go` |
| 管理员账号列表/详情 | `list_admin_accounts_logic.go` / `get_admin_account_logic.go` |
| 手机号精确搜索 | `search_admin_account_by_phone_logic.go` |
| 开通医生 | `promote_doctor_logic.go` |
| 更新医生资料 | `update_doctor_logic.go` |
| 调整医生科室 | `change_doctor_department_logic.go` |
| 撤销医生身份 | `revoke_doctor_logic.go` |
| 停用/恢复账号 | `disable_account_logic.go` / `enable_account_logic.go` |

这些 Logic 统一调用 `IdentityAdminManager`。公共转换和错误映射位于 `identity_admin_helpers.go`。

## 组织目录与管理员 CRUD

| RPC Logic | 使用者 | 文件 | 当前状态 |
| --- | --- | --- | --- |
| 获取医院和院区上下文 | 包括未登录访客在内的所有用户 | `get_organization_context_logic.go` | 已实现并有针对性测试 |
| 查询某院区下的科室 | 包括未登录访客在内的所有用户 | `list_departments_logic.go` | 已实现并有针对性测试 |
| 查询某科室下的医生 | 包括未登录访客在内的所有用户 | `list_doctors_by_department_logic.go` | 已实现并有数据库集成测试 |
| 管理端筛选组织节点 | 有组织管理权限的用户 | `list_organization_units_logic.go` | 已实现并有针对性测试 |
| 管理端查询单个节点 | 有组织管理权限的用户 | `get_organization_unit_logic.go` | 已实现并有针对性测试 |
| 创建院区或科室 | 有组织管理权限的用户 | `create_organization_unit_logic.go` | 已实现 |
| 修改院区或科室 | 有组织管理权限的用户 | `update_organization_unit_logic.go` | 已实现 |
| 禁用院区或科室 | 有组织管理权限的用户 | `disable_organization_unit_logic.go` | 已实现 |
| 启用院区或科室 | 有组织管理权限的用户 | `enable_organization_unit_logic.go` | 已实现 |

组织公共转换在 `organization_helpers.go`：

- `organizationRPCError`：组织业务错误到 gRPC 状态码。
- `adminOrganizationUnitResponse`：内部 `organization.Unit` 到管理员 protobuf 响应。
- 公共目录响应使用独立转换，避免把管理字段误暴露给公共接口。

## 组织模块调用顺序

管理接口的固定调用顺序：

```text
取得当前用户
  -> 权限或登录状态检查
  -> protobuf Request 转 Manager Command/Filter
  -> 调用 Manager
  -> 业务错误转 gRPC 错误
  -> 内部 Model 转 protobuf Response
  -> 针对性 Logic 测试
```

## 测试索引

| 文件 | 主要验证内容 |
| --- | --- |
| `authorization_logging_test.go` | 授权上下文读取失败的错误码和结构化日志 |
| `get_organization_unit_logic_test.go` | 查询成功、无权限、未登录三种情况 |
| `organization_read_logic_test.go` | 公共目录免登录、院区科室响应和管理列表筛选 |
| `organization_helpers_test.go` | 稳定 gRPC 错误原因映射 |

数据库集成测试不在 Logic 目录；它们位于 `repository/mysqlstore/`。
