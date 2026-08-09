# 手机号主登录与阿里云 PNVS 短信认证

> 文档状态：当前登录认证基线
>
> 取代范围：原先以微信 OpenID 为首要登录身份、手机号仅为自报资料的首版方案

## 1. 决策

小程序改为使用阿里云号码认证服务（PNVS）的短信认证能力完成注册和登录：

```text
输入手机号
  -> POST /api/v1/auth/phone/code
  -> Identity 调用 SendSmsVerifyCode
  -> 用户输入验证码
  -> POST /api/v1/auth/phone/login
  -> Identity 调用 CheckSmsVerifyCode
  -> PASS 后按手机号查找或创建账号
  -> 签发 Hospital Access Token 和 Refresh Token
```

微信 `wx.login`、OpenID 和 AppSecret 不再是小程序登录的必需条件。旧微信登录接口暂时保留用于
兼容和迁移，但新小程序入口不再调用它。

## 2. 手机号是唯一登录标识，不是数据库物理主键

手机号是唯一、首要登录标识；数据库仍使用不可变的 `account_id` UUID 作为主键。

原因：

- 用户可能换手机号；
- 运营商可能回收并重新发放号码；
- 手机号属于敏感信息，数据库当前只保存 HMAC 指纹和脱敏值；
- 预约、报告和审计等外键不能随手机号变化而整体迁移。

数据库唯一性由 `identity_account_phones.phone_fingerprint` 的唯一索引保证：

```text
手机号 -> 规范化为 +86xxxxxxxxxxx -> HMAC-SHA256 -> 唯一指纹
```

一个手机号只能对应一个 Hospital 账号，一个账号当前也只能绑定一个手机号。

## 3. PNVS 产品边界

本项目使用“短信认证服务”，不使用“一键登录/本机号码校验 SDK”。后者的 uni-app 原生插件只
支持 Android/iOS App，不能用于微信小程序；短信认证只在服务端调用，因此可以用于小程序。

阿里云负责动态生成和校验验证码：

- `SendSmsVerifyCode` 使用平台预置签名和登录/注册模板；
- `TemplateParam` 使用 `##code##`，不由 Hospital 生成固定验证码；
- `ReturnVerifyCode=false`，验证码不会返回 Hospital 服务；
- `CheckSmsVerifyCode` 只有返回 `PASS` 才能登录；
- AccessKey 仅保存在 Identity Service 的运行环境中。

## 4. 注册与登录规则

验证码通过后：

1. 规范化手机号并计算 HMAC 指纹；
2. 指纹已存在：复用原账号，并把手机号状态更新为 `verified/sms`；
3. 指纹不存在：事务创建 `patient` 账号和 `verified/sms` 手机号绑定；
4. 使用内部 `account_id` 创建 Refresh Session；
5. 返回 Hospital Token，不向前端返回手机号明文。

医生和管理员仍然使用同一个手机号登录入口。登录后 Identity 根据账号记录返回其真实
`account_type`、角色、科室和权限，用户不能在登录页面自行选择医生或管理员身份。

## 5. 对外接口

```http
POST /api/v1/auth/phone/code
Content-Type: application/json

{"phone":"13800138000"}
```

成功响应只表示发送请求已受理：

```json
{"accepted":true,"retry_after_seconds":60}
```

```http
POST /api/v1/auth/phone/login
Content-Type: application/json

{"phone":"13800138000","verification_code":"123456"}
```

校验通过后返回现有 `TokenResponse`。验证码错误或过期统一返回认证失败，不能暴露账号是否存在。

## 6. 配置和安全

Identity Service 使用以下环境配置：

```text
ALIBABA_CLOUD_ACCESS_KEY_ID
ALIBABA_CLOUD_ACCESS_KEY_SECRET
ALIYUN_PNVS_SIGN_NAME
ALIYUN_PNVS_TEMPLATE_CODE
ALIYUN_PNVS_SCHEME_NAME（可选，但发送和校验必须一致）
```

API 区域使用 `cn-shanghai`，Endpoint 使用 `dypnsapi.aliyuncs.com`。生产环境使用 RAM 子账号并仅
授予 PNVS 发送和校验权限。接口还需要：

- 阿里云发送间隔限制，首版 60 秒；
- API 网关/IP 级限流，防止短信轰炸和费用攻击；
- 同一手机号日级次数限制；
- 日志只记录请求 ID 和结果，不记录完整手机号或验证码；
- 发生号码换绑、号码回收争议时走人工申诉，不能自动合并账号。

原 `PUT /api/v1/auth/me/phone` 不能覆盖 `verified/sms` 登录手机号。后续换号必须新增专用流程，
验证新号码，并根据风险决定是否同时验证旧号码或转人工处理。

## 7. 医疗数据边界

短信验证码证明的是“当前用户控制这个手机号”，仍不等同于法定实名或医院患者身份。手机号
登录成功后可以创建当前账号的新预约，但不能凭手机号自动读取医院既有病历、检查报告或其他
患者资料。历史医疗数据仍需患者主索引、证件信息或医院核验建立绑定。

## 8. 前端登录与退出流程

小程序入口页提供手机号、短信验证码、获取验证码和“手机号登录 / 注册”控件。头像和昵称可以
同时采集为展示资料，但不能参与身份认证，也不能代替患者实名姓名。

登录成功后，就诊人页只读展示后端返回或登录流程保存的脱敏手机号，不再提供“登录后绑定
手机号”入口。已登录用户可以从就诊人页执行“退出登录”：前端先清除本地 Access Token、
Refresh Token 和当前会话，再请求服务端撤销 Refresh Token，最后返回手机号登录页。退出登录
不删除账号、本人档案、预约或报告；不可恢复的账号注销必须另行设计身份确认和数据保留流程。

## 9. 验收标准

- 个人主体小程序不依赖微信手机号组件或 OpenID 即可注册登录；
- 同一手机号重复登录始终得到同一个 `account_id`；
- 并发首次登录不会创建两个账号；
- 验证码错误、过期或阿里云返回非 `PASS` 时不签发 Token；
- 新绑定状态为 `verified/sms`，不再是 `self_reported`；
- 医生和管理员通过手机号登录后仍获得原有角色，不会被重新注册为患者；
- 手机号和验证码不进入日志、Token 或普通业务响应；
- 手机号不直接作为预约、报告、审计等业务表的外键。
- 就诊人页不允许重新录入或覆盖已验证登录手机号；
- 退出登录立即清除本地会话并返回手机号登录页，不被服务端撤销失败阻塞。
