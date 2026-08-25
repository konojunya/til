-- name: SaveOutboxEvent :exec
INSERT INTO outbox_events (
    id,
    event_type,
    payload
) VALUES (
    sqlc.arg(id),
    sqlc.arg(event_type),
    sqlc.arg(payload)::jsonb
);