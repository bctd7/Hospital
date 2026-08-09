# Production deployment and server migration

This directory is the repeatable deployment entry point for the public test backend. It runs MySQL, Redis, Identity RPC, and App API on one ECS instance without Kafka.

## What each file does

- `Dockerfile`: builds reproducible Linux images for the two Go services.
- `docker-compose.yml`: starts the complete stack, runs pending Identity migrations, keeps MySQL/Redis private, publishes only the App API, and limits memory usage.
- `config/`: contains production service discovery and logging settings.
- `env.example`: documents required secrets without storing real values in Git.
- `scripts/deploy.sh`: validates, builds, starts, and health-checks an update.
- `scripts/backup.sh`: creates a compressed, transaction-consistent MySQL backup.
- `scripts/export-images.ps1`: pulls official base images and builds a portable image archive on Windows.
- `scripts/import-images.sh`: loads a portable image archive on Ubuntu and deploys without contacting Docker Hub.

The real `deploy/production/.env.production` is server-only and ignored by Git and Docker build context.

## First deployment

Prerequisites: Ubuntu 22.04, Docker Engine with Compose v2, a project-specific SSH key, and an ECS security group that exposes TCP 22 only for administration and TCP 8888 for the CloudBase AnyService source.

```bash
cd /opt/hospital/deploy/production
cp env.example .env.production
# Fill every placeholder with a production value.
chmod 600 .env.production
chmod +x scripts/*.sh
./scripts/deploy.sh
curl --fail http://127.0.0.1:8888/api/v1/health
```

Do not expose MySQL `3306`, Redis `6379`, or Identity RPC `8080` in the ECS security group.

## Uploading a later version

Upload the changed repository files to `/opt/hospital`, then run:

```bash
cd /opt/hospital/deploy/production
./scripts/backup.sh
./scripts/deploy.sh
```

Rebuilding service containers does not delete the named MySQL and Redis volumes. A schema change must be delivered as a reviewed migration under `migrations/`; never use `docker compose down -v` as an update mechanism.

`identity-migrate` builds and runs the repository's `tools/db-migrate` executable, backed by the pinned `golang-migrate` Go dependency. Every deployment runs `up`; already-recorded versions are skipped and only pending migrations execute. A failed migration prevents Identity RPC and App API from being replaced with a partially compatible release. Production migration remains an explicit deployment action after backup; CI validates migrations against an isolated test database and never connects directly to production.

## When Docker Hub is unavailable

Alibaba Cloud's Docker Hub accelerator does not guarantee that every exact image tag is cached. For this test deployment, build and export the five required images on the Windows development machine:

```powershell
.\deploy\production\scripts\export-images.ps1 `
    -OutputPath "$env:TEMP\hospital-images.tar.gz"
scp -i "$env:USERPROFILE\.ssh\hospital_ecs" `
    "$env:TEMP\hospital-images.tar.gz" `
    root@SERVER_IP:/tmp/hospital-images.tar.gz
```

Then import and deploy on Ubuntu without any registry pull:

```bash
cd /opt/hospital/deploy/production
./scripts/import-images.sh /tmp/hospital-images.tar.gz
```

For a long-lived production environment, push these pinned images to a private Alibaba Cloud ACR repository and change the Compose `image` references to ACR. Do not treat the Docker Hub mirror as a reliable release dependency.

Useful checks:

```bash
docker compose --env-file .env.production -f docker-compose.yml ps
docker compose --env-file .env.production -f docker-compose.yml logs --tail=200 app-api identity-rpc
curl --fail http://127.0.0.1:8888/api/v1/health
```

## Moving to another server

1. Keep the old server running and run `scripts/backup.sh`.
2. Create the new Ubuntu server and restrict its security group before deployment.
3. Upload the same repository revision and `.env.production` to `/opt/hospital` on the new server. Keeping the token signing keys avoids invalidating existing access tokens. Rotate infrastructure passwords only after the database restore is verified.
4. Run `scripts/deploy.sh` once so MySQL creates its volumes and service accounts.
5. Stop the application containers on the new server and restore the backup:

   ```bash
   cd /opt/hospital/deploy/production
   gunzip -c /path/to/hospital-TIMESTAMP.sql.gz \
     | docker compose --env-file .env.production -f docker-compose.yml exec -T mysql \
       sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot'
   ./scripts/deploy.sh
   ```

6. Verify the new server locally with `curl http://127.0.0.1:8888/api/v1/health` and exercise login/CRUD through a test account.
7. In CloudBase AnyService, change only the origin public IP from the old ECS address to the new one while keeping the CloudBase environment ID and service name unchanged. In that case the miniapp does not need to be rebuilt or uploaded again.
8. If the miniapp later uses direct HTTPS instead of AnyService, update `VITE_API_BASE_URL`, the WeChat request allowlist, DNS, and the TLS certificate, then rebuild and upload the miniapp.
9. Keep the old server stopped but recoverable until final verification. To roll back, point AnyService back to the old public IP and restart the old stack.

Redis stores refresh-session state. A MySQL-only migration may require users to log in again, which is acceptable for this test environment. If uninterrupted sessions become necessary, add a tested Redis backup/restore procedure before migration.

## Backup handling

Backups contain account and phone-related records. Keep them outside the Git repository, restrict file permissions, copy them to a second protected location, and remove obsolete copies according to the project's retention policy. Test a restore before relying on a backup.

The backup script does not automatically delete old files. This avoids silent data loss, but disk usage must be monitored on the 40 GiB ECS system disk.
