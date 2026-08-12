#!/bin/sh

# The official MySQL entrypoint sources *.sh files in its own shell. Keep
# strict-mode options inside a subshell so they cannot leak into that script.
(
set -eu

: "${MYSQL_ROOT_PASSWORD:?MYSQL_ROOT_PASSWORD is required}"
: "${IDENTITY_MYSQL_USER:?IDENTITY_MYSQL_USER is required}"
: "${IDENTITY_MYSQL_PASSWORD:?IDENTITY_MYSQL_PASSWORD is required}"

MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --protocol=socket -uroot <<EOSQL
CREATE DATABASE IF NOT EXISTS hospital_identity
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS '${IDENTITY_MYSQL_USER}'@'%'
    IDENTIFIED BY '${IDENTITY_MYSQL_PASSWORD}';
GRANT ALL PRIVILEGES ON hospital_identity.* TO '${IDENTITY_MYSQL_USER}'@'%';
FLUSH PRIVILEGES;
EOSQL
)
