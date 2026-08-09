# 微信注册与医生身份开通方案（历史登录方案）

> 当前登录基线已调整为阿里云 PNVS 手机号短信认证，见
> `miniapp/07-phone-primary-authentication.md`。微信登录暂时只作为兼容接口；医生开通、权限审批和
> 医疗数据访问边界继续沿用本文规则。

> 文档状态：首期方案已确认，可进入实现
>
> 关联文档：`03-identity-and-access-control.md`、`../docs/identity-authentication.md`

## 1. 目标与边界

首期使用个人主体微信小程序能够使用的基础登录能力：小程序调用 `wx.login` 获取一次性 code，Identity Service 调用微信 `code2Session` 换取当前小程序下的 OpenID。OpenID 只用于证明“这是同一个微信用户”，不用于证明其真实姓名、手机号或医生资格。

普通用户首次微信登录时自动注册为患者账号，不需要超级管理员录入。医生和超级管理员属于平台内部身份，不能由用户自行选择：

- 新 OpenID 自动创建 `patient` 账号；
- 用户可以在登录后自行登记手机号；
- 超级管理员使用完整手机号精确查找账号，在线下确认人员身份和手机号归属后，将账号设置为指定部门的医生；
- 第一位超级管理员由部署初始化命令建立，后续管理员授权沿用 Identity 的受控授权流程。

个人主体暂不依赖微信手机号快捷验证组件。用户自行输入的号码默认是 `self_reported`，不是已验证号码；本期医生开通必须由超级管理员明确确认已经完成线下核验。

## 2. 身份模型

```text
微信 OpenID（登录凭证）
  -> Hospital Account（平台账号）
      -> account_type = patient / staff
      -> role = department_doctor / super_admin
      -> department_id（工作人员当前部门）
```

微信与平台角色不存在直接映射。微信只完成外部身份认证，账号类型、角色、部门和权限以 Identity 数据库为准。

一个新用户的状态变化如下：

```text
首次微信登录
  -> 自动创建 patient 账号
  -> 用户登记手机号（self_reported）
  -> 超级管理员按完整手机号精确查找
  -> 线下确认本人、手机号和医生资格
  -> 设置 account_type=staff
  -> 创建工作人员档案并设置部门
  -> 授予 department_doctor
  -> authorization_version + 1
```

医生开通后，下一次刷新 Token 或重新登录即可获得最新的工作人员类型、医生角色、部门和权限。旧 Access Token 最长在当前短有效期结束后失效。

## 3. 数据库设计

### 3.1 拆表边界

Identity 首期保留规范化拆表，但必须区分“账号资料”和“围绕账号产生的关系或历史”：

- `identity_accounts` 是唯一的平台账号主表；
- 微信身份、手机号、工作人员资料和当前角色通过账号 ID 关联，不是新的用户；
- 当前明确限制一个账号最多一个手机号、一个当前角色，工作人员最多一个当前科室；
- 角色和权限是多对多关系，授权审计和 Outbox 是一对多历史数据，因此不能作为账号字段保存；
- 不允许仅为了抽象或假设中的未来需求继续创建一对一扩展表。新增表必须具备一对多关系、独立
  生命周期、独立安全要求或明确的业务边界之一。

11 张表的逐表职责、当前基数和未来合并或拆分条件统一记录在
`migrations/identity/README.md`。后续修改表结构时必须同步更新该说明，避免遗忘首期约束。

### 3.2 外部登录身份

新增 `identity_external_identities`：

| 字段 | 含义 |
|---|---|
| `id` | 绑定记录 ID |
| `account_id` | Hospital 账号 ID |
| `provider` | 首期固定为 `wechat` |
| `provider_app_id` | 小程序 AppID，防止不同小程序的 OpenID 混用 |
| `provider_subject` | 微信返回的 OpenID |
| `created_at` / `updated_at` | 创建和更新时间 |

`(provider, provider_app_id, provider_subject)` 必须唯一。OpenID、登录 code、session_key 不写入业务日志；`session_key` 首期不持久化。

### 3.3 用户登记手机号

新增 `identity_account_phones`，首期一个账号只保留一个当前手机号：

| 字段 | 含义 |
|---|---|
| `account_id` | Hospital 账号 ID，同时作为主键 |
| `phone_fingerprint` | 使用服务端密钥进行 HMAC-SHA256 后的精确查找值 |
| `phone_masked` | 仅供页面展示，例如 `138****1234` |
| `verification_status` | `self_reported` 或 `verified` |
| `verification_source` | `self_reported`、`sms`、`wechat` 或 `admin` |
| `verified_at` | 完成验证的时间，未验证时为空 |
| `created_at` / `updated_at` | 创建和更新时间 |

数据库不保存明文手机号。超级管理员输入完整手机号后，后端使用同一密钥计算指纹并进行等值查询，不提供手机号模糊搜索和全量枚举。手机号指纹设置唯一约束，避免同一个号码绑定多个账号。

本期没有短信或微信手机号验证能力，因此用户登记后状态保持 `self_reported`。超级管理员开通医生时必须提交 `offline_verified=true`；成功后将号码标记为 `verified/admin`。以后接入短信或微信手机号能力时沿用同一张表和状态机。

### 3.4 工作人员档案

继续使用现有 `identity_staff_profiles`：

- 医生开通时创建或更新工作人员档案；
- 首期必须选择一个有效部门；
- `staff_no` 仍然可空，未来可以补充工号；
- 医生开通、调岗和禁用都必须通过 Identity 业务操作，不能直接修改数据库。

## 4. 接口设计

### 4.1 公共接口

`POST /api/v1/auth/wechat/login`

```json
{
  "login_code": "wx.login 返回的一次性 code"
}
```

Identity 调用微信换取 OpenID，查找或自动创建患者账号，然后创建 Redis Refresh Session 并返回平台自己的 Access Token 与 Refresh Token。

### 4.2 当前用户接口

- `GET /api/v1/auth/me`：返回 Token 中的账号类型、角色、部门和权限；
- `PUT /api/v1/auth/me/phone`：当前用户登记或更换自报手机号。

登记手机号只证明账号持有人提交了这个号码，不表示手机号已经由运营商或微信验证。

### 4.3 超级管理员接口

- `POST /api/v1/admin/identity/accounts/search-by-phone`：按完整手机号精确查找，返回账号 ID、账号类型、状态、脱敏手机号、验证状态和当前角色；
- `POST /api/v1/admin/identity/doctors/promote`：传入账号 ID、部门 ID 和 `offline_verified=true`，将账号开通为部门医生。

两个接口都要求 `identity.authorization.manage` 权限。查找操作记录安全日志但不记录完整手机号；医生开通写入授权审计和 Outbox。

## 5. 首位超级管理员

系统不能依赖一个尚不存在的超级管理员创建自己。首期提供只在部署阶段手动执行的初始化命令：

```powershell
go run ./tools/identity-bootstrap-admin --account-id <已登录账号ID>
```

命令只允许把一个已经存在的账号提升为首位超级管理员；数据库已有超级管理员后默认拒绝再次执行。日常患者注册和医生开通不使用该命令。

## 6. 配置与安全要求

Identity Service 新增以下 Secret 配置：

```text
WECHAT_MINIAPP_APP_ID
WECHAT_MINIAPP_APP_SECRET
IDENTITY_PHONE_LOOKUP_KEY_BASE64
```

- AppSecret 和手机号查找密钥只通过运行环境注入，不进入 Git；
- 微信登录 code 只能使用一次，不落库、不写日志；
- 微信 `session_key` 不返回客户端，本期不持久化；
- 登录失败响应不暴露微信内部字段；
- 手机号只允许中国大陆 `+86` 的 11 位号码，后续需要国际号码时再扩展；
- 管理员查找必须是完整号码精确匹配，并纳入频率限制和审计扩展计划；
- 生产环境不开放按账号 ID 直接领取 Token 的开发后门。

## 7. 本期实现顺序

1. 新增第二版 Identity 数据库迁移；
2. 实现微信 provider 和外部身份账号自动注册事务；
3. 开放微信登录 RPC 和 HTTP API；
4. 增加 app-api 的 Access Token 本地校验中间件和 `/auth/me`；
5. 实现手机号登记与管理员精确查找；
6. 实现医生开通事务、授权版本、审计和事件；
7. 增加首位超级管理员初始化命令；
8. 使用微信 provider 的假实现完成自动化测试，真实 AppID/AppSecret 只用于人工联调。

## 8. 暂不进入本期

- 微信手机号快捷验证组件；
- 短信验证码供应商；
- 一个医生同时任职多个部门；
- 医生资质证书线上审核；
- 账号合并和手机号争议申诉；
- 后台管理页面本身。
