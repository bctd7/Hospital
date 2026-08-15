# Appointment RPC 内部结构

请求链路：

```text
server -> logic -> SharedManager / PatientManager / StaffManager -> mysqlstore
```

`manager/` 是 Appointment 的业务边界。第一层按“谁在使用能力”划分，而不是按数据库表或模糊的 `catalog`、`resource` 概念划分：

```text
manager/
├─ shared/                 患者与工作人员共同使用的能力
├─ patient/                患者独有的预约生命周期
├─ staff/                  工作人员独有的管理和现场操作
│  ├─ project.go           项目基本信息与项目预约时间
│  ├─ room.go              房间基本信息与房间开放时间、容量
│  ├─ project_room.go      房间可执行项目关系
│  ├─ booking.go           管理删除及配置变化对预约的影响
│  ├─ examination.go       在项目时间内开始检查
│  ├─ report.go            报告草稿、完成发布、正式更正和工作人员读取
│  ├─ message.go           当前科室消息与逐账号已读
│  ├─ input/               业务方法输入及输入整体规则
│  └─ support/             校验、授权、窗口约束、幂等和缓存辅助
└─ common/                 无角色、无业务动作的基础契约
```

- `manager/shared/`：患者与工作人员共同使用的业务能力。当前包括项目列表、工作人员项目详情、患者可选房间和项目开放窗口。方法名明确标出适用角色，权限规则仍然分别校验。
- `manager/patient/`：患者独有的预约生命周期，包括预约操作及本人已发布报告读取。
- `manager/staff/`：管理员和授权科室工作人员独有的管理及现场操作。根层按项目、房间、房间—项目关系、预约、检查执行和报告拆分文件；房间开放时间归房间，项目预约时间归项目。
- `manager/staff/input/`：只定义各业务方法允许接收的字段，以及“至少修改一个字段”这类输入整体规则。按 `project`、`room`、`booking`、`operation` 分文件，避免输入结构继续堆在业务包根层。它不是持久化模型，也不执行事务。
- `manager/staff/support/`：只提供纯字段校验、权限范围判断、窗口包含约束、幂等和缓存等辅助能力；不编排项目、房间或预约业务，也没有 Manager。
- `manager/common/`：无角色、无业务动作的基础契约，包括领域数据、统一错误、缓存实现和事务内原子操作；这里不存在业务 Manager。
- `repository/mysqlstore/`：只实现 MySQL 查询、事务锁、幂等记录与审计，不编排业务。
- `logic/`：只负责 Proto 转换、登录主体读取和 gRPC 错误映射。
- `svc/`：组装数据库、鉴权 Redis、业务 Redis 以及三个业务 Manager。

每个业务包在包内声明自己真正需要的窄 Store 接口。`mysqlstore.Store` 可以同时实现多个接口，但业务代码不会因此得到与当前职责无关的数据库能力。

`staff/support` 可以接收调用方提供的最小接口完成辅助工作，但不能持有业务状态，也不能决定项目、房间或预约最终变成什么状态。跨房间与项目的 `ConfigurationTxStore` 只是 MySQL 事务内的原子操作集合，不代表新的业务方向。

所有写操作以 MySQL 事务结果为准。Redis 仅缓存热点读取副本，通过科室 generation 失效，不参与授权、预约容量扣减或防超卖判断。
