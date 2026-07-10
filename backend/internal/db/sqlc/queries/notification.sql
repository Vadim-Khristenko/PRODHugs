-- name: SaveNotificationRef :exec
INSERT INTO notification_refs (kind, event_id, provider, chat_ref, message_ref)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (kind, event_id, provider)
DO UPDATE SET chat_ref = EXCLUDED.chat_ref, message_ref = EXCLUDED.message_ref, created_at = now();

-- name: GetNotificationRefs :many
SELECT provider, chat_ref, message_ref
FROM notification_refs
WHERE kind = $1 AND event_id = $2;
