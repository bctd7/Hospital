# HTTP 接口文档

本目录保存由 go-zero `.api` 契约生成的 HTTP 接口文档。

- [`openapi.json`](./openapi.json)：当前 App API 的 Swagger 2.0 快照；
- `contracts/api/app.api`：唯一契约入口；
- `contracts/api/*.api`：按认证、账号管理和组织目录拆分的契约源文件。

`openapi.json` 是生成物，不手工修改。业务说明、权限边界和状态机写入 `plan/`；请求方法、路径和字段以
`.api` 契约为准；内部服务调用以 `contracts/proto/` 为准。

## 生成与校验

项目使用的 `goctl` 已内置 Swagger 生成能力，不需要额外安装插件：

```powershell
goctl api validate -api contracts/api/app.api
goctl api swagger `
  --api contracts/api/app.api `
  --dir docs/api `
  --filename openapi
```

修改 `.api` 后必须重新执行以上命令，并将契约与 `openapi.json` 一起提交。当前 Swagger 文件应包含
`BearerAuth`，受保护接口通过 `Authorization: Bearer <access_token>` 调用。

## 本地预览

可以使用官方 Swagger UI Docker 镜像查看和调试接口：

```powershell
$apiDocs = (Resolve-Path .\docs\api).Path
docker run --rm -p 8081:8080 `
  -e SWAGGER_JSON=/spec/openapi.json `
  -v "${apiDocs}:/spec:ro" `
  swaggerapi/swagger-ui:v5.32.11
```

浏览器访问 `http://127.0.0.1:8081`。需要调用管理员接口时，先通过登录接口取得 Access Token，再在
Swagger UI 的 `Authorize` 中填写 `Bearer <access_token>`。

Swagger UI 仅用于本地和受控测试环境，不直接暴露到生产公网。完整手机号、验证码、Refresh Token
和其他敏感字段不得写入示例、截图或提交记录。

## 新增接口的交付顺序

1. 在 `plan/` 明确业务流程、权限、状态变化、数据归属和验收标准；
2. 修改 `contracts/api/*.api`，必要时同步修改 `contracts/proto/` 与事件 Schema；
3. 运行 `goctl api validate`，确认路径、字段和命名无冲突；
4. 生成或更新 Handler、Types 和 RPC 代码；
5. 在 Logic/Manager 中实现用例与权限规则，在 Store/Repository 中实现数据访问；
6. 写入操作同时处理事务、`operation_id`、乐观锁、审计和 Outbox；
7. 补充单元测试、数据库集成测试和 HTTP 全链路测试；
8. 重新生成 `docs/api/openapi.json`，运行 `scripts/check.ps1` 后提交。
