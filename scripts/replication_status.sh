#!/usr/bin/env bash
set -euo pipefail

for db in auth accounts transactions; do
    echo "== postgres-$db -> postgres-$db-replica"
    docker compose exec -T "postgres-$db" psql -U postgres -P pager=off -c "
        SELECT client_addr, state, sync_state,
               pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)) AS behind,
               write_lag, flush_lag, replay_lag
        FROM pg_stat_replication"
    docker compose exec -T "postgres-$db-replica" psql -U postgres -tA -c "
        SELECT 'replica: in_recovery=' || pg_is_in_recovery()
            || ', apply_delay=' || current_setting('recovery_min_apply_delay')
            || ', last_replay=' || coalesce(pg_last_xact_replay_timestamp()::text, '-')"
    echo
done
