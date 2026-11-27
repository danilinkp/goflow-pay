#!/bin/sh
set -eu

if [ ! -s "$PGDATA/PG_VERSION" ]; then
    mkdir -p "$PGDATA"
    chown postgres:postgres "$PGDATA"
    chmod 700 "$PGDATA"

    until gosu postgres pg_basebackup \
        -h "$PRIMARY_HOST" -U "$REPLICATION_USER" -D "$PGDATA" \
        -S "$REPLICATION_SLOT" -X stream -R -w; do
        echo "pg_basebackup from $PRIMARY_HOST failed, retrying"
        find "$PGDATA" -mindepth 1 -delete
        sleep 2
    done
fi

exec docker-entrypoint.sh postgres \
    -c hba_file=/etc/postgresql/pg_hba.conf \
    -c recovery_min_apply_delay="${REPLICA_APPLY_DELAY:-0}"
