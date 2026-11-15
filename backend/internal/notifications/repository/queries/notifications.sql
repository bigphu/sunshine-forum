-- dedupe_key: event giao lại không tạo thông báo trùng
-- name: InsertItem :execrows
INSERT INTO notifications.items (id, account_id, kind, dedupe_key, body, link)
VALUES (@id, @account_id, @kind, @dedupe_key, @body, @link)
ON CONFLICT (account_id, dedupe_key) DO NOTHING;

-- name: ListItems :many
SELECT * FROM notifications.items WHERE account_id = @account_id ORDER BY created_at DESC LIMIT @lim;

-- name: CountUnread :one
SELECT count(*) FROM notifications.items WHERE account_id = @account_id AND read_at IS NULL;

-- name: MarkAllRead :exec
UPDATE notifications.items SET read_at = now() WHERE account_id = @account_id AND read_at IS NULL;

-- name: PurgeItems :execrows
DELETE FROM notifications.items WHERE created_at < now() - make_interval(days => @days::int);

