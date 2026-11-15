-- name: InsertExport :one
INSERT INTO administration.exports (id, created_by, item_count, purpose, data, period_from, period_to, delete_after)
VALUES (@id, @created_by, @item_count, @purpose, @data, @period_from, @period_to, @delete_after)
RETURNING id, created_by, item_count, purpose, period_from, period_to, delete_after, created_at;

-- Danh sách không kèm dữ liệu
-- name: ListExports :many
SELECT id, created_by, item_count, purpose, period_from, period_to, delete_after, created_at
FROM administration.exports ORDER BY created_at DESC;

-- name: GetExportData :one
SELECT data, created_at FROM administration.exports WHERE id = @id;

-- name: DeleteExport :execrows
DELETE FROM administration.exports WHERE id = @id;

-- Bản xuất bị xoá trước khi bản gốc tới hạn 90 ngày (US-6.4, NFR-3.3)
-- name: PurgeExports :execrows
DELETE FROM administration.exports WHERE delete_after < now();
