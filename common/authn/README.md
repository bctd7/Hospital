# common/authn

`authn` 只回答“调用者是谁，以及凭证是否有效”。

```text
principal.go       Principal、账号类型、角色
token.go           Ed25519 JWT 签发与验证
http.go            HTTP Bearer Token 中间件
grpc.go            gRPC Bearer Token 拦截器
validator.go       Token 验证后的运行时校验接口
```

这里不判断具体业务权限，也不判断科室资源范围。权限和范围进入
`common/authz`；授权版本校验及 Redis 实现进入 `common/authz/version`。
