# 数据库迁移与 CI 实施计划

> 状态：待审核，尚未实施
>
> 范围：数据库版本管理、本地自动化脚本、GitHub 持续集成；不包含业务建模和生产部署

## 1. 目标

随着 Patient、Appointment、Planning 和 Navigation 逐步拥有独立逻辑数据库，需要用统一方式完成：

1. 本地开发环境创建服务数据库和最小权限账号；
2. 按版本创建表、索引、约束和基础系统数据；
3. 已存在的数据库只执行尚未运行的迁移；
4. 开发者用一条命令完成后端、前端和契约检查；
5. GitHub 在每次推送和 Pull Request 时自动验证代码与迁移；
6. 任何检查失败时返回非零退出码，阻止错误被当作成功。

数据库迁移不是每次删库重建。它类似数据库结构的 Git 历史，每个环境记录自己已经执行到的版本。

## 2. 当前问题

当前 Compose 通过 `/docker-entrypoint-initdb.d` 在全新 MySQL 数据卷中执行 Identity SQL。这种方式只适合
首次初始化：已有数据卷不会因为新增迁移文件而再次执行初始化目录。

如果继续使用当前方式，后续可能出现：

- 开发数据库已经执行 `000003`，测试数据库仍停留在 `000002`；
- 新增服务数据库后，旧数据卷中没有对应数据库和账号；
- SQL 执行失败后，不清楚数据库处于哪个版本；
- 开发者需要手工复制 SQL，容易漏执行或执行顺序错误；
- GitHub 只能检查代码，不能证明迁移可以在空库中成功运行。

## 3. 核心决定

### 3.1 迁移工具

计划采用 `golang-migrate` 管理 MySQL 表结构版本。实施时固定工具版本，不使用未经确认的自动升级。

选择原因：

- 支持 `*.up.sql` / `*.down.sql` 现有命名方式；
- 不要求业务服务引入 ORM；
- 可在本地脚本和 GitHub Actions 中使用同一套命令；
- 每个逻辑数据库独立保存迁移版本。

### 3.2 建库与建表分离

自动化分成两个权限边界：

```text
本地数据库引导
  -> 创建逻辑数据库和服务账号
  -> 需要本地 root 权限

数据库迁移
  -> 使用各服务自己的数据库账号
  -> 创建或升级该服务拥有的表
  -> 不持有其他数据库权限
```

正式环境的数据库和账号由部署流程或数据库管理员创建，普通迁移脚本不得依赖生产 root 密码。

### 3.3 数据库归属

首期映射如下：

| 服务/模块 | 逻辑数据库 | 迁移目录 | 连接环境变量 |
|---|---|---|---|
| Identity | `hospital_identity` | `migrations/identity` | `IDENTITY_MYSQL_DSN` |
| Patient | `hospital_patient` | `migrations/patient` | `PATIENT_MYSQL_DSN` |
| Appointment | `hospital_appointment` | `migrations/appointment` | `APPOINTMENT_MYSQL_DSN` |
| Planning | `hospital_planning` | `migrations/planning` | `PLANNING_MYSQL_DSN` |
| Navigation | `hospital_navigation` | `migrations/navigation` | `NAVIGATION_MYSQL_DSN` |

未开始实现的服务不提前创建空迁移目录和空数据库；进入对应实施阶段时再加入映射。

## 4. 计划目录

```text
scripts/
├── check.ps1                 # 本地统一质量检查
├── db-bootstrap-local.ps1    # 仅用于本地创建逻辑数据库和服务账号
└── migrate.ps1               # 按服务执行 up/down/version

.github/workflows/
└── ci.yml                    # GitHub 自动检查

migrations/
├── identity/
├── patient/                  # Patient 实施时创建
├── appointment/              # Appointment 实施时创建
├── planning/                 # Planning 实施时创建
└── navigation/               # Navigation 实施时创建
```

现有 `scripts/check-foundation.ps1` 的有效检查会合并到 `scripts/check.ps1`，避免两个“全量检查”脚本
逐渐产生不同结果。是否保留旧文件作为兼容入口，在实施时根据调用情况决定。

## 5. 脚本设计

### 5.1 `db-bootstrap-local.ps1`

职责仅限本地环境：

- 检查 Docker 和 MySQL 是否可用；
- 幂等创建已经进入实施阶段的逻辑数据库；
- 幂等创建对应服务账号并授予本库权限；
- 不删除数据库，不覆盖已有数据；
- 非 `local` 环境立即拒绝执行。

示例：

```powershell
.\scripts\db-bootstrap-local.ps1
```

Compose 中的 MySQL 初始化脚本可以继续负责全新数据卷的首次引导，但本地引导脚本必须能够处理
“数据卷已经存在、后来新增一个服务数据库”的情况。

### 5.2 `migrate.ps1`

计划接口：

```powershell
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service identity -Direction version
.\scripts\migrate.ps1 -Service identity -Direction down -Steps 1
```

约束：

- `Service` 必须来自显式白名单，不能把任意路径或任意 DSN 拼接进命令；
- 默认只执行 `up`，回滚必须显式传入方向和步数；
- 连接信息从环境变量读取，命令输出不得打印密码；
- 找不到迁移目录、DSN 或迁移工具时立即失败；
- 检测到 dirty migration 时明确报错，不自动强制修改版本；
- 生产环境回滚不由普通开发脚本自动执行。

### 5.3 `check.ps1`

统一执行：

```text
go test ./...
go vet ./...
goctl api validate -api contracts/api/app.api
事件 JSON Schema 解析
docker compose config --quiet
npm ci（仅 CI；本地已有依赖时不重复安装）
npm test -- --run
npm run type-check
npm run build:mp-weixin
```

脚本应在任意步骤失败后停止并返回非零退出码，同时使用 `try/finally` 恢复调用者原始目录。

## 6. 迁移文件规则

每次数据库变化新增一对文件：

```text
000003_identity_organization.up.sql
000003_identity_organization.down.sql
```

规则：

1. 编号在同一服务目录内严格递增；
2. 已经进入共享分支或环境执行的迁移不得改写，通过新迁移修正；
3. `up` 必须能从上一个正式版本升级；
4. `down` 只回退当前迁移，不回退更早版本；
5. 不在迁移中写入环境相关的医院、科室或测试账号数据；
6. 稳定的系统角色和权限可以作为系统基础数据，但业务样例使用独立 seed/import 工具；
7. 破坏性字段变更采用“新增字段 -> 双写/回填 -> 切换读取 -> 后续删除”的分阶段方式；
8. 表和索引名称必须明确归属，禁止跨服务外键和跨库触发器。

## 7. CI 设计

GitHub Actions 在推送到 `main`、`plan` 以及 Pull Request 时运行。`plan` 只有 Markdown 变化时可跳过
耗时构建，但工作流文件本身变化必须执行完整检查。

首版包含三个 Job：

### 7.1 Backend

- 安装项目指定 Go 版本；
- 缓存 Go module；
- 执行 `go test ./...`；
- 执行 `go vet ./...`；
- 校验 go-zero API 契约。

### 7.2 Miniapp

- 安装项目指定 Node.js 版本；
- 使用 `npm ci` 严格按 lockfile 安装；
- 执行单元测试、类型检查和微信小程序生产构建；
- 不上传 `dist` 到 Git。

### 7.3 Migration smoke test

- 启动临时 MySQL 8.4 Service Container；
- 创建 `hospital_identity` 测试数据库和最小权限账号；
- 从空库执行 Identity 全部 `up`；
- 检查迁移版本为最新版本；
- 回退最近一版后重新执行 `up`；
- 任务结束后销毁临时数据库。

后续每增加一个真实业务数据库，就把它加入同一套矩阵测试，不复制多份工作流。

CI 不连接开发或生产数据库，不读取仓库中的真实 Secret，也不自动部署服务。

## 8. 实施顺序

审核通过后，该能力属于新的工程建设，按照分支规则从最新 `main` 创建新分支，例如：

```text
chore/database-migration-ci
```

实施步骤：

1. 固定 `golang-migrate` 的安装方式和版本；
2. 实现 `migrate.ps1`，先只接入 Identity；
3. 调整 Compose，使首次建库与迁移执行职责清晰，避免同一 SQL 被两套机制重复执行；
4. 实现本地数据库引导脚本；
5. 将 `check-foundation.ps1` 的能力整合进统一检查脚本；
6. 创建 GitHub Actions 后端与前端 Job；
7. 增加 MySQL migration smoke test；
8. 在全新数据卷和已有 Identity 数据卷上分别验证；
9. 更新实际使用说明和 `.env.example`；
10. 提交功能分支，CI 通过后再合并 `main`。

## 9. 验收标准

- 一条本地命令能够把空的 `hospital_identity` 升级到最新结构；
- 重复执行 `up` 不重复建表，也不破坏数据；
- 已执行前两版迁移的数据库能够只执行后续新增迁移；
- 可以查看当前迁移版本；
- 最近一版迁移可以在测试数据库完成 `down -> up`；
- 脚本不在输出中泄露 DSN 密码；
- 不使用 root 账号执行普通表迁移；
- `check.ps1` 覆盖后端、契约、Compose 和小程序检查；
- GitHub 推送后自动运行 Backend、Miniapp 和 Migration 检查；
- 任一迁移 SQL、测试、类型检查或构建失败时 CI 为失败状态；
- CI 不依赖开发者电脑上已经存在的数据库或 `node_modules`。

## 10. 暂不包含

- 自动部署生产环境；
- 自动备份或恢复生产数据库；
- 自动执行不可逆生产迁移；
- Kubernetes、云数据库或 Secret Manager 集成；
- Kafka 集成测试；
- Patient、Appointment 等尚未实施服务的空数据库；
- 医院、院区、部门和检查项目等业务数据导入。

这些内容分别在实际部署或业务模块进入实施时补充，不能与首版迁移和 CI 基线捆绑上线。
