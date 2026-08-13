# Identity 数据库

`hospital_identity` 是 Identity 的业务事实库。当前尚未发布且没有需要保留的正式业务数据，迁移已压平为单一最新初始版本 `000001`，共包含 13 张表；账号、登录、
组织、RBAC、审计和 Outbox 分开保存，但只有 `identity_accounts` 是账号主表。

## 当前表

| 分组 | 表 | 职责 |
|---|---|---|
| 账号 | `identity_accounts` | 账号类型、状态、授权版本和管理版本 |
| 账号 | `identity_account_profiles` | 用户本人维护的可选展示昵称 |
| 登录 | `identity_account_phones` | 手机号 HMAC 指纹、脱敏值和验证状态 |
| 登录 | `identity_external_identities` | 微信等兼容外部身份映射 |
| 组织 | `identity_organization_units` | 医院、院区、科室与组织乐观锁版本 |
| 医生 | `identity_staff_profiles` | 当前科室、工号、公开资料和医生状态 |
| RBAC | `identity_roles` | 角色定义 |
| RBAC | `identity_permissions` | permission 定义 |
| RBAC | `identity_account_roles` | 账号角色关系 |
| RBAC | `identity_role_permissions` | 角色 permission 关系 |
| 审计 | `identity_authorization_audit` | 账号、医生和授权变更审计及幂等结果 |
| 审计 | `identity_organization_audit` | 组织变更审计及幂等结果 |
| 事件 | `identity_outbox_events` | 与业务事务一起写入的待发布事件 |

```text
identity_accounts
├── identity_account_profiles
├── identity_account_phones
├── identity_external_identities
├── identity_staff_profiles -> identity_organization_units(department)
└── identity_account_roles -> identity_roles -> identity_role_permissions -> identity_permissions

identity_organization_units(hospital -> campus -> department)
identity_authorization_audit
identity_organization_audit
identity_outbox_events
```

账号不等于患者就诊人。手机号验证码只证明号码控制权，不证明医疗实名；姓名、身份证、医保、预约和报告
不得进入 Identity 账号表。

## 当前初始迁移

`000001_identity_initial_schema` 直接创建当前代码需要的最终结构，包括统一组织单元、账号管理版本、展示昵称、完整医生资料、手机号与外部身份、RBAC、审计和 Outbox，不再创建旧 `identity_departments` 后再执行重命名和字段回填。

## 修改要求

- 新迁移同步提供 down 文件；
- Repository、Manager 集成测试和本 README 同步更新；
- 外键只能表达引用存在性，`department_id` 必须指向 department 等类型约束由 Manager 校验；
- `operation_id` 唯一约束和资源版本字段不得绕开；
- 审计、Outbox 和主数据在同一 MySQL 事务中写入；
- 空库必须完成 `up`，测试环境验证需要时执行 `down → up`。
- 正式发布后不得修改已执行迁移，必须新增版本。
