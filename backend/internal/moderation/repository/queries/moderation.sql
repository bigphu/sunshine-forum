-- name: OpenCase :exec
INSERT INTO moderation.cases (id, content_type, content_id, author_id, channel_id, status)
VALUES (@id, @content_type, @content_id, @author_id, @channel_id, 'open')
ON CONFLICT (content_type, content_id) WHERE status = 'open' DO NOTHING;

-- name: CloseOpenCase :execrows
UPDATE moderation.cases SET status = @status, closed_at = now()
WHERE content_type = @content_type AND content_id = @content_id AND status = 'open';

-- name: LockOpenCase :one
SELECT * FROM moderation.cases
WHERE content_type = @content_type AND content_id = @content_id AND status = 'open'
FOR UPDATE;

-- name: ListOpenCases :many
SELECT * FROM moderation.cases WHERE status = 'open' ORDER BY created_at;

-- name: InsertDecision :one
INSERT INTO moderation.decisions (
    id, case_id, content_type, content_id, author_id, channel_id, action, rule, moderator_id,
    ai_label, ai_score, model_version, overturned_ai, appeal_deadline
) VALUES (
    @id, @case_id, @content_type, @content_id, @author_id, @channel_id, @action, @rule, @moderator_id,
    @ai_label, @ai_score, @model_version, @overturned_ai, @appeal_deadline
)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: GetDecision :one
SELECT * FROM moderation.decisions WHERE id = @id;

-- name: GetDecisionsByIDs :many
SELECT * FROM moderation.decisions WHERE id = ANY(@ids::uuid[]);

-- Quyết định của một cư dân (trừ duyệt), kèm yêu cầu xem xét lại nếu có
-- name: ListDecisionsByAuthor :many
SELECT d.*, a.id AS appeal_id, a.status AS appeal_status, a.reason AS appeal_reason,
       a.created_at AS appeal_created_at, a.decided_at AS appeal_decided_at
FROM moderation.decisions d
LEFT JOIN moderation.appeals a ON a.decision_id = d.id
WHERE d.author_id = @author_id AND d.action <> 'approve'
ORDER BY d.created_at DESC;

-- name: InsertAppeal :one
INSERT INTO moderation.appeals (id, decision_id, author_id, reason) VALUES (@id, @decision_id, @author_id, @reason)
RETURNING *;

-- name: LockAppeal :one
SELECT * FROM moderation.appeals WHERE id = @id FOR UPDATE;

-- name: DecideAppeal :exec
UPDATE moderation.appeals SET status = @status, reviewer_id = @reviewer_id, decided_at = now() WHERE id = @id;

-- name: ListPendingAppeals :many
SELECT * FROM moderation.appeals WHERE status = 'pending' ORDER BY created_at;

-- name: InsertAudit :exec
INSERT INTO moderation.audit_log (action, moderator_id, subject, content_type, ai_label, ai_score, model_version, rule, outcome, overturned_ai)
VALUES (@action, @moderator_id, @subject, @content_type, @ai_label, @ai_score, @model_version, @rule, @outcome, @overturned_ai);

-- name: InsertAuditFromEvent :exec
INSERT INTO moderation.audit_log (action, moderator_id, subject, outcome, event_id)
VALUES (@action, @moderator_id, @subject, @outcome, @event_id)
ON CONFLICT (event_id) DO NOTHING;

-- name: ListAudit :many
SELECT * FROM moderation.audit_log
WHERE (sqlc.narg('before_id')::bigint IS NULL OR id < sqlc.narg('before_id'))
  AND (NOT @overturned_only::boolean OR overturned_ai)
  AND (NOT @config_only::boolean OR action = 'config_change')
ORDER BY id DESC
LIMIT @lim;

-- name: DecisionStats :one
SELECT
    count(*) FILTER (WHERE action = 'approve')     AS approved,
    count(*) FILTER (WHERE action = 'reject')      AS rejected,
    count(*) FILTER (WHERE action = 'remove')      AS removed,
    count(*) FILTER (WHERE action = 'auto_delete') AS auto_deleted,
    count(*) FILTER (WHERE overturned_ai)          AS overturned_ai,
    count(*) FILTER (WHERE action = 'approve' AND overturned_ai) AS held_wrong,
    count(*) FILTER (WHERE action = 'remove' AND overturned_ai)  AS missed,
    count(*) FILTER (WHERE action = 'reject' AND ai_label IN ('offensive', 'hate')) AS agreed
FROM moderation.decisions WHERE created_at >= @since;

-- name: AppealStats :one
SELECT
    count(*) FILTER (WHERE ap.status = 'pending')  AS pending,
    count(*) FILTER (WHERE ap.status = 'accepted') AS accepted,
    count(*) FILTER (WHERE ap.status = 'declined') AS declined,
    count(*) FILTER (WHERE ap.status = 'accepted' AND EXISTS (
        SELECT 1 FROM moderation.decisions d WHERE d.id = ap.decision_id AND d.action = 'auto_delete')) AS deleted_wrong
FROM moderation.appeals ap WHERE ap.created_at >= @since;

-- Sửa đổi đã chốt cho bản xuất huấn luyện: có nhãn AI, qua hạn xem xét lại, và không bị
-- xem xét lại thành công sau đó (xem xét lại thành công đã có quyết định ngược lại ở audit)
-- name: ListCorrections :many
SELECT d.*, a.status AS appeal_status FROM moderation.decisions d
LEFT JOIN moderation.appeals a ON a.decision_id = d.id
WHERE d.ai_label IS NOT NULL
  AND d.created_at BETWEEN @period_from AND @period_to
  AND (d.appeal_deadline IS NULL OR d.appeal_deadline < now())
  AND (a.id IS NULL OR a.status <> 'pending')
ORDER BY d.created_at;
