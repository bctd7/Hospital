# Identity 数据库表说明

这里是 `hospital_identity` 数据库结构的事实来源。当前共有 11 张表，但只有
`identity_accounts` 是账号主表；其余表分别保存登录凭证映射、工作人员资料、组织、
RBAC 关系、审计和可靠事件，不是 11 张用户表。

## 表的职责

| 分组 | 表 | 当前职责与约束 |
|---|---|---|
| 账号 | `identity_accounts` | 平台账号主表；保存账号类型、状态和授权版本 |
| 登录 | `identity_external_identities` | 可选外部身份与平台账号的映射；当前保留微信兼容能力，不保存 AppSecret、登录 code 或 session_key |
| 登录 | `identity_account_phones` | 当前主登录标识；保存手机号 HMAC 唯一指纹、脱敏值和验证状态，`account_id` 为主键 |
| 组织 | `identity_departments` | 科室主数据和上下级关系 |
| 组织 | `identity_staff_profiles` | 工作人员专属资料；`account_id` 为主键，一个工作人员首期只有一个当前科室 |
| RBAC | `identity_roles` | 角色定义，例如超级管理员、部门医生 |
| RBAC | `identity_permissions` | 权限定义，例如创建预约、发布报告 |
| RBAC | `identity_account_roles` | 账号当前角色；`account_id` 为主键，一个账号首期最多一个角色 |
| RBAC | `identity_role_permissions` | 角色与权限的多对多关系 |
| 审计 | `identity_authorization_audit` | 授权和组织变更的不可变业务审计记录 |
| 事件 | `identity_outbox_events` | 与授权事务一起写入、等待发布的事件；不是用户属性，也不是 Kafka 本身 |

## 当前关系

```text
identity_accounts（平台账号）
├── identity_external_identities（可选微信兼容身份）
├── identity_account_phones（唯一手机号主登录标识，0..1）
├── identity_staff_profiles（工作人员资料，0..1）
└── identity_account_roles（当前角色，0..1）
      └── identity_roles
            └── identity_role_permissions
                  └── identity_permissions

identity_staff_profiles -> identity_departments
identity_authorization_audit -> 操作人账号 + 目标账号
identity_outbox_events -> 待发布的授权变更事件
```

`identity_accounts` 表示“系统认识哪个账号”，不等于患者业务中的“就诊人”。手机号验证码
证明号码控制权，但不证明医疗实名；姓名、身份证、医保资料、预约和报告不能放进 Identity 账号表。

## 为什么当前保留拆表

- 微信 OpenID 是可选外部供应商标识，不作为平台账号主键。
- 手机号是当前唯一登录标识，但号码会更换、回收且需要脱敏，因此仍使用不可变 UUID 作为账号
  主键，并把手机号验证状态独立保存。
- 工作人员资料只属于 staff 账号；患者账号不需要科室和工号。医生资格等更复杂资料未来也不
  应继续堆入账号主表。
- 角色与权限需要被多个账号复用。当前一个账号只有一个角色，但角色拥有多个权限，权限也可
  被多个角色复用，因此保留标准 RBAC 关系。
- 审计记录和 Outbox 事件都是一对多历史数据，不能作为用户字段保存。

## 防止继续过度拆分

新增表不能只因为“这个字段概念不同”。满足以下至少一个条件时才考虑拆表：

1. 与主体是一对多或多对多关系；
2. 有独立生命周期、状态机或保留周期；
3. 有明确不同的安全、访问控制或合规要求；
4. 属于可独立演进的业务边界，而不是单纯为了预留未来。

反之，确定的一对一简单属性优先留在所属实体中。未来如果产品永久确认只有一个微信身份、
一个手机号、一个角色，且这些数据没有独立生命周期，可以通过正式迁移合并；在需求确认前
不要为了减少表数量直接删除边界。

## 迁移文件

- `000001_identity_authorization.up.sql`：账号、组织、RBAC、审计和 Outbox，共 9 张表；
- `000002_identity_login.up.sql`：外部登录身份和手机号，共 2 张表。

修改结构时必须同步更新回滚脚本、Repository、集成测试和本文档。已经执行过的生产迁移不得
原地修改；当前项目尚未发布，基线迁移修改也必须重新验证空库初始化。
