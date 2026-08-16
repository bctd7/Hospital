# Guidance 服务

Guidance 是独立的智能导诊服务，负责检查项目之间的规划规则和地图路线能力，不拥有预约、容量、房间严格地址或
检查状态。完整业务方案见
[Guidance 智能导诊业务逻辑](../../plan/backend/modules/proposals/02-guidance-service.md)。

## 当前已实现

- 工作人员维护检查项目的直接先后关系，并阻止直接或间接循环；
- 查询直接关系以及由直接关系推导出的完整顺序；
- 通过高德 POI 2.0 搜索地点；未精确命中完整楼栋名称时，再用官方地理编码补充 GCJ-02 坐标；
- 计算任意 A、B 两点之间的步行距离、预计耗时、分段指引和折线；
- 起终点非常接近时直接返回本地零距离结果，不请求外部地图；
- App API 向已登录小程序提供地点搜索和步行路线入口；
- 小程序使用微信原生 `<map>` 绘制高德路线结果，高德 Key 不下发前端。

当前尚未实现智能预约方案生成、整组原子预约、医院楼栋入口资料和根据 Appointment 实时状态调整下一站。它们是
上述规则与路线能力的后续使用者，不应提前塞入两点路线接口。

## 服务边界

- Appointment 是项目、房间、窗口、容量、预约和检查状态的唯一事实来源；
- Guidance 通过 Appointment RPC 读取必要项目事实，不跨库查询；
- 高德只提供地点检索和道路路线，不决定检查顺序或预约是否成功；
- 医院楼栋入口以后由医院自己的数据库保存，高德 POI 只用于辅助定位；
- 路线结果按请求实时返回，当前不落库。

## 当前内部结构

```text
rpc/internal/
├─ rules/precedence/          检查项目直接先后关系及循环校验
│  ├─ manager/                规则维护、查询和图校验
│  └─ appointmentcatalog/     规则写入时读取 Appointment 项目事实
├─ routing/                   地点搜索与两点路线的稳定领域契约
│  └─ amap/                   高德地点搜索、地理编码和步行路线适配
├─ repository/mysqlstore/     Guidance 自有规则持久化
├─ logic/                     gRPC 协议适配与错误映射
└─ svc/                       依赖装配
```

`routing.Manager` 只编排稳定的地点与路线操作；供应商细节收在 `routing/amap`，不再建立含义宽泛的顶层
`integration`。Appointment 项目目录只服务于先后规则，因此下降到 `rules/precedence/appointmentcatalog`。
不为尚未确认的准备规则、规划器或动态导航提前创建空包。

## 配置与验证

高德密钥只配置在后端运行环境：

```env
AMAP_WEB_SERVICE_KEY=你的高德Web服务Key
```

本地 `.env` 和生产环境分别填写真实值；`.env.example` 与 `deploy/production/env.example` 只保留变量名或占位值，
不得提交真实 Key。

```powershell
.\scripts\migrate.ps1 -Service guidance -Direction up
.\scripts\start-backend.ps1 -Restart
go test ./service/guidance/...
.\scripts\check.ps1
```
