# 认证与会话消费方案

> 状态：当前实现基线
>
> 本文只规定小程序如何消费认证接口。准确路径、字段、鉴权和错误以 `contracts/api/` 与生成的
> `docs/api/openapi.json` 为准。

## 1. 调用边界

小程序只调用 App API，不直接访问 Identity gRPC：

```text
miniapp -> HTTP App API -> gRPC Identity -> MySQL / Redis / PNVS
```

真机联调使用手机可访问的局域网地址；PNVS AccessKey、JWT 私钥和其他服务端 Secret 不进入小程序。

## 2. 登录与展示资料

- 主登录使用手机号和阿里云 PNVS 验证码；微信登录只保留兼容能力，不作为当前页面入口；
- 手机号只证明号码控制权，不代表医疗实名，也不作为数据库业务主键；
- 登录后读取当前 Principal，再按角色和 permissions 确定患者端或工作人员端；
- 昵称由 Identity 保存，本地 Storage 只做按 `account_id` 隔离的界面缓存；
- 禁用账号在验证码校验后拒绝签发 Token，发送验证码阶段不暴露账号是否存在。

## 3. 会话策略

`api/client.ts` 是唯一网络请求入口：

1. 受保护请求自动携带 Bearer Access Token；
2. 收到 `401` 后所有并发请求共享一次 `refreshOnce()`；
3. 刷新成功后原请求最多重试一次；
4. 授权版本变化、Refresh Session 失效或账号禁用时清理会话并回登录页；
5. 登录和刷新请求自身不得递归刷新；
6. 退出先清除本地状态，再尽力撤销服务端 Refresh Session。

Token、验证码、完整手机号和微信临时 code 不进入日志、Toast、埋点或错误上报正文。

## 4. 客户端状态

会话 Store 只维护认证状态、Principal、TokenPair 和当前应用版本。角色或科室变化后不在旧会话中
自行拼装新身份，必须重新取得服务端签发的 Principal。

## 5. 验收边界

- 同一手机号重复登录复用同一账号；
- 并发 `401` 只触发一次刷新；
- Refresh Token 轮换后旧值不再使用；
- 授权变化导致旧会话退出，普通 Access Token 过期可以正常刷新；
- 构建产物和客户端日志不包含服务端 Secret 或敏感认证材料。
