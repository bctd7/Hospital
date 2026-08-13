# Appointment Service

Appointment 服务当前实现两组能力：

- 检查项目目录：一个项目只属于一个科室；
- 科室资源配置：一个房间只属于一个科室，房间与本部门检查项目可多对多关联；
- 独立周配置：房间维护开放时间和共享容量，项目维护预约开始、停止新增和结束时间；
- 严格窗口校验：启用关联后，项目窗口必须完整落在对应房间同星期、同上午/下午窗口内；
- 患者查询：按检查项目返回可选择的有效房间，不做医生绑定或后端自动分配。

## 热点缓存

资源读取使用 Redis Cache Aside，MySQL 始终是最终数据源：

- 房间、房间项目关系、项目可选房间、房间周窗口、项目周窗口缓存 5 分钟并增加 0–20% 随机抖动；
- 患者侧候选查询缓存 30 秒，未找到的单实体缓存 10 秒；
- 同一实例对相同缓存键合并并发回源，降低热点失效时的数据库冲击；
- 列表键包含科室 generation。写事务提交后递增 generation，新请求立即使用新键，无需扫描或批量删除旧键；
- Redis 读写失败自动回源 MySQL；后续预约与容量扣减必须在 MySQL 事务中复核，缓存不能作为容量授权依据。

Redis 键统一使用配置的 `AppointmentRedis.Prefix`，资源模块再追加 `resource:` 命名空间。

## 验证

```powershell
go test ./service/appointment/...
go test ./service/app/api/...
goctl api validate -api contracts/api/app.api
```
