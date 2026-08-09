# Identity 登录与授权实现说明

## 1. 当前实现范围

本版本已经实现权限管理基础：

- 固定权限动作目录；
- 超级管理员和部门医生两个角色；
- 角色、部门和账号状态变更；
- 授权版本、变更审计和 Outbox 事件；
- JWT Access Token 的签发与本地验证组件；
- 本部门可操作、跨部门只读的公共授权规则；
- Identity RPC 授权管理契约。

本版本已经实现 Redis Refresh Session、Refresh Token 轮换和注销接口。微信、手机号登录仍只预留供应商适配接口；在 AppID、AppSecret、短信供应商和账号绑定规则确定前，不对外提供伪造的登录实现。

## 2. 微信小程序登录

推荐流程：

```text
小程序调用 wx.login
  -> 得到短期 login code
  -> POST /api/v1/auth/wechat/login
  -> Identity 的 WeChatProvider 调用 code2Session
  -> 得到 OpenID、session_key，以及满足条件时的 UnionID
  -> 按 AppID + OpenID 查找或创建平台账号
  -> Identity 签发平台自己的 Access Token 和 Refresh Token
```

注意事项：

- `code` 只能作为短期交换凭证，不能当作平台 Token；
- `session_key` 只保存在服务端，不能返回给小程序；
- AppSecret 只通过服务端 Secret 注入，不能写入小程序或仓库；
- 平台 Access Token 使用 Ed25519 签名；Identity 独占私钥，其他业务服务只持有公钥并在本地验证；
- OpenID 需要与具体小程序 AppID 一起识别，不能假设不同小程序的 OpenID 相同；
- UnionID 只有在满足微信开放平台关联条件时才可能返回，不能作为首版必填字段；
- 微信头像和昵称不是可靠登录标识，也不应作为完成登录的强制条件。

官方资料：

- [wx.login](https://developers.weixin.qq.com/miniprogram/dev/api/open-api/login/wx.login.html)
- [小程序登录凭证校验 code2Session](https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html)

## 3. 微信授权手机号

手机号能力和 `wx.login` 是两套不同凭证：

```text
用户主动点击 open-type="getPhoneNumber" 按钮
  -> 小程序收到 phone code
  -> POST /api/v1/auth/wechat/phone
  -> Identity 的 WeChatProvider 使用服务端 access_token 和 phone code
  -> 调用 getuserphonenumber
  -> 返回手机号并建立账号绑定
```

`phone code` 与 `wx.login` 返回的 `login code` 不能混用。手机号属于敏感信息：日志只记录绑定结果和脱敏号码，不记录完整手机号、code、access_token 或 session_key。

是否允许“获得手机号后自动合并已有账号”需要单独设计。首版默认不自动合并，发现冲突时进入明确的账号确认流程。

官方资料：[获取手机号 getPhoneNumber](https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-info/phone-number/getPhoneNumber.html)

## 4. 短信验证码登录

短信验证码不是微信接口，而是平台自建能力，需要接入短信供应商。预留流程：

```text
POST /api/v1/auth/sms/code
  -> 校验手机号和用途
  -> 限制手机号、IP、设备和账号频率
  -> 生成一次性验证码
  -> Redis 只保存验证码哈希、有效期和剩余尝试次数
  -> SMSProvider 发送短信

POST /api/v1/auth/sms/login
  -> 原子消费验证码
  -> 查找或创建允许的账号
  -> 签发平台 Token
```

最低安全要求：

- 验证码短期有效、一次性消费；
- 限制发送频率、校验次数和每日总量；
- 不在数据库、Redis、日志中保存明文验证码；
- 发送接口无论手机号是否已注册都返回一致响应，避免枚举账号；
- 患者登录、绑定手机号和找回账号使用不同 `purpose`；
- 后台工作人员首版不建议只依赖短信验证码，后续应结合医院账号或更强认证。

## 5. 预留接口

对外 API 契约位于 `contracts/api/identity-auth.api`：

| 接口 | 用途 | 当前状态 |
|---|---|---|
| `POST /api/v1/auth/wechat/login` | 微信 code 登录 | 已预留契约，等待微信配置和账号规则 |
| `POST /api/v1/auth/wechat/phone` | 获取并绑定微信手机号 | 已预留契约，等待微信配置和绑定规则 |
| `POST /api/v1/auth/sms/code` | 发送短信验证码 | 已预留契约，等待短信供应商 |
| `POST /api/v1/auth/sms/login` | 手机验证码登录 | 已预留契约，等待账号创建规则 |
| `POST /api/v1/auth/token/refresh` | 刷新平台 Token | 已实现 Redis Session 校验和 Token 轮换 |
| `POST /api/v1/auth/token/revoke` | 注销当前 Refresh Session | 已实现，重复注销保持幂等 |

代码适配端口位于 `service/identity/rpc/internal/login/providers.go`。后续微信和短信实现只能依赖这些端口，不能把供应商 HTTP 字段泄漏进 Identity 业务逻辑。

## 6. Access Token 与 Refresh Token

当前会话设计为：

```text
Access Token
  -> Ed25519 JWT
  -> 默认 15 分钟
  -> 业务服务使用公钥本地验证
  -> 不写入 Redis

Refresh Token
  -> 32 字节安全随机数形成的不透明凭证
  -> 默认 30 天绝对有效期
  -> Redis 只保存 SHA-256 哈希，不保存原文
  -> 每次刷新后立即轮换，旧 Token 不可再次使用
```

Redis Key 默认使用 `identity:refresh:{session_id}`。Session 保存账号、会话族、Token 哈希、授权版本和绝对过期时间。刷新时 Identity 会重新从 MySQL 读取账号最新状态、角色、科室和权限，再签发新的 Access Token，因此 Refresh Session 不是权限事实来源。

刷新流程：

```text
app-api 接收 Refresh Token
  -> 调用 Identity.RefreshAccessToken RPC
  -> Redis 读取 Session
  -> MySQL 读取账号最新授权上下文
  -> 签发新 Access Token
  -> Redis Lua 原子替换 Refresh Token 哈希
  -> 返回新 Access Token 和新 Refresh Token
```

若旧 Refresh Token 在轮换后被再次提交，Identity 将其视为重放，撤销该 Session，要求用户重新登录。并发发起两次刷新也可能触发这一保护，因此客户端必须串行刷新，并在成功后原子替换本地 Token。

当前 `SessionManager.Start` 仅供未来已经完成微信或短信身份校验的登录逻辑调用，没有开放“传入账号 ID 直接领取 Token”的 RPC，避免绕过登录认证。

## 7. 尚未确定的产品规则

1. 微信首次登录是否自动创建患者账号；
2. 是否强制绑定手机号后才能预约；
3. 相同手机号已存在账号时如何确认和合并；
4. 患者是否可以完全使用短信登录而不绑定微信；
5. 工作人员使用医院账号、手机号还是统一身份平台登录；
6. 是否允许一个账号同时保留多个设备 Session；
7. 账号注销、解绑微信和换绑手机号的处理方式；
8. 是否要求工作人员权限变化后立即使尚未过期的 Access Token 失效。

## 8. 当前采用的首期登录方案

首期已经确定并实现以下规则：

1. 个人主体小程序使用 `wx.login` 和 `code2Session` 完成真实微信登录；
2. 新 OpenID 自动创建 `patient` 账号；
3. 用户登录后可以自行登记手机号，但号码默认是 `self_reported`；
4. 超级管理员只能用完整手机号精确查找账号；
5. 超级管理员在线下确认本人、手机号和医生资格后，将账号开通为指定部门的 `department_doctor`；
6. 数据库不保存明文手机号，只保存带密钥的 HMAC 指纹和脱敏展示值；
7. 第一位超级管理员通过一次性部署命令初始化。

详细数据结构、接口和安全边界见 `plan/05-wechat-registration-and-doctor-onboarding.md`。
