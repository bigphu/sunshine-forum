-- name: InsertComment :one
INSERT INTO content.comments (
    id, post_id, parent_id, author_id, official, body, status,
    screen_reason, ai_label, ai_score, model_version, flag_rule, ai_sentences, needs_rescan
) VALUES (
    @id, @post_id, @parent_id, @author_id, @official, @body, @status,
    @screen_reason, @ai_label, @ai_score, @model_version, @flag_rule, @ai_sentences, @needs_rescan
)
RETURNING *;

-- name: GetComment :one
SELECT * FROM content.comments WHERE id = @id;

-- name: GetCommentForUpdate :one
SELECT * FROM content.comments WHERE id = @id FOR UPDATE;

-- name: GetCommentsByIDs :many
SELECT * FROM content.comments WHERE id = ANY(@ids::uuid[]);

-- name: UpdateCommentContent :one
UPDATE content.comments
SET body = @body, status = @status, screen_reason = @screen_reason, ai_label = @ai_label, ai_score = @ai_score,
    model_version = @model_version, flag_rule = @flag_rule, ai_sentences = @ai_sentences,
    needs_rescan = @needs_rescan, edited = true, updated_at = now(),
    status_changed_at = CASE WHEN status = @status THEN status_changed_at ELSE now() END
WHERE id = @id
RETURNING *;

-- name: UpdateCommentScreening :one
UPDATE content.comments
SET status = @status, screen_reason = @screen_reason, ai_label = @ai_label, ai_score = @ai_score,
    model_version = @model_version, flag_rule = @flag_rule, ai_sentences = @ai_sentences, needs_rescan = false,
    status_changed_at = CASE WHEN status = @status THEN status_changed_at ELSE now() END
WHERE id = @id AND needs_rescan
RETURNING *;

-- name: SetCommentStatus :one
UPDATE content.comments SET status = @status, status_changed_at = now(), needs_rescan = false
WHERE id = @id AND status = ANY(@from_statuses::text[])
RETURNING *;

-- Cả chủ đề trong một truy vấn (bình luận chỉ hai cấp)
-- name: ListThreadComments :many
SELECT * FROM content.comments WHERE post_id = @post_id ORDER BY created_at, id;

-- name: ListCommentsToRescan :many
SELECT * FROM content.comments WHERE needs_rescan ORDER BY created_at LIMIT @lim;

-- name: ListPendingCommentsByAuthor :many
SELECT * FROM content.comments WHERE author_id = @author_id AND status = 'pending' ORDER BY created_at DESC;

-- Bình luận bị xoá quá hạn còn trả lời: xoá chữ, giữ dòng làm chỗ cho các trả lời (NFR-3.3)
-- name: BlankPurgeableComments :execrows
UPDATE content.comments c SET body = ''
WHERE c.status IN ('deleted', 'rejected', 'removed', 'auto_deleted') AND c.body <> ''
  AND c.status_changed_at < now() - make_interval(days => @days::int)
  AND EXISTS (SELECT 1 FROM content.comments r WHERE r.parent_id = c.id);

-- name: PurgeComments :execrows
DELETE FROM content.comments c
WHERE c.status IN ('deleted', 'rejected', 'removed', 'auto_deleted')
  AND c.status_changed_at < now() - make_interval(days => @days::int)
  AND NOT EXISTS (SELECT 1 FROM content.comments r WHERE r.parent_id = c.id);
