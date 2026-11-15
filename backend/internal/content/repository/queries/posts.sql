-- name: InsertPost :one
INSERT INTO content.posts (
    id, channel_id, author_id, official, title, body, search_text, status,
    screen_reason, ai_label, ai_score, model_version, flag_rule, ai_sentences, needs_rescan
) VALUES (
    @id, @channel_id, @author_id, @official, @title, @body, @search_text, @status,
    @screen_reason, @ai_label, @ai_score, @model_version, @flag_rule, @ai_sentences, @needs_rescan
)
RETURNING *;

-- name: GetPost :one
SELECT * FROM content.posts WHERE id = @id;

-- name: GetPostForUpdate :one
SELECT * FROM content.posts WHERE id = @id FOR UPDATE;

-- name: GetPostsByIDs :many
SELECT * FROM content.posts WHERE id = ANY(@ids::uuid[]);

-- Sửa nội dung: quét lại và có thể đổi trạng thái (US-3.3)
-- name: UpdatePostContent :one
UPDATE content.posts
SET title = @title, body = @body, search_text = @search_text, status = @status,
    screen_reason = @screen_reason, ai_label = @ai_label, ai_score = @ai_score, model_version = @model_version,
    flag_rule = @flag_rule, ai_sentences = @ai_sentences, needs_rescan = @needs_rescan,
    edited = true, updated_at = now(),
    status_changed_at = CASE WHEN status = @status THEN status_changed_at ELSE now() END
WHERE id = @id
RETURNING *;

-- Kết quả quét lại sau sự cố; giữ cờ edited
-- name: UpdatePostScreening :one
UPDATE content.posts
SET status = @status, screen_reason = @screen_reason, ai_label = @ai_label, ai_score = @ai_score,
    model_version = @model_version, flag_rule = @flag_rule, ai_sentences = @ai_sentences, needs_rescan = false,
    status_changed_at = CASE WHEN status = @status THEN status_changed_at ELSE now() END
WHERE id = @id AND needs_rescan
RETURNING *;

-- Chỉ đổi khi trạng thái hiện tại nằm trong @from: subscriber chạy lại không đổi sai
-- name: SetPostStatus :one
UPDATE content.posts SET status = @status, status_changed_at = now(), needs_rescan = false
WHERE id = @id AND status = ANY(@from_statuses::text[])
RETURNING *;

-- name: SetPostPinned :execrows
UPDATE content.posts SET pinned = @pinned WHERE id = @id AND status = 'published';

-- Bảng tin: bài đã đăng, không ghim, phân trang theo (created_at, id) giảm dần
-- name: ListFeed :many
SELECT * FROM content.posts
WHERE status = 'published' AND NOT pinned
  AND (sqlc.narg('channel_id')::uuid IS NULL OR channel_id = sqlc.narg('channel_id'))
  AND (sqlc.narg('before_at')::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg('before_at')::timestamptz, sqlc.narg('before_id')::uuid))
ORDER BY created_at DESC, id DESC
LIMIT @lim;

-- name: ListPinned :many
SELECT * FROM content.posts
WHERE status = 'published' AND pinned
  AND (sqlc.narg('channel_id')::uuid IS NULL OR channel_id = sqlc.narg('channel_id'))
ORDER BY created_at DESC;

-- Tìm theo tiêu đề và nội dung đã bỏ dấu (gõ không dấu vẫn khớp)
-- name: SearchPosts :many
SELECT * FROM content.posts
WHERE status = 'published' AND search_text LIKE '%' || @pattern::text || '%'
ORDER BY created_at DESC, id DESC
LIMIT @lim;

-- name: RecountComments :exec
UPDATE content.posts SET
    comment_count = (SELECT count(*) FROM content.comments c WHERE c.post_id = @id AND c.status = 'published'),
    official_replied = EXISTS (SELECT 1 FROM content.comments c WHERE c.post_id = @id AND c.status = 'published' AND c.official)
WHERE id = @id;

-- name: ListPostsToRescan :many
SELECT * FROM content.posts WHERE needs_rescan ORDER BY created_at LIMIT @lim;

-- name: ListPendingPostsByAuthor :many
SELECT * FROM content.posts WHERE author_id = @author_id AND status = 'pending' ORDER BY created_at DESC;

-- name: CountSubmittedSince :one
SELECT ((SELECT count(*) FROM content.posts p WHERE p.created_at >= @since::timestamptz)
     + (SELECT count(*) FROM content.comments c WHERE c.created_at >= @since::timestamptz))::bigint AS total;

-- Bị bộ lọc giữ lại: khớp blocklist, khác ngôn ngữ, sự cố, hoặc mô hình gắn nhãn vi phạm,
-- bất kể quyết định sau đó
-- name: FlaggedByDay :many
SELECT to_char(created_at AT TIME ZONE 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD')::text AS day, count(*) AS n
FROM (
    SELECT created_at, screen_reason, ai_label FROM content.posts WHERE created_at >= @since::timestamptz
    UNION ALL
    SELECT created_at, screen_reason, ai_label FROM content.comments WHERE created_at >= @since::timestamptz
) x
WHERE screen_reason IN ('blocklist', 'language', 'outage') OR ai_label IN ('offensive', 'hate')
GROUP BY 1 ORDER BY 1;

-- Nội dung đã xoá, bị từ chối hay bị gỡ quá @days ngày thì xoá hẳn (NFR-3.3)
-- name: PurgePosts :execrows
DELETE FROM content.posts
WHERE status IN ('deleted', 'rejected', 'removed', 'auto_deleted')
  AND status_changed_at < now() - make_interval(days => @days::int);

-- Số nội dung bị gắn cờ theo nhãn, cho trang Phân tích
-- name: FlaggedByLabel :one
SELECT
    count(*) FILTER (WHERE ai_label = 'hate')            AS hate,
    count(*) FILTER (WHERE ai_label = 'offensive')       AS offensive,
    count(*) FILTER (WHERE screen_reason = 'blocklist')  AS blocklist,
    count(*) FILTER (WHERE screen_reason = 'language')   AS language
FROM (
    SELECT screen_reason, ai_label FROM content.posts WHERE created_at >= @since::timestamptz
    UNION ALL
    SELECT screen_reason, ai_label FROM content.comments WHERE created_at >= @since::timestamptz
) x;
