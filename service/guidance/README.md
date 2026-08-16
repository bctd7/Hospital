# Guidance 服务

Guidance 是独立的智能导诊服务，负责检查项目之间的规划规则和地图路线能力，不拥有预约、容量、房间严格地址或
检查状态。完整业务方案见
[Guidance 智能导诊业务逻辑](../../plan/backend/modules/proposals/02-guidance-service.md)。

## 当前已实现

- 工作人员维护检查项目的直接先后关系，并阻止直接或间接循环；
- 查询直接关系以及由直接关系推导出的完整顺序；
- 通过三步流程一次配置项目事实、直接先后关系、自然语言准备说明和患者提醒；
- 把空腹、禁水和饮水说明解析为固定结构化规则，用药、怀孕等内容只形成非强制提醒；
- 由 Guidance 协调、Appointment 参与 TCC，确保项目事实与规则创建或更新一次成功、一次取消；
- 患者选择多个项目和本周非连续日期后生成不超过三个宏观预约方案；
- 患者确认方案时由 Appointment 在单个 MySQL 事务内校验额度、重复占用和共享容量并创建整组预约；
- 读取患者当天活动与已完成预约，按检查中、正在叫号、候检/待到院、已完成分阶段生成当日顺序；
- 通过高德 POI 2.0 搜索地点；未精确命中完整楼栋名称时，再用官方地理编码补充 GCJ-02 坐标；
- 计算任意 A、B 两点之间的步行距离、预计耗时、分段指引和折线；
- 起终点非常接近时直接返回本地零距离结果，不请求外部地图；
- App API 向已登录小程序提供地点搜索和步行路线入口；
- 小程序使用微信原生 `<map>` 绘制高德路线结果，高德 Key 不下发前端。

医院楼栋入口资料库、跨院区组合和可靠现场负载重排尚未实现。第一版使用 Appointment 房间楼栋文字解析高德地点，
不把规划、预约或多站状态塞进稳定的两点路线接口。

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
├─ projectconfiguration/      完整项目配置、版本校验及 TCC 协调
├─ planning/                  智能预约方案、整组确认和当日顺序
├─ rules/
│  ├─ precedence/             检查项目直接先后关系及循环校验
│  │  ├─ manager/             规则维护、查询和图校验
│  │  └─ appointmentcatalog/  规则写入时读取 Appointment 项目事实
│  └─ preparation/            固定准备状态、提醒和自然语言解析
├─ routing/                   地点搜索与两点路线的稳定领域契约
│  └─ amap/                   高德地点搜索、地理编码和步行路线适配
├─ repository/mysqlstore/     Guidance 自有规则持久化
├─ logic/                     gRPC 协议适配与错误映射
└─ svc/                       依赖装配
```

`projectconfiguration`、`planning`、`rules` 和 `routing` 分别表达完整配置、方案编排、规则事实与地图能力；
`routing.Manager` 只编排稳定的地点和路线操作，供应商细节收在 `routing/amap`，不建立含义宽泛的顶层
`integration`。所有目录都对应已实现业务，不保留空包。

## 配置与验证

高德密钥只配置在后端运行环境：

```env
AMAP_WEB_SERVICE_KEY=你的高德Web服务Key
```

本地 `.env` 和生产环境分别填写真实值；`.env.example` 与 `deploy/production/env.example` 只保留变量名或占位值，
不得提交真实 Key。

```powershell
.\scripts\migrate.ps1 -Service guidance -Direction up
.\scripts\seed-comprehensive-test-data.ps1 -Reset
.\scripts\start-backend.ps1 -Restart
go test ./service/guidance/...
.\scripts\check.ps1
```
