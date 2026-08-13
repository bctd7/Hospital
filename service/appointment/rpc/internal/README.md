# Appointment RPC 内部结构

请求链路：

```text
server -> logic -> StaffManager / PatientManager -> mysqlstore
```

`manager/` 是唯一业务边界，不再按 `catalog`、`resource` 或其他对象建立并列业务包。

- `StaffManager`：管理员和授权科室工作人员的入口，负责项目、房间、房间项目关系及周时间配置；未来检查单也归入这个入口。
- `PatientManager`：患者入口，负责项目可选房间和可预约时间查询；未来预约、取消和容量扣减也归入这个入口。
- `staff_project.go`：员工端检查项目操作。
- `staff_room.go`：员工端房间操作。
- `staff_project_room.go`：员工端房间与检查项目关系操作。
- `staff_schedule.go`：员工端房间开放时间与项目预约时间操作。
- `patient_project.go`：患者端按项目查询可选房间和预约时间。
- `types.go`、`store.go`：两个 Manager 共同使用的数据和持久化接口，不是第三套业务入口。
- `repository/mysqlstore/`：只实现 MySQL 查询、事务锁、幂等记录与审计，不编排业务。
- `logic/`：只负责 Proto 转换、登录主体读取和 gRPC 错误映射。
- `svc/`：组装数据库、鉴权 Redis、业务 Redis 以及两个 Manager。

所有写操作以 MySQL 事务结果为准。Redis 仅缓存热点读取副本，通过科室 generation 失效，不参与授权、预约容量扣减或防超卖判断。
