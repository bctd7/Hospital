# 后端交付与质量基线

## 契约和生成物

- HTTP 源文件：`contracts/api/app.api` 及其 import；
- gRPC 源文件：`contracts/proto/*/v1/*.proto`；
- 事件源文件：`contracts/events/`；
- 数据库源文件：`migrations/<service>/`。

仓库暂不提交生成式 OpenAPI 文档。字段和路径直接查看 `.api` 契约；重新引入可浏览接口文档时，应通过 CI
从契约生成，不能维护第二份手写接口事实。

## 写操作

写操作按业务风险选择并明确：

- `operation_id` 幂等；
- `expected_version` 乐观锁；
- 主数据、审计和 Outbox 的同事务提交；
- 状态机前置条件与权限范围；
- 缓存写后失效；
- 对配置变化影响既有预约的事务处理。

## Migration

Identity 与 Appointment 在首次体验环境发布前都已压平为单个 `000001` 初始迁移。正式上线以后不得改写已执行迁移，
必须新增版本。空库升级、必要的降级、应用启动兼容性和测试数据脚本需要一起验证。

## 测试层次

1. 纯规则单元测试；
2. Manager 与 Store 的数据库/Redis 集成测试；
3. RPC 与 App API 协议适配测试；
4. 小程序单元、TypeScript 和构建检查；
5. 对高风险主流程进行受控端到端验证。

统一入口：

```powershell
.\scripts\quality\verify-repository.ps1
```

## 完成标准

- 代码、契约、Migration 和中文文档一致；
- 不存在旧路由、旧 Mock、重复 Manager 或失效链接；
- 并发、幂等、权限和隐私测试覆盖关键失败路径；
- 本地开发版与发布版小程序产物分别生成；
- 新配置同步进入示例配置和部署说明，但不提交真实 Secret。
