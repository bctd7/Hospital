# 生产部署与服务器迁移

本目录是公网测试后端的可重复部署入口。在一台 ECS 上运行 MySQL、Redis、Identity RPC 和 App API，当前阶段不启用 Kafka。

## 文件作用

- `Dockerfile`：构建可重复生成的 Go 服务 Linux 镜像；
- `docker-compose.yml`：启动完整服务栈、执行待运行的 Identity 迁移、限制资源，并且只向宿主机发布 App API；
- `config/`：保存生产环境的服务发现和日志配置；
- `env.example`：说明必需配置项，不保存真实密钥；
- `scripts/deploy.sh`：校验配置、构建、启动并检查服务健康状态；
- `scripts/backup.sh`：生成事务一致的 MySQL 压缩备份；
- `scripts/export-images.ps1`：在 Windows 开发机拉取基础镜像、构建服务镜像并导出归档；
- `scripts/import-images.sh`：在 Ubuntu 服务器导入镜像归档，无需访问 Docker Hub 即可部署。

真实配置文件 `deploy/production/.env.production` 仅保存在服务器，已经被 Git 和 Docker 构建上下文忽略。

## 首次部署

前置条件：Ubuntu 22.04、Docker Engine、Compose v2、项目专用 SSH 密钥。ECS 安全组只对管理来源开放 TCP 22，并为 CloudBase AnyService 源站开放 TCP 8888。

```bash
cd /opt/hospital/deploy/production
cp env.example .env.production
# 填写所有占位配置
chmod 600 .env.production
chmod +x scripts/*.sh
./scripts/deploy.sh
curl --fail http://127.0.0.1:8888/api/v1/health
```

安全组不要开放 MySQL `3306`、Redis `6379` 或 Identity RPC `8080`。

## 后续更新

将对应版本的仓库文件上传到 `/opt/hospital` 后执行：

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
./scripts/deploy.sh
```

重建服务容器不会删除具名 MySQL、Redis 数据卷。表结构变化必须通过 `migrations/` 下经过评审的新迁移交付，禁止使用 `docker compose down -v` 更新服务。

`identity-migrate` 构建并运行 `tools/db-migrate`，依赖版本由仓库锁定。每次部署执行 `up`：已登记版本会跳过，只运行待执行版本。迁移失败时，Identity RPC 和 App API 不会继续启动为不兼容版本。生产迁移必须在备份后明确执行；CI 只连接隔离测试数据库，不连接生产库。

## Docker Hub 不可用时

阿里云 Docker Hub 加速器不能保证缓存每个精确镜像标签。测试部署可以在 Windows 开发机生成包含全部依赖的镜像归档：

```powershell
.\deploy\production\scripts\export-images.ps1 `
    -OutputPath "$env:TEMP\hospital-images.tar.gz"
scp -i "$env:USERPROFILE\.ssh\hospital_ecs" `
    "$env:TEMP\hospital-images.tar.gz" `
    root@SERVER_IP:/tmp/hospital-images.tar.gz
```

在 Ubuntu 服务器导入并部署：

```bash
cd /opt/hospital/deploy/production
./scripts/import-images.sh /tmp/hospital-images.tar.gz
```

长期运行时应将固定版本镜像推送到阿里云 ACR 私有仓库，并把 Compose 镜像地址改为 ACR；不要把 Docker Hub 镜像加速器当作可靠发布源。

常用检查命令：

```bash
docker compose --env-file .env.production -f docker-compose.yml ps
docker compose --env-file .env.production -f docker-compose.yml logs --tail=200 app-api identity-rpc
curl --fail http://127.0.0.1:8888/api/v1/health
```

## 更换服务器

1. 保持旧服务器运行，执行 `scripts/backup.sh`；
2. 创建新的 Ubuntu 服务器，先收紧安全组；
3. 把同一仓库版本和 `.env.production` 上传到新服务器 `/opt/hospital`。保留 Token 签名密钥可以避免现有 Access Token 立即失效；基础设施密码应在数据库恢复验证后再轮换；
4. 先执行一次 `scripts/deploy.sh`，创建 MySQL 数据卷和服务账号；
5. 停止新服务器的应用容器并恢复备份：

   ```bash
   cd /opt/hospital/deploy/production
   gunzip -c /path/to/hospital-TIMESTAMP.sql.gz \
     | docker compose --env-file .env.production -f docker-compose.yml exec -T mysql \
       sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot'
   ./scripts/deploy.sh
   ```

6. 在新服务器本机调用健康检查，并使用测试账号验证登录和关键接口；
7. 在 CloudBase AnyService 中只修改源站公网 IP，保持云环境 ID 和服务标识不变。此时小程序无需重新构建或上传；
8. 如果未来改为小程序直接请求 HTTPS，则需要更新 `VITE_API_BASE_URL`、微信请求合法域名、DNS 和 TLS 证书，然后重新构建上传；
9. 新服务器验证完成前，旧服务器保持停止但可恢复。需要回滚时，把 AnyService 指回旧 IP 并重新启动旧服务。

Redis 保存 Refresh Session。只迁移 MySQL 可能要求用户重新登录；测试环境可以接受。如果未来要求会话无感迁移，需要先增加并验证 Redis 备份恢复流程。

## 备份管理

备份包含账号和手机号相关记录。备份不得提交 Git，应限制文件权限、复制到第二个受保护位置，并按照保留周期清理。正式依赖备份前必须实际验证恢复流程。

备份脚本不会自动删除旧文件，以避免静默数据丢失；需要持续监控 40 GiB ECS 系统盘用量。
