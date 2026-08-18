# 契约脚本

本目录只负责从 `contracts/` 的源契约生成或验证代码，不承载业务逻辑。

```powershell
# 只验证全部 Proto 的导入和语法，不修改文件
.\scripts\contracts\generate-protobuf.ps1 -Check

# 重新生成 contracts/gen 下的 Go 类型与 gRPC Client/Server 接口
.\scripts\contracts\generate-protobuf.ps1
```

生成后必须提交源 Proto 和生成的 Go 文件，并运行 `go test ./...`。HTTP 契约仍由 `goctl api validate` 校验；
不要从 Proto 自动推导小程序 HTTP 模型。
