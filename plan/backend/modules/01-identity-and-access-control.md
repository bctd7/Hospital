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
       -> Redis Refresh Session
       -> Hospital Access Token + Refresh Token
```

手机号是登录标识，不是数据库主键。数据库使用 UUID `account_id`，手机号只保存 HMAC-SHA256
指纹、脱敏值和 `verified/sms` 状态。一个手机号指纹只能绑定一个账号。

微信登录接口和 `identity_external_identities` 作为兼容能力保留。OpenID 不能证明手机号、真实
姓名、患者身份或医生资格；微信 code、OpenID、`session_key` 不写业务日志。

## 3. 角色和工作人员开通

普通注册只能创建患者账号，用户不能在登录页自行选择医生或管理员身份。

```text
super_admin 按完整手机号精确查找账号
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

完整手机号只用于请求时计算指纹，不记录日志、不提供模糊搜索或账号枚举。未来前端的“身份录入”
页面是超级管理员专属能力，使用同一工作人员端框架并按 permission 显示。

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

部门被业务引用后不得物理删除；停用部门仍保留稳定 ID 和历史快照。未来多部门任职应增加明确
的医生—部门授权关系，不能把多个部门写进业务数据。

## 6. 会话与权限变更

- Access Token 是短期 JWT，各服务本地验签；
- Refresh Token 每次使用都会轮换，Redis 保存 Refresh Session；
- 并发刷新由前端合并为一次；
- 退出撤销 Refresh Session，前端立即删除本地 Token；
- 角色、部门或账号状态变化提升授权版本；新权限在重新登录或刷新后进入新 Token；
- 高风险禁用和撤权需要结合短 Access Token 有效期，不能假设旧 JWT 立即消失。

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
- 医生资质线上审批；
- 多部门任职和临时支援；
- 手机号换绑、号码回收争议和账号合并；
- 医院统一身份系统同步；
- 家庭成员授权关系。

这些能力进入实施前分别补充业务规则和契约，不在 Identity 中提前创建无用途字段。
