\set ON_ERROR_STOP on

SELECT format('CREATE ROLE %I REPLICATION LOGIN', :'repl_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'repl_user')
\gexec

ALTER ROLE :"repl_user" WITH REPLICATION LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    PASSWORD :'repl_password';

SELECT pg_create_physical_replication_slot(:'repl_slot')
WHERE NOT EXISTS (SELECT 1 FROM pg_replication_slots WHERE slot_name = :'repl_slot');
