# 线上操作手册

当前调用链：

```text
微信小程序
  → CloudBase AnyService（hospitalapi）
  → ECS:8888
  → App API
  ├→ Identity RPC
  └→ Appointment RPC
  → 各自的 MySQL / Redis
  → Outbox → Kafka → Redis 授权版本同步
```

先判断你要做哪件事：

| 场景 | 需要重新部署后端 | 需要重新上传小程序 |
|---|---:|---:|
| 只换 ECS 服务器，CloudBase 环境和服务标识不变 | 是 | 否 |
| 换 CloudBase 环境或 AnyService 服务标识 | 否 | 是 |
| 修改 Go 后端代码 | 是 | 否 |
| 修改小程序代码 | 否 | 是 |
| 增加表或修改表结构 | 是 | 通常否 |
| 增加一个全新的业务数据库 | 是 | 视前端接口是否变化而定 |

## 零、从空数据卷发布当前体验版

当前版本前后端、数据库结构和 Identity 异步授权同步都发生了变化。首次使用空数据卷时，需要完整部署应用栈，
但“完整部署”不等于重装服务器或手写 SQL：

```text
构建最新镜像
  -> MySQL 创建空数据库
  -> identity-migrate 与 appointment-migrate 分别执行压平后的 000001 后退出
  -> identity-bootstrap-admin 创建医院根节点和多个超级管理员后退出
  -> Kafka 启动，kafka-init 创建授权事件 Topic 后退出
  -> Redis、identity-rpc、appointment-rpc、app-api 启动
  -> 后端健康检查通过
  -> 构建并上传微信体验版
```

`identity-migrate`、`identity-bootstrap-admin` 和 `kafka-init` 是一次性任务，显示 `Exited (0)` 表示成功，
不是服务崩溃。
以后再次执行 `deploy.sh` 时，迁移只运行新版本，管理员初始化按手机号指纹幂等跳过已有账号。
当前 Compose 使用单节点 Kafka（副本数 1），适合体验环境，不是高可用生产集群。

如果服务器以前运行过旧体验版，而本次已经明确决定不要旧数据，先核对并删除**这三个指定卷**：

```bash
cd /opt/hospital/deploy/production
docker compose --env-file .env.production -f docker-compose.yml down
docker volume inspect hospital-production-mysql-data hospital-production-redis-data hospital-production-kafka-data
docker volume rm hospital-production-mysql-data hospital-production-redis-data hospital-production-kafka-data
```

这一步会永久删除旧账号、组织、审计、会话和未处理消息，只用于本次“全新体验环境”初始化；若卷本来不存在，
直接跳过。不要把删卷命令加入日常发布脚本。

### 1. 配置两个体验管理员

复制 `env.example` 为服务器上的 `.env.production`，至少替换以下内容：

```dotenv
IDENTITY_BOOTSTRAP_HOSPITAL_CODE=HOSPITAL
IDENTITY_BOOTSTRAP_HOSPITAL_NAME=体验医院名称
IDENTITY_BOOTSTRAP_ADMIN_PHONES=第一个管理员手机号,第二个管理员手机号
```

手机号原文只保存在服务器 `.env.production`，不得提交 Git。初始化任务为每个号码直接创建可短信登录的
`staff` 账号，授予 `super_admin`，并写入授权审计和 Outbox；数据库只保存 HMAC 指纹和脱敏手机号。
除唯一医院根节点和这组管理员外，不创建院区、科室、医生或演示业务数据。

以后需要追加初始化管理员时，在服务器修改手机号列表后单独运行：

```bash
docker compose --env-file .env.production -f docker-compose.yml run --rm identity-bootstrap-admin
```

已有管理员会幂等跳过，新手机号会新增管理员；从列表删除手机号不会自动撤销已有管理员。

### 2. 部署后端

```bash
cd /opt/hospital/deploy/production
chmod 600 .env.production
chmod +x scripts/*.sh
./scripts/deploy.sh
```

检查一次性任务和常驻服务：

```bash
docker compose --env-file .env.production -f docker-compose.yml ps -a
docker compose --env-file .env.production -f docker-compose.yml logs identity-migrate identity-bootstrap-admin
curl --fail http://127.0.0.1:8888/api/v1/health
```

两个管理员手机号属于真实用户账号，随后使用各自手机号接收真实短信验证码登录。需要完整体验数据时执行下方受保护的体验数据脚本；
不执行脚本时，院区、科室和医生需通过 Identity 管理接口建立。

体验版需要预置完整测试链路时，只能在刚完成迁移和两个管理员初始化、尚无业务数据的数据库上执行：

```bash
cd /opt/hospital/deploy/production
EXPERIENCE_SEED_ACKNOWLEDGE=fresh-experience-database ./scripts/seed-experience.sh
```

该脚本保留 `.env.production` 中的两个真实超级管理员，另外创建两名医生和三名虚假患者，并将
自动识别配置中唯一一位 153 开头管理员，将其既有 `account_id` 复用为一名报告测试患者；不会创建同手机号的第二个账号，也不会
修改管理员的账号类型、角色、手机号或已有个人资料。脚本还会创建两个院区、四个科室、
六个房间、十个检查项目，以及待检查、检查中、已完成、已取消、未到场、报告草稿、正式报告和更正版本数据。
消息场景同时覆盖预约成功、两次到院提醒、预约取消、未到场、报告到期、报告超时、报告发布和报告更正。
“当日急诊 CT”会按照执行脚本时的上午或下午自动配置当前有效窗口，并把最晚到院时间放在当前时间附近，
便于体验到院提醒和开始检查。
检测到已有科室或 Appointment 业务数据时脚本会拒绝执行，不会覆盖现有数据。

### 3. 上传体验版

后端健康检查和两个管理员登录均成功后，再在开发机通过终端完成构建和上传：

```powershell
$miniappRoot = 'C:\Users\27902\GolandProjects\Hospital\apps\miniapp'
$wechatCli = 'C:\Program Files (x86)\Tencent\微信web开发者工具\cli.bat'
$releaseVersion = '0.3.4'
$releaseDescription = '增加检查报到、候检叫号与未到场展示，完善检查报告流程。'

Set-Location $miniappRoot
npm run test
npm run type-check
npm run build:mp-weixin

& $wechatCli upload `
  --project (Join-Path $miniappRoot 'dist\build\mp-weixin') `
  --version $releaseVersion `
  --desc $releaseDescription `
  --lang zh
```

不要从 `apps/miniapp` 项目根目录或 `dist/dev/mp-weixin` 上传；两者固定用于直连本机 `127.0.0.1` 的开发调试。
构建前关闭正在打开 `dist/build/mp-weixin` 的开发者工具项目，避免重建产物时出现目录删除提示。终端输出
`√ upload` 后，再到微信公众平台将该版本设为体验版。不要先上传依赖尚未部署接口的小程序。

## 一、更换 ECS 服务器

目标：把数据库、配置和服务搬到新服务器，最后只修改 AnyService 源站 IP。

### 1. 在旧服务器备份

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
```

确认输出的 `.sql.gz` 存在，并执行：

```bash
gzip -t backups/hospital-时间戳.sql.gz
```

### 2. 准备新服务器

- 安装 Ubuntu 22.04、Docker Engine 和 Compose v2；
- 安全组开放 SSH `22` 和后端源站 `8888`；
- 不开放 MySQL `3306`、Redis `6379`、Identity RPC `8080` 和 Appointment RPC `8081`；
- 把同一版本仓库放到 `/opt/hospital`；
- 把旧服务器的 `.env.production` 安全复制到新服务器：

```text
/opt/hospital/deploy/production/.env.production
```

必须保留原来的 Token 公私钥、手机号指纹密钥和短信配置。否则现有会话可能失效，手机号指纹也无法继续匹配旧账号。

### 3. 首次启动并恢复数据

```bash
cd /opt/hospital/deploy/production
chmod 600 .env.production
chmod +x scripts/*.sh
./scripts/deploy.sh
docker compose --env-file .env.production -f docker-compose.yml stop app-api identity-rpc appointment-rpc
```

把备份上传到新服务器后恢复：

```bash
gunzip -c /path/to/hospital-时间戳.sql.gz \
  | docker compose --env-file .env.production -f docker-compose.yml exec -T mysql \
    sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot'

./scripts/deploy.sh
```

### 4. 验证新服务器

```bash
curl --fail http://127.0.0.1:8888/api/v1/health
docker compose --env-file .env.production -f docker-compose.yml ps
```

再测试一次真实短信登录。

### 5. 切换 AnyService

进入 CloudBase → AnyService → `hospitalapi`，只把源站从旧 IP 改为：

```text
新服务器公网IP:8888
```

CloudBase 环境 ID 和服务标识没有变化，因此小程序不需要重新构建或上传。

### 6. 回滚

新服务器验证完成前不要释放旧服务器。回滚时把 AnyService 源站 IP 改回旧服务器，并重新启动旧服务器服务。

## 二、更换 CloudBase 或 AnyService

这里的“更换 CloudBase”包括：换云开发环境、重新创建 AnyService、修改 AnyService 服务标识。

### 1. 创建新服务

在新的 CloudBase 环境中创建 AnyService：

```text
服务标识：hospitalapi
源站协议：HTTP
源站地址：ECS公网IP:8888
```

把新环境绑定到小程序 AppID：

```text
wx8ba66b98c93423fd
```

### 2. 修改小程序本地生产配置

编辑不会提交 Git 的文件：

```text
apps/miniapp/.env.production
```

```dotenv
VITE_API_TRANSPORT=cloudbase
VITE_CLOUDBASE_ENV_ID=新的云开发环境ID
VITE_ANYSERVICE_NAME=hospitalapi
```

如果服务标识不是 `hospitalapi`，同时修改 `VITE_ANYSERVICE_NAME`。

### 3. 重新构建并上传

```powershell
$miniappRoot = 'C:\Users\27902\GolandProjects\Hospital\apps\miniapp'
$wechatCli = 'C:\Program Files (x86)\Tencent\微信web开发者工具\cli.bat'
$releaseVersion = '填写新版本号'
$releaseDescription = '填写面向体验客户的版本说明'

Set-Location $miniappRoot
npm run type-check
npm run test
npm run build:mp-weixin

& $wechatCli upload `
  --project (Join-Path $miniappRoot 'dist\build\mp-weixin') `
  --version $releaseVersion `
  --desc $releaseDescription `
  --lang zh
```

终端输出 `√ upload` 后，到微信公众平台把新版本设为体验版。

### 4. 验证

在开发者工具 Console 调用 `/api/v1/health`，再在手机体验版完成一次短信登录。

ECS、MySQL、Redis 和 Kafka 没有变化，所以不需要搬数据库或消息数据。

## 三、加入新代码后如何发布

### 情况 A：只修改小程序代码

```powershell
$miniappRoot = 'C:\Users\27902\GolandProjects\Hospital\apps\miniapp'
$wechatCli = 'C:\Program Files (x86)\Tencent\微信web开发者工具\cli.bat'
$releaseVersion = '填写新版本号'
$releaseDescription = '填写面向体验客户的版本说明'

Set-Location $miniappRoot
npm run type-check
npm run test
npm run build:mp-weixin

& $wechatCli upload `
  --project (Join-Path $miniappRoot 'dist\build\mp-weixin') `
  --version $releaseVersion `
  --desc $releaseDescription `
  --lang zh
```

然后：

```text
确认终端输出 √ upload
→ 微信公众平台版本管理
→ 设为体验版
```

不需要重启 ECS。

### 情况 B：只修改 Go 后端代码

先在本地检查并提交代码：

```powershell
cd C:\Users\27902\GolandProjects\Hospital
.\scripts\quality\verify-repository.ps1
```

让服务器获取同一个提交后执行：

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
./scripts/deploy.sh
```

验证：

```bash
curl --fail http://127.0.0.1:8888/api/v1/health
docker compose --env-file .env.production -f docker-compose.yml logs --tail=100 app-api identity-rpc appointment-rpc
```

后端接口兼容时，小程序不需要重新上传。

### 情况 C：服务器无法从 Docker Hub 拉镜像

在 Windows 开发机导出镜像：

```powershell
.\deploy\production\scripts\export-images.ps1 `
  -OutputPath "$env:TEMP\hospital-images.tar.gz"

scp -i "$env:USERPROFILE\.ssh\hospital_ecs" `
  "$env:TEMP\hospital-images.tar.gz" `
  root@服务器IP:/tmp/hospital-images.tar.gz
```

服务器先备份，再导入并部署：

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
./scripts/import-images.sh /tmp/hospital-images.tar.gz
```

### 情况 D：前后端都修改

先发布后端并验证健康，再上传小程序新版本。不要先上传依赖新接口的小程序。

## 四、加入新数据库或修改表结构

### 情况 A：只给现有 `hospital_identity` 增加表或字段

不要修改已经执行过的 `000001` 至 `000005`。新增一组迁移，例如：

```text
migrations/identity/000006_add_xxx.up.sql
migrations/identity/000006_add_xxx.down.sql
```

本地验证：

```powershell
.\scripts\database\migrate.ps1 -Service identity -Direction up
.\scripts\database\migrate.ps1 -Service identity -Direction version
.\scripts\quality\verify-repository.ps1
```

发布时：

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
./scripts/deploy.sh
```

`identity-migrate` 会跳过已执行版本，只执行新的 `000006`。迁移失败时后端不会继续启动。

### 情况 B：增加一个全新的业务数据库

需要同时完成以下内容：

1. 新建迁移目录：

   ```text
   migrations/新服务名/
   ```

2. 在 `.env.example` 和 `deploy/production/env.example` 增加数据库账号、密码和 DSN；
3. 更新 `deploy/compose/mysql/init/001-create-service-databases.sh`，为全新数据卷创建数据库和账号；
4. 更新 `scripts/database/migrate.ps1`，让 `-Service` 支持新服务；
5. 在 `deploy/production/docker-compose.yml` 增加对应的迁移任务，并让业务服务依赖迁移成功；
6. 更新 `.github/workflows/ci.yml`，在隔离数据库中测试新迁移；
7. 更新 `scripts/backup.sh`，把新数据库加入备份列表；
8. 在业务服务配置中注入新 DSN；
9. 先在空数据库完成一次 `up → down → up` 验证，再部署。

注意：`docker-entrypoint-initdb.d` 只在 MySQL 数据卷第一次初始化时运行。已有服务器增加新数据库时，不能只修改初始化脚本；还必须在部署前对现有 MySQL 实例执行一次经过评审的幂等数据库/账号创建脚本。

## 五、任何操作完成后都检查

```bash
curl --fail http://127.0.0.1:8888/api/v1/health
docker compose --env-file .env.production -f docker-compose.yml ps
docker compose --env-file .env.production -f docker-compose.yml logs --tail=100 app-api identity-rpc appointment-rpc kafka
```

然后在手机体验版验证：启动、短信登录、退出登录和关键页面。

## 六、禁止事项

- 不提交 `.env.production`、AccessKey、Token 私钥；
- 除“零、从空数据卷发布当前体验版”中已经明确确认的数据重置外，不删除 MySQL、Redis 和 Kafka 数据卷；
- 不改写已经在共享环境执行过的迁移；
- 不在没有备份的情况下执行生产数据库迁移；
- 不在新服务器验证完成前释放旧服务器。
