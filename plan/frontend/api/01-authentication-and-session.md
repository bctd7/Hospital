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
  -> 返回 Hospital TokenPair

GET /api/v1/auth/me
  Authorization: Bearer <access_token>
  -> 返回 account_id、account_type、roles、department_id、permissions
```

手机号是主登录标识，数据库关联仍使用不可变 `account_id`。短信验证只证明用户控制该手机号，
不等同于医疗实名或医院患者主索引匹配。

`POST /api/v1/auth/wechat/login` 作为兼容接口保留，但当前入口页不调用它。OpenID、AppSecret、
微信 `session_key` 和 PNVS AccessKey 均不得进入前端。

## 3. 会话接口

| 接口 | 用途 |
|---|---|
| `POST /api/v1/auth/token/refresh` | 使用 Refresh Token 轮换整对 Token |
| `POST /api/v1/auth/token/revoke` | 退出时撤销 Refresh Session |
| `GET /api/v1/auth/me` | 获取当前 Principal |
| `PUT /api/v1/auth/me/phone` | 兼容的手机号绑定接口，不用于覆盖 PNVS 已验证登录号码 |

前端 `api/client.ts` 是唯一 `uni.request` 入口：

- 受保护请求自动添加 Bearer Access Token；
- 401 时调用全局共享的 `refreshOnce()`；
- 同一时刻只允许一个刷新请求；
- 刷新成功后原请求最多重试一次；
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

Storage key 为 `hospital:session`。登录、恢复和刷新后重新校验 Principal 与可用应用版本；角色被
撤销后不能继续选择工作人员端。退出先清理本地状态，再尽力撤销服务端 Refresh Session。

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
- 退出立即清除本地会话；
- TypeScript、前端测试、小程序构建和后端测试通过；
- 构建产物不包含服务端 Secret。
