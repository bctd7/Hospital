# Guidance 服务

Guidance 是独立的智能导诊服务，负责检查项目之间的规划规则和地图路线能力，不拥有预约、容量、房间严格地址或
检查状态。完整业务方案见
[Guidance 智能导诊业务逻辑](../../plan/backend/modules/proposals/02-guidance-service.md)。

## 当前已实现

- 工作人员维护检查项目的直接先后关系，并阻止直接或间接循环；
- 查询直接关系以及由直接关系推导出的完整顺序；
- 通过三步流程一次配置项目事实、直接先后关系、自然语言准备说明和患者提醒；
- 把空腹、禁水和饮水说明解析为固定结构化规则，用药、怀孕等内容只形成非强制提醒；
- 向已登录患者提供只返回第三步 `reminders` 的窄读接口，患者报到页据此决定是否显示提醒确认；
- 由 Guidance 协调、Appointment 参与 TCC，确保项目事实与规则创建或更新一次成功、一次取消；
- 患者选择多个项目和本周非连续日期后生成不超过三个可执行预约方案，并展示大窗内的规划参考时刻；
- 患者确认方案时由 Appointment 在单个 MySQL 事务内校验额度、重复占用和共享容量并创建整组预约；
- 批量确认因本周额度不足而失败时，返回独立的 `SMART_APPOINTMENT_WEEKLY_QUOTA_EXHAUSTED` 业务原因和中文提示，不再误报为通用的“预约条件已经变化”；
- 读取患者当天活动与已完成预约，按检查中、正在叫号、候检/待到院、已完成分阶段生成当日顺序；
- 通过高德 POI 2.0 搜索地点；未精确命中完整楼栋名称时，再用官方地理编码补充 GCJ-02 坐标；
- 计算任意 A、B 两点之间的步行、公交或驾车距离、预计耗时、分段指引和折线；
- 起终点非常接近时直接返回本地零距离结果，不请求外部地图；
- App API 向已登录小程序提供地点搜索和步行路线入口；
- 小程序使用微信原生 `<map>` 绘制高德路线结果，高德 Key 不下发前端。

## 服务边界

- Appointment 是项目、房间、窗口、容量、预约和检查状态的唯一事实来源；
- Guidance 通过 Appointment RPC 读取必要项目事实，不跨库查询；
- Appointment 项目创建和更新的公开 HTTP 入口已经收口到 Guidance 完整配置流程，房间关联与窗口仍独立维护；
- 高德只提供地点检索和道路路线，不决定检查顺序或预约是否成功；
- 医院楼栋入口以后由医院自己的数据库保存，高德 POI 只用于辅助定位；
- 路线结果按请求实时返回，当前不落库。

## 当前内部结构

```text
rpc/internal/
├─ appointmentclient/         所有调用 Appointment RPC 的出站适配器
│  ├─ configuration.go        项目查询与配置 TCC Try/Confirm/Cancel
│  ├─ planning.go             窗口、预约查询与原子批量预约
│  └─ precedence.go           先后规则校验所需的只读项目目录
├─ projectconfiguration/      完整三步项目配置
│  ├─ read.go                 配置读取和检查说明解析预览
│  ├─ configure.go            一次配置的主流程
│  ├─ transaction.go          TCC 确认、补偿和 Guidance 本地事务
│  └─ validation.go           输入、权限、项目引用和循环校验
├─ planning/                  患者方案编排
│  ├─ generate.go             智能预约方案生成
│  ├─ search.go               DFS、状态记忆化、共享容量与全局方案比较
│  ├─ timeline.go             精确窗口、预计时长、楼栋移动估算与路线缓存
│  ├─ existing_bookings.go    已有预约对新方案的先后边界
│  ├─ confirm.go              整组预约确认
│  ├─ today.go                当日检查顺序
│  └─ ordering.go             先后关系和说明规则排序
├─ rules/
│  ├─ precedence/             项目之间的直接先后关系、维护和图校验
│  └─ description/            检查说明的解析、结构化转换和患者提醒
├─ routing/                   地点搜索与两点路线的稳定领域契约
│  └─ amap/                   高德地点搜索、地理编码和步行/公交/驾车路线适配
├─ repository/mysqlstore/     Guidance 自有规则持久化
├─ logic/                     gRPC 协议适配与错误映射
└─ svc/                       依赖装配
```

`projectconfiguration`、`planning`、`rules` 和 `routing` 是四个业务边界。`appointmentclient` 不是第五种业务，
它只是把前三个业务需要的 Appointment RPC 调用集中隔离，避免外部服务适配器散落在规则目录或 Manager 文件中。

`rules` 只有两块：`precedence` 表达“哪个项目建议先做”；`description` 把医生填写的项目说明转换为禁食、禁水、
喝水三种封闭状态以及非强制患者提醒。`description` 优先调用兼容 OpenAI Chat Completions 的外部模型生成候选结构，
再执行确定性枚举、范围和冲突校验；模型不可用或结果非法时降级到默认解析，最终都必须由工作人员确认。

模型候选中的规则、提醒和待处理片段在服务端统一归一化为空集合而不是 `null`，小程序 API 边界还会再次防御性
归一化。自然语言说明发生变化后，旧候选不能继续保存，必须重新解析并由工作人员确认。

`planning/search.go` 对项目顺序和 Appointment 可预约选项执行完整 DFS，并以已选集合、最后日期和分钟、准备状态、
地点及争抢容量构成记忆化状态。它不是逐项贪心：共享容量、项目预计时长和楼栋移动都会参与全局比较。每个分支
按可开始的最早时刻排程，完整落在项目窗口与房间窗口交集内才算可执行。跨楼栋步行时间由高德计算并向上取整到
五分钟；结果按楼栋对缓存一天，高德临时失败时按 15 分钟保守估算并短暂缓存。候选日期内已有活动/完成预约作为
先后关系锚点，`no_show` 不满足前置关系；硬约束不会被评分抵消。禁食、禁水和喝水共用同一准备成熟度模型：
未到最短时长不可执行，合理区间内越充分越好，超过最大建议时长仍可兜底但逐渐降级；整组更早完成仍优先于单纯
延长准备时间。

规划结果按患者可见的项目、房间、日期、时段和参考时刻去重。没有可行分配属于成功的空结果，不使用冲突错误；
只有患者确认方案时容量、项目版本或预约事实已经变化，才要求重新计算。

高德公交接口的部分可选字段会以空数组或缺省值返回，适配器统一使用宽容的供应商解析结构，再转换成 Guidance
自己的稳定路线契约，避免把供应商 JSON 差异暴露为小程序 `internal error`。

阅读一条请求时按以下顺序即可：`server → logic → 对应业务包 Manager → repository/appointmentclient`。
`logic` 只是 go-zero 的协议适配层；真正的业务判断位于四个业务包。`svc` 只在启动时组装依赖，不参与业务流程。

## 配置与验证

高德密钥只配置在后端运行环境：

```env
AMAP_WEB_SERVICE_KEY=你的高德Web服务Key
```

本地 `.env` 和生产环境分别填写真实值；`.env.example` 与 `deploy/production/env.example` 只保留变量名或占位值，
不得提交真实 Key。

外部模型调用允许 20 秒，App API 到 Guidance 的调用链预留 30 秒；模型超时或输出不符合封闭规则时，服务会在
同一请求中降级到确定性解析，而不是把 `context deadline exceeded` 直接暴露给前端。

```powershell
.\scripts\database\migrate.ps1 -Service guidance -Direction up
.\scripts\database\seed-comprehensive-test-data.ps1 -Reset
.\scripts\development\start-backend.ps1 -Restart
go test ./service/guidance/...
.\scripts\quality\verify-repository.ps1
```

综合数据中 `13482154556` 复用真实超级管理员账号作为患者侧导诊体验账号，预置今日“血常规 → 冠状动脉 CTA
→ 泌尿系彩超”三条预约。其中冠状动脉 CTA 使用当前可报到短窗口，并配置“请按检查说明和医嘱提前用药”的患者
提醒，可直接验证报到前提醒弹窗；三条预约还可验证先后关系、饮水项目后置和逐站导航。提前规划的完整组合与预期结果见
[`scripts/database/README.md`](../../scripts/database/README.md#智能导诊联调组合)。

## 当前不包含

- 医院楼栋入口资料库和室内地图；第一版根据 Appointment 房间楼栋文字检索高德地点；
- 跨院区智能组合；
- 可靠的现场精确等待时间和自动负载重排；
- 把规划、预约或多站执行状态塞入稳定的两点路线接口；
- 由地图供应商直接决定检查顺序或预约结果。

后续边界和演进条件以
[Guidance 智能导诊业务逻辑](../../plan/backend/modules/proposals/02-guidance-service.md) 为准，不能从外部地图接口或
前端展示直接扩展业务状态。
