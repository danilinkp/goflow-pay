-- +goose Up
ALTER TABLE outbox ADD COLUMN traceparent TEXT NOT NULL DEFAULT '';
ALTER TABLE outbox ADD COLUMN tracestate TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE outbox DROP COLUMN tracestate;
ALTER TABLE outbox DROP COLUMN traceparent;
