#!/usr/bin/env bash
# TTGQLDL development database bootstrap — runs once, on an empty data volume.
#
# Creates two least-privileged login roles inside the bootstrap database:
#   ttg_owner  owns schema `ttg` and is used by migrations (W0103)
#   ttg_app    may only read and write rows in schema `ttg`; it cannot create,
#              alter or drop schema objects and is never a superuser
#
# The bootstrap superuser created by the official image is development-only and
# is never the role the backend connects with.
set -euo pipefail

: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${TTG_OWNER_PASSWORD:?TTG_OWNER_PASSWORD is required}"
: "${TTG_APP_PASSWORD:?TTG_APP_PASSWORD is required}"

for value in "$TTG_OWNER_PASSWORD" "$TTG_APP_PASSWORD"; do
    case "$value" in
        *"'"*)
            echo "bootstrap: passwords must not contain single quotes" >&2
            exit 1
            ;;
        "")
            echo "bootstrap: passwords must not be empty" >&2
            exit 1
            ;;
    esac
done

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<EOSQL
-- Nothing is public by default.
REVOKE ALL ON DATABASE "$POSTGRES_DB" FROM PUBLIC;

CREATE ROLE ttg_owner LOGIN PASSWORD '$TTG_OWNER_PASSWORD' NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE ROLE ttg_app   LOGIN PASSWORD '$TTG_APP_PASSWORD'   NOSUPERUSER NOCREATEDB NOCREATEROLE;

CREATE SCHEMA ttg AUTHORIZATION ttg_owner;
REVOKE ALL ON SCHEMA ttg FROM PUBLIC;

GRANT CONNECT ON DATABASE "$POSTGRES_DB" TO ttg_owner, ttg_app;
-- The owner role is the only role allowed to create schema objects.
GRANT CREATE ON DATABASE "$POSTGRES_DB" TO ttg_owner;
GRANT USAGE ON SCHEMA ttg TO ttg_owner, ttg_app;

-- Tables created by the owner inherit row-level privileges for the app role.
ALTER DEFAULT PRIVILEGES FOR ROLE ttg_owner IN SCHEMA ttg
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ttg_app;
ALTER DEFAULT PRIVILEGES FOR ROLE ttg_owner IN SCHEMA ttg
    GRANT USAGE, SELECT ON SEQUENCES TO ttg_app;

-- Never search the public schema with an application role.
ALTER ROLE ttg_owner IN DATABASE "$POSTGRES_DB" SET search_path = ttg;
ALTER ROLE ttg_app   IN DATABASE "$POSTGRES_DB" SET search_path = ttg;
EOSQL

echo "bootstrap: created roles ttg_owner and ttg_app with schema ttg"
