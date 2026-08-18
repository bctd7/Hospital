# 质量检查脚本

`verify-repository.ps1` 是提交和部署前的统一检查入口，按顺序验证：

1. Protobuf 与 go-zero HTTP 契约；
2. 事件 JSON Schema 与开发/生产 Compose；
3. 全部 Go 测试和 `go vet`；
4. 小程序测试、类型检查、开发版与发布版构建。

```powershell
.\scripts\quality\verify-repository.ps1
```

脚本失败时立即返回非零退出码，不负责自动修复，也不修改数据库。
