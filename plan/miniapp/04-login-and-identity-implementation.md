# 小程序登录与身份验证实现计划

> 文档状态：前后端代码已实现，真实微信环境联调待验证
>
> 适用范围：微信登录、Hospital 会话、当前账号身份、Token 刷新和退出
>
> 关联后端方案：`../05-wechat-registration-and-doctor-onboarding.md`

## 1. 实现目标

本阶段在现有小程序页面外壳上增加统一登录和身份状态，不开发患者实名或就诊人业务：

1. 调用 `uni.login` 获取一次性微信 code；
2. 使用 HTTP 把 code 发送给 Hospital App API；
3. 接收并管理 Hospital Access Token 和 Refresh Token；
4. 从 Access Token 对应接口读取当前账号类型、角色、科室和权限；
5. 统一处理 Access Token 过期、Refresh Token 轮换和退出；
6. 登录失败时允许浏览静态页面，但依赖账号的操作必须显示未登录状态。

这里存在三层不同概念：

```text
微信身份认证：微信 code -> OpenID，证明是同一个微信用户
Hospital 账号认证：OpenID -> account_id -> Hospital Token
Hospital 权限上下文：Token -> account_type / role / department / permissions
```

以上都不等于患者实名。首版一个 Hospital 登录账号只管理本人；登录后继续使用 OpenID +
自报手机号 + 本人资料完成基础身份，具体见 `05-self-patient-basic-identity.md`。

## 2. HTTP 与 RPC 边界

小程序调用的是 HTTP JSON：

```http
POST /api/v1/auth/wechat/login
```

本地完整地址默认是：

```text
http://127.0.0.1:8888/api/v1/auth/wechat/login
```

该接口属于 `service/app/api`。`service/identity/rpc` 是 App API 在后端内部调用的 gRPC 服务，
不是小程序可直接访问的 HTTP 服务。

```text
小程序
  -- HTTP JSON :8888 --> App API
  -- 后端内部 gRPC :8080 --> Identity RPC
  -- 微信 HTTPS --> code2Session
```

| 边界 | 契约 | 用途 |
|---|---|---|
| 小程序 -> App API | `contracts/api/identity-auth.api` | 对外 HTTP 请求和 JSON 响应 |
| App API -> Identity RPC | `contracts/proto/identity/v1/identity.proto` | 后端内部 gRPC 调用 |
| Identity RPC -> 微信 | 微信 `code2Session` HTTPS 接口 | 服务端使用 AppID 和 AppSecret 换取 OpenID |

小程序不依赖 `.proto` 生成代码，不知道 Identity RPC 地址，也不能获得 AppSecret 或 session_key。

## 3. 当前代码实际执行状态

### 3.1 小程序登录链路已经接入

当前调用链：

```text
pages/entry/index.vue
  -> enterMiniapp()
      -> saveDisplayProfile()
      -> session.initializeFromWechat()
          -> getWechatLoginCode()
              -> uni.login()
          -> authApi.wechatLogin(code)
              -> POST /api/v1/auth/wechat/login
          -> 保存 Hospital Token
          -> authApi.getCurrentIdentity()
              -> GET /api/v1/auth/me
      -> uni.switchTab('/pages/home/index')
```

成功后 Session 进入 `authenticated`；任一步失败都会清理不完整会话并进入 `guest`，随后仍进入主应用。

### 3.2 后端微信登录调用链已经完成

前端发出 `POST /api/v1/auth/wechat/login` 后，函数调用顺序是：

```text
routes.go
  -> WeChatLoginHandler()
      -> httpx.Parse()
      -> WeChatLoginLogic.WeChatLogin()
          -> IdentityService.WeChatLogin()          // gRPC Client
              -> IdentityServiceServer.WeChatLogin()
                  -> identity logic WeChatLogin()
                      -> AccountManager.WeChatLogin()
                          -> WeChatClient.ExchangeLoginCode()
                              -> 微信 code2Session
                          -> MySQLStore.FindOrCreateWeChatAccount()
                              -> 查找或创建 patient 账号
                          -> SessionManager.Start()
                              -> MySQLStore.GetAuthorizationContext()
                              -> TokenManager.Issue()
                              -> RedisSessionStore.Create()
      <- TokenResponse
```

文件对应关系：

| 函数层 | 文件 |
|---|---|
| HTTP 路由 | `service/app/api/internal/handler/routes.go` |
| HTTP Handler | `service/app/api/internal/handler/auth/we_chat_login_handler.go` |
| App API Logic | `service/app/api/internal/logic/auth/we_chat_login_logic.go` |
| gRPC Client | `service/identity/rpc/identityservice/identity_service.go` |
| gRPC Server | `service/identity/rpc/internal/server/identity_service_server.go` |
| Identity Logic | `service/identity/rpc/internal/logic/we_chat_login_logic.go` |
| 登录编排 | `service/identity/rpc/internal/account/manager.go` |
| 微信接口 | `service/identity/rpc/internal/login/wechat.go` |
| MySQL | `service/identity/rpc/internal/repository/mysql_store.go` |
| 会话签发 | `service/identity/rpc/internal/session/manager.go` |
| Redis Session | `service/identity/rpc/internal/repository/redis_session_store.go` |

### 3.3 `/auth/me` 使用本地 Token 校验

`GET /api/v1/auth/me` 不调用 Identity RPC，也不查询 MySQL：

```text
HTTP Authorization: Bearer <access_token>
  -> AccessTokenMiddleware.Handle()
      -> authn.HTTPMiddleware()
          -> bearerToken()
          -> TokenManager.Verify()
          -> ContextWithPrincipal()
  -> CurrentIdentityHandler()
      -> CurrentIdentityLogic.CurrentIdentity()
          -> PrincipalFromContext()
          -> CurrentIdentityResponse
```

这是此前确定的“权限本地判断”：Identity 在登录或刷新时把账号类型、角色、科室和权限写进
短期 Access Token，App API 本地验签后即可得到 Principal。账号禁用或权限变化最迟在当前
Access Token 过期后生效；刷新时会重新读取数据库中的最新权限。

### 3.4 Refresh Token 调用链

```text
POST /api/v1/auth/token/refresh
  -> RefreshTokenHandler()
  -> RefreshTokenLogic.RefreshToken()
  -> IdentityService.RefreshAccessToken()           // gRPC
  -> RefreshAccessTokenLogic.RefreshAccessToken()
  -> SessionManager.Refresh()
      -> RedisSessionStore.Get()
      -> 校验 Refresh Token 哈希、过期和重放
      -> MySQLStore.GetAuthorizationContext()
      -> TokenManager.Issue()
      -> RedisSessionStore.Rotate()
  <- 新 Access Token + 新 Refresh Token
```

Refresh Token 每次使用都会轮换。前端如果并发提交同一个旧 Token，后端会按重放处理并撤销
会话，因此前端必须使用一个共享刷新任务，不能让多个请求同时刷新。

### 3.5 退出调用链

```text
POST /api/v1/auth/token/revoke
  -> RevokeTokenHandler()
  -> RevokeTokenLogic.RevokeToken()
  -> IdentityService.RevokeRefreshToken()            // gRPC
  -> RevokeRefreshTokenLogic.RevokeRefreshToken()
  -> SessionManager.Revoke()
  -> RedisSessionStore.Revoke()
```

退出主要撤销 Redis 中的 Refresh Session。本期 Access Token 是短期 JWT，退出后可能在剩余
有效期内继续通过本地验签，因此前端必须立即清除本地 Token。

## 4. 小程序目标调用结构

小程序完成后的调用关系应为：

```text
entry 页面
  -> session.initializeFromWechat()
      -> getWechatLoginCode()
          -> uni.login()
      -> authApi.wechatLogin(code)
          -> apiClient.request()
      -> session.saveTokenPair()
      -> authApi.getCurrentIdentity()
      -> session.setPrincipal()
  -> 无论成功或失败，进入主应用

其他业务页面
  -> 业务 api module
      -> apiClient.request()
          -> 添加 Access Token
          -> Token 过期时 session.refreshOnce()
          -> 原请求最多重试一次
```

页面只调用 Session 或业务 API，不直接调用 `uni.request`，也不自行读取和刷新 Token。

## 5. 前端文件规划

在现有 `apps/miniapp/src` 中新增或修改：

```text
src/
├── api/
│   ├── client.ts          # 唯一 uni.request 入口、Authorization、错误转换和单次重试
│   └── auth.ts            # wechatLogin、getCurrentIdentity、refresh、revoke
├── config/
│   └── environment.ts     # API Base URL，只允许非敏感配置
├── stores/
│   └── session.ts         # TokenPair、Principal、初始化、刷新和退出
├── types/
│   └── auth.ts            # HTTP 请求、Token 和 CurrentIdentity 类型
├── utils/
│   └── wechatCode.ts      # 修改为 Promise<string>，只负责获取 code
└── pages/entry/index.vue  # 调用 session.initializeFromWechat，不实现网络细节
```

当前项目没有引入 Pinia。首期 Session Store 可以使用 Vue `reactive` 和模块单例实现，等出现
多个复杂 Store、插件或持久化需求时再决定是否引入状态管理库。

## 6. 主要前端函数职责

### 6.1 `getWechatLoginCode(): Promise<string>`

- 包装 `uni.login({ provider: "weixin" })`；
- 成功返回 code；
- 超时或失败抛出统一错误；
- 不保存、不输出 code。

### 6.2 `authApi.wechatLogin(code)`

- 请求 `POST /api/v1/auth/wechat/login`；
- 不携带 Access Token；
- 返回 `TokenPair`；
- 不负责写 Storage。

### 6.3 `session.initializeFromWechat()`

- 编排获取 code、登录、保存 Token、读取 `/auth/me`；
- 成功进入 `authenticated` 状态；
- 失败清理不完整会话并进入 `guest` 状态；
- 不负责页面跳转和头像昵称保存。

### 6.4 `apiClient.request()`

- 拼接 API Base URL；
- 统一超时和响应错误；
- 需要鉴权的接口自动携带 Bearer Access Token；
- Access Token 过期时调用 `session.refreshOnce()`；
- 刷新成功后原请求最多重试一次；
- 登录、刷新接口本身禁止触发自动刷新，避免递归。

### 6.5 `session.refreshOnce()`

- 整个应用同一时刻只有一个刷新 Promise；
- 使用当前 Refresh Token 请求新 TokenPair；
- 原子替换 Access Token 和 Refresh Token；
- 失败时清空会话，不能继续使用旧 Token。

### 6.6 `session.logout()`

- 有 Refresh Token 时尽力调用 revoke；
- 无论后端调用是否成功，都清除本地 Token 和 Principal；
- 不删除用户主动选择的本地头像、昵称等展示资料。

## 7. Session 状态模型

首期只需要以下状态：

| 状态 | 含义 |
|---|---|
| `idle` | 尚未初始化 |
| `authenticating` | 正在获取微信 code 或请求登录 |
| `authenticated` | 已有有效 Hospital 会话和 Principal |
| `guest` | 登录失败或用户当前没有有效会话，但可浏览静态页面 |

Session 保存：

```text
status
accessToken
refreshToken
accessExpiresAt
refreshExpiresAt
principal
```

Storage key 统一使用 `hospital:session`。微信 code、OpenID、AppSecret 和 session_key 均不进入
Storage。完整 Token 不得写入控制台、错误提示或埋点。

## 8. 需要使用的 HTTP 接口

| 接口 | 是否鉴权 | 前端用途 |
|---|---|---|
| `POST /api/v1/auth/wechat/login` | 否 | 首次登录或重新登录 |
| `GET /api/v1/auth/me` | Access Token | 将 Token 中 Principal 转成前端身份状态 |
| `POST /api/v1/auth/token/refresh` | 否，提交 Refresh Token | 轮换整对 Token |
| `POST /api/v1/auth/token/revoke` | 否，提交 Refresh Token | 退出当前会话 |
| `PUT /api/v1/auth/me/phone` | Access Token | 后续登记手机号，不属于首次登录必需步骤 |

微信登录请求：

```json
{
  "login_code": "uni.login 返回的一次性 code"
}
```

返回的是 Hospital Token，不返回 OpenID：

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "access_expires_in_seconds": 900,
  "refresh_expires_in_seconds": 2592000
}
```

## 9. 手写与生成代码边界

本阶段已手写完成：

- `apps/miniapp/src/api`、`config`、`stores` 和 `types/auth.ts`；
- `wechatCode.ts` 的返回值和错误处理；
- 入口页对 Session 的调用；
- 前端会话和 API Client 测试。

本阶段未修改或重新生成：

- `contracts/api/identity-auth.api`，HTTP 契约已经存在；
- `contracts/proto/identity/v1/identity.proto`，内部 RPC 契约已经存在；
- goctl 生成的 Handler、Types 和 gRPC 文件；
- Identity 数据库表和 Redis Session 结构；
- AppSecret 相关后端代码。

如果实现过程中发现请求或响应字段确实不够，应先修改 `.api` 或 `.proto` 契约，再重新生成，
不能直接手改生成文件。

## 10. 实现顺序

当前第 1 至第 9 步代码已完成：

1. 建立 `types/auth.ts` 和非敏感 `environment.ts`；
2. 实现不带自动刷新的基础 `api/client.ts`；
3. 实现 `api/auth.ts` 四个会话接口；
4. 将 `attemptWechatCode()` 改为 `getWechatLoginCode()`；
5. 实现 Session 状态、Token 持久化和 `initializeFromWechat()`；
6. 入口页接入 Session，同时保留登录失败后可浏览的行为；
7. 接入 `/auth/me`，在内存中建立 Principal；
8. 实现共享 `refreshOnce()` 和 API 原请求单次重试；
9. 实现退出和本地会话清理。

仍待完成：

10. 增加前端会话单元测试，并在开发者工具中完成真实微信登录验证。

前六步形成登录主链路；第七至第九步形成完整首期会话闭环。当前代码已覆盖首期功能，自动化
测试和真实环境验证不能因为代码存在而标记为完成。

## 11. 验收标准

- 小程序只请求 App API 的 HTTP 地址，不请求 `:8080` Identity gRPC；
- 第一次微信登录创建 `patient` 账号，重复登录复用同一个 `account_id`；
- 登录成功后 Session 同时具有 TokenPair 和 Principal；
- `/auth/me` 通过 App API 本地 JWT 验签完成，不额外调用 Identity RPC；
- 多个请求同时过期时只发送一次 Refresh 请求；
- Refresh Token 轮换后不再使用旧值；
- 退出后立即清理前端会话，并尽力撤销 Redis Refresh Session；
- 登录失败可以进入静态主页面，受保护操作不会伪装成成功；
- 前端和日志中没有 AppSecret、微信 code、session_key、OpenID 或完整 Token；
- TypeScript 检查、小程序构建和后端测试通过。

## 12. 当前结论

后端登录、Token、权限上下文、刷新和注销，以及小程序 `api + session` 两层和入口页调用均已完成。
当前不需要增加另一套登录服务，也不需要修改 Identity RPC。下一步是启动依赖服务，在微信开发者
工具中使用真实 `uni.login` code 验证首次注册、重复登录、Token 刷新和访客降级。
