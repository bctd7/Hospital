# 线上部署手册

本目录是体验环境后端部署的唯一入口。当前稳定方案是：在 Windows 开发机按 `main` 构建离线 Docker 镜像包，
通过 SSH 上传到 ECS，再由服务器导入镜像并启动 Compose。不要在服务器临时改代码，也不要切换到另一套发布方式。

## 当前拓扑

```text
微信小程序
  -> CloudBase AnyService（hospitalapi）
  -> ECS:8888 / App API
       -> Identity RPC   -> hospital_identity
       -> Appointment RPC -> hospital_appointment
       -> Guidance RPC   -> hospital_guidance
  -> Redis / Kafka
```

一次性任务包括 `identity-migrate`、`appointment-migrate`、`guidance-migrate`、
`identity-bootstrap-admin` 和 `kafka-init`。它们显示 `Exited (0)` 表示执行成功，不是服务崩溃。

## 一、服务器配置

服务器文件：

```text
/opt/hospital/deploy/production/.env.production
```

由 `env.example` 创建，权限保持 `600`。以下配置必须存在：

- Identity、Appointment、Guidance 三个 MySQL DSN；
- Redis、Kafka、JWT、手机号 HMAC 与短信配置；
- 高德 Web 服务 Key；需要模型解析时配置 Guidance LLM；
- 唯一医院根节点；
- 两名真实超级管理员；
- 第三位真实医生体验账号。

```dotenv
IDENTITY_BOOTSTRAP_HOSPITAL_CODE=SH-SECOND-PEOPLES-HOSPITAL
IDENTITY_BOOTSTRAP_HOSPITAL_NAME=上海市第二人民医院
IDENTITY_BOOTSTRAP_ADMIN_PHONES=第一名真实管理员,第二名真实管理员
EXPERIENCE_REAL_DOCTOR_PHONE=第三位真实医生
```

真实手机号只保存在服务器 `.env.production`。数据库保存手机号 HMAC 指纹和脱敏快照。

## 二、本地验证并构建离线镜像

部署前必须保证目标提交已经推送到 `main`，本地 `main` 与工作区干净：

```powershell
Set-Location C:\Users\27902\GolandProjects\Hospital
git switch main
git pull --ff-only origin main
git status --short
.\scripts\quality\verify-repository.ps1
```

构建完整离线镜像包：

```powershell
$archive = Join-Path $env:TEMP 'hospital-images.tar.gz'
if (Test-Path -LiteralPath $archive) { Remove-Item -LiteralPath $archive -Force }
.\deploy\production\scripts\export-images.ps1 -OutputPath $archive
Get-FileHash -Algorithm SHA256 $archive
```

镜像包包含 MySQL、Redis、Kafka、三个迁移任务、管理员初始化、三个 RPC 和 App API，不允许手工漏传 Guidance。

## 三、上传并部署

服务器地址：`121.41.119.163`，部署密钥：`$env:USERPROFILE\.ssh\hospital_ecs`。

```powershell
$archive = Join-Path $env:TEMP 'hospital-images.tar.gz'
scp -i "$env:USERPROFILE\.ssh\hospital_ecs" -o BatchMode=yes `
  $archive root@121.41.119.163:/tmp/hospital-images.tar.gz
```

普通发布保留数据库，先在服务器备份，再导入镜像：

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
./scripts/import-images.sh /tmp/hospital-images.tar.gz
```

本项目尚未上线且明确要求重置全部体验数据时，才执行下面的完整替换。删除目标仅限三个明确命名的卷：

```bash
cd /opt/hospital/deploy/production
docker compose --env-file .env.production -f docker-compose.yml down --remove-orphans
docker volume inspect hospital-production-mysql-data hospital-production-redis-data hospital-production-kafka-data
docker volume rm hospital-production-mysql-data hospital-production-redis-data hospital-production-kafka-data
./scripts/import-images.sh /tmp/hospital-images.tar.gz
```

不得把删卷操作放入日常 `deploy.sh` 或 `import-images.sh`。

## 四、装载综合体验数据

只允许在三个压平迁移和两名超级管理员初始化完成、尚无业务数据的空库执行：

```bash
cd /opt/hospital/deploy/production
set -a
source ./.env.production
set +a
EXPERIENCE_SEED_ACKNOWLEDGE=fresh-experience-database ./scripts/seed-experience.sh
```

脚本会验证：

- 恰好两名超级管理员；
- `EXPERIENCE_REAL_DOCTOR_PHONE` 创建为普通科室医生，且没有超级管理员角色；
- 只有一个上海市第二人民医院院区，地点限 1、2、3 号楼；
- 客户 Excel 的十五类检查项目和补充项目全部存在；
- 周一至周日均有上午/下午窗口；
- 八种预约状态、队列、消息、报告草稿/发布/更正和 Guidance 规则齐全；
- 第三位真实医生同时绑定患者侧历史报告、当前及未来记录。

脚本检测到已有科室或业务数据会拒绝执行，不会覆盖线上数据。

## 五、健康检查

```bash
cd /opt/hospital/deploy/production
curl --fail http://127.0.0.1:8888/api/v1/health
docker compose --env-file .env.production -f docker-compose.yml ps -a
docker compose --env-file .env.production -f docker-compose.yml logs --tail=100 \
  identity-rpc appointment-rpc guidance-rpc app-api kafka
```

常驻服务 `identity-rpc`、`appointment-rpc`、`guidance-rpc` 与 `app-api` 都必须处于运行状态。健康检查通过后，再上传
依赖这些接口的小程序。

## 六、小程序构建与上传

必须从小程序源码目录执行，`--project` 必须指向发布产物，不要在 `dist/build/mp-weixin` 内再次拼接 `dist`：

```powershell
$miniappRoot = 'C:\Users\27902\GolandProjects\Hospital\apps\miniapp'
$wechatCli = 'C:\Program Files (x86)\Tencent\微信web开发者工具\cli.bat'
$releaseVersion = '填写新版本号'
$releaseDescription = '填写面向体验客户的版本说明'

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

终端显示 `√ upload` 后，再到微信公众平台把该版本设为体验版。开发者工具日常联调导入
`apps/miniapp/dist/dev/mp-weixin`；发布上传只使用 `apps/miniapp/dist/build/mp-weixin`。

## 七、备份与恢复

`backup.sh` 备份 `hospital_identity`、`hospital_appointment` 和 `hospital_guidance` 三个数据库：

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
gzip -t backups/hospital-时间戳.sql.gz
```

恢复时先停止四个业务服务，再导入 SQL，最后重新部署：

```bash
docker compose --env-file .env.production -f docker-compose.yml stop \
  app-api identity-rpc appointment-rpc guidance-rpc
gunzip -c /path/to/hospital-时间戳.sql.gz \
  | docker compose --env-file .env.production -f docker-compose.yml exec -T mysql \
    sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot'
./scripts/deploy.sh
```

## 八、禁止事项

- 不提交 `.env.production`、AccessKey、LLM Key、AppSecret、短信密钥或 Token 私钥；
- 不在服务器工作区直接修改代码；
- 不从功能分支直接发布，部署前必须先合并并推送 `main`；
- 未明确要求重置体验环境时，不删除 MySQL、Redis 或 Kafka 卷；
- 不绕过迁移任务手写线上表结构；
- 不在 Guidance 未健康时只凭 App API 健康就宣布部署成功。
