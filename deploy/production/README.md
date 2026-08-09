# 线上操作手册

当前调用链：

```text
微信小程序
  → CloudBase AnyService（hospitalapi）
  → ECS:8888
  → App API
  → Identity RPC
  → MySQL / Redis
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
- 不开放 MySQL `3306`、Redis `6379`、Identity RPC `8080`；
- 把同一版本仓库放到 `/opt/hospital`；
- 把旧服务器的 `.env.production` 安全复制到新服务器：

```text
/opt/hospital/deploy/production/.env.production
```

必须保留原来的 Token 公私钥、手机号指纹密钥、微信 AppSecret 和短信配置。否则现有会话可能失效，手机号指纹也无法继续匹配旧账号。

### 3. 首次启动并恢复数据

```bash
cd /opt/hospital/deploy/production
chmod 600 .env.production
chmod +x scripts/*.sh
./scripts/deploy.sh
docker compose --env-file .env.production -f docker-compose.yml stop app-api identity-rpc
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
VITE_STAFF_DATA_SOURCE=mock
```

如果服务标识不是 `hospitalapi`，同时修改 `VITE_ANYSERVICE_NAME`。

### 3. 重新构建并上传

```powershell
cd C:\Users\27902\GolandProjects\Hospital\apps\miniapp
npm run type-check
npm run test
npm run build:mp-weixin
```

在微信开发者工具上传一个新版本，并到微信公众平台把它设为体验版。

### 4. 验证

在开发者工具 Console 调用 `/api/v1/health`，再在手机体验版完成一次短信登录。

ECS、MySQL 和 Redis 没有变化，所以不需要搬数据库。

## 三、加入新代码后如何发布

### 情况 A：只修改小程序代码

```powershell
cd C:\Users\27902\GolandProjects\Hospital\apps\miniapp
npm run type-check
npm run test
npm run build:mp-weixin
```

然后：

```text
微信开发者工具上传新版本
→ 微信公众平台版本管理
→ 设为体验版
```

不需要重启 ECS。

### 情况 B：只修改 Go 后端代码

先在本地检查并提交代码：

```powershell
cd C:\Users\27902\GolandProjects\Hospital
.\scripts\check.ps1
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
docker compose --env-file .env.production -f docker-compose.yml logs --tail=100 app-api identity-rpc
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

不要修改已经执行过的 `000001`、`000002`。新增一组迁移，例如：

```text
migrations/identity/000003_add_xxx.up.sql
migrations/identity/000003_add_xxx.down.sql
```

本地验证：

```powershell
.\scripts\migrate.ps1 -Service identity -Direction up
.\scripts\migrate.ps1 -Service identity -Direction version
.\scripts\check.ps1
```

发布时：

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
./scripts/deploy.sh
```

`identity-migrate` 会跳过旧版本，只执行新的 `000003`。迁移失败时后端不会继续启动。

### 情况 B：增加一个全新的业务数据库

需要同时完成以下内容：

1. 新建迁移目录：

   ```text
   migrations/新服务名/
   ```

2. 在 `.env.example` 和 `deploy/production/env.example` 增加数据库账号、密码和 DSN；
3. 更新 `deploy/compose/mysql/init/001-create-service-databases.sh`，为全新数据卷创建数据库和账号；
4. 更新 `scripts/migrate.ps1`，让 `-Service` 支持新服务；
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
docker compose --env-file .env.production -f docker-compose.yml logs --tail=100 app-api identity-rpc
```

然后在手机体验版验证：启动、短信登录、退出登录和关键页面。

## 六、禁止事项

- 不提交 `.env.production`、AppSecret、AccessKey、Token 私钥；
- 不执行 `docker compose down -v`，它会删除数据库和 Redis 数据卷；
- 不改写已经在共享环境执行过的迁移；
- 不在没有备份的情况下执行生产数据库迁移；
- 不在新服务器验证完成前释放旧服务器。
