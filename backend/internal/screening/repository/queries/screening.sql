-- name: GetSettings :one
SELECT * FROM screening.settings WHERE id = 1;

-- name: UpdateSettings :one
UPDATE screening.settings
SET hold_threshold = @hold_threshold, auto_delete_enabled = @auto_delete_enabled,
    auto_delete_threshold = @auto_delete_threshold, outage_policy = @outage_policy, updated_at = now()
WHERE id = 1
RETURNING *;

-- name: SetStandInOffline :exec
UPDATE screening.settings SET stand_in_offline = @offline, updated_at = now() WHERE id = 1;

-- name: ListChannelThresholds :many
SELECT * FROM screening.channel_thresholds;

-- name: UpsertChannelThreshold :exec
INSERT INTO screening.channel_thresholds (channel_id, hold_threshold, auto_delete_threshold)
VALUES (@channel_id, @hold_threshold, @auto_delete_threshold)
ON CONFLICT (channel_id) DO UPDATE
SET hold_threshold = EXCLUDED.hold_threshold, auto_delete_threshold = EXCLUDED.auto_delete_threshold;

-- name: DeleteChannelThreshold :exec
DELETE FROM screening.channel_thresholds WHERE channel_id = @channel_id;

-- name: ListBlocklist :many
SELECT * FROM screening.blocklist ORDER BY term_folded;

-- name: InsertBlock :one
INSERT INTO screening.blocklist (id, term, term_folded, rule, created_by) VALUES (@id, @term, @term_folded, @rule, @created_by)
RETURNING *;

-- name: DeleteBlock :execrows
DELETE FROM screening.blocklist WHERE id = @id;
