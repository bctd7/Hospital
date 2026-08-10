# 认证、会话与身份接口

> 状态：当前实现基线

## 1. 调用边界

小程序只调用 `service/app/api` 暴露的 HTTP JSON 接口，不访问 Identity gRPC。默认开发地址为
`http://127.0.0.1:8888`，真机联调必须改为手机可访问的地址。

```text
miniapp -- HTTP /api/v1 --> app-api -- gRPC --> identity-rpc
                                      -> MySQL / Redis / PNVS
```

前端契约以 `contracts/api/identity-auth.api` 和 `identity-session.api` 为准，不能根据数据库字段
自行扩展响应。

## 2. 当前登录流程

```text
POST /api/v1/auth/phone/code
  body: { phone }
  -> PNVS 发送验证码

POST /api/v1/auth/phone/login
  body: { phone, verification_code }
  -> PNVS 校验 PASS
  -> 按手机号指纹查找或创建账号
  -> 已禁用账号拒绝签发 Token，返回稳定的 ACCOUNT_DISABLED 业务错误
  -> 返回 Hospital TokenPair

GET /api/v1/auth/me
  Authorization: Bearer <access_token>
  -> 返回 account_id、account_type、roles、department_id、permissions
```

手机号是主登录标识，数据库关联仍使用不可变 `account_id`。短信验证只证明用户控制该手机号，
不等同于医疗实名或医院患者主索引匹配。

账号禁用不等同于短信风控。验证码发送接口保持统一响应，不在验证手机号所有权之前暴露账号是否
存在或已禁用；验证码校验成功后再读取账号状态，禁用账号不能换取 Access Token 或 Refresh Token。
如需阻止短信轰炸或特定号码收码，应使用独立的短信风控状态和手机号/IP 限流。

账号昵称目前只保存在小程序本地。为支持超级管理员按昵称查询，后续增加经过用户确认的账号展示资料
同步接口；昵称不唯一、不参与登录，也不能作为账号或患者实名依据。

计划增加：

```http
GET /api/v1/auth/me/display-profile
PUT /api/v1/auth/me/display-profile
    body: { nickname }
```

用户在就诊人管理页面确认昵称后调用 `PUT`；本机 Storage 降级为界面缓存，不能继续作为唯一数据源。

`POST /api/v1/auth/wechat/login` 作为兼容接口保留，但当前入口页不调用它。OpenID、AppSecret、
微信 `session_key` 和 PNVS AccessKey 均不得进入前端。

## 3. 会话接口

| 接口 | 用途 |
|---|---|
| `POST /api/v1/auth/token/refresh` | 使用 Refresh Token 轮换整对 Token |
| `POST /api/v1/auth/token/revoke` | 退出时撤销 Refresh Session |
| `GET /api/v1/auth/me` | 获取当前 Principal |
| `PUT /api/v1/auth/me/phone` | 兼容的手机号绑定接口，不用于覆盖 PNVS 已验证登录号码 |
| `GET/PUT /api/v1/auth/me/display-profile` | 计划中的账号昵称读取与同步接口 |

前端 `api/client.ts` 是唯一 `uni.request` 入口：

- 受保护请求自动添加 Bearer Access Token；
- 401 时调用全局共享的 `refreshOnce()`；
- 同一时刻只允许一个刷新请求；
- 刷新成功后原请求最多重试一次；
- Access Token 仅自然过期、Redis 版本缺失且 Refresh Session 版本仍与 MySQL 一致时允许无感刷新；角色、
  当前科室或账号状态变化造成版本不一致时，Refresh 必须失败，前端清理会话并回到登录页；
- 登录和刷新接口本身不触发递归刷新；
- 不在日志、Toast 或埋点输出 Token、手机号、验证码或微信 code。

## 4. 前端状态

`stores/session.ts` 维护：

```text
status = idle / authenticating / authenticated / guest
principal
TokenPair
appVariant = patient / staff
```

Storage key 为 `hospital:session`。登录、恢复和刷新后重新校验 Principal 与可用应用版本；角色、当前
科室或账号状态发生变化后不在已打开的会话中无感切换身份，而是由 Refresh 版本冲突清理会话并要求
重新登录。退出先清理本地状态，再尽力撤销服务端 Refresh Session。

## 5. 角色与权限

- `department_doctor` 与 `super_admin` 都可使用工作人员端；
- 两者共享页面框架，不复制两套后台；
- 页面或菜单差异读取 `permissions`；
- 后端业务服务必须再次完成最终授权，前端可见性不是安全边界。

超级管理员后续增加“身份录入/医生开通”等专属入口，医生不显示；具体权限代码以 Identity 权限
目录和后端契约为准。

## 6. 联调与验收

- 手机号发送、校验、重复登录和错误验证码均通过真实或可控测试环境验证；
- 同一手机号重复登录复用相同 `account_id`；
- 并发 401 只发生一次 Token 刷新；
- Refresh Token 轮换后不再使用旧值；
- 未发生授权变化时 Access Token 过期可以无感刷新；发生授权变化时旧 Refresh Session 被拒绝并回到
  登录流程；
- 退出立即清除本地会话；
- TypeScript、前端测试、小程序构建和后端测试通过；
- 构建产物不包含服务端 Secret。
