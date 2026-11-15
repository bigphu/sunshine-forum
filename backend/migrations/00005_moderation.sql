-- Module moderation: hàng chờ (case), quyết định, yêu cầu xem xét lại và nhật ký
-- kiểm toán. Bảng chỉ giữ ID và kết quả sàng lọc; nội dung bài nằm ở module content.

-- +goose Up
CREATE SCHEMA moderation;

-- Một case mở cho mỗi nội dung đang chờ duyệt; content_type + content_id trỏ tới content.*
CREATE TABLE moderation.cases (
    id           uuid PRIMARY KEY,
    content_type text NOT NULL CHECK (content_type IN ('post', 'comment')),
    content_id   uuid NOT NULL,
    author_id    uuid NOT NULL,
    channel_id   uuid NOT NULL,
    status       text NOT NULL CHECK (status IN ('open', 'decided', 'withdrawn')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    closed_at    timestamptz
);
CREATE UNIQUE INDEX cases_open_content_idx ON moderation.cases (content_type, content_id) WHERE status = 'open';
CREATE INDEX cases_queue_idx ON moderation.cases (created_at) WHERE status = 'open';

CREATE TABLE moderation.decisions (
    id              uuid PRIMARY KEY,
    case_id         uuid REFERENCES moderation.cases (id),
    content_type    text NOT NULL CHECK (content_type IN ('post', 'comment')),
    content_id      uuid NOT NULL,
    author_id       uuid NOT NULL,
    channel_id      uuid NOT NULL,
    action          text NOT NULL CHECK (action IN ('approve', 'reject', 'remove', 'auto_delete')),
    rule            int CHECK (rule BETWEEN 1 AND 7),
    -- NULL: bộ lọc tự động xoá
    moderator_id    uuid,
    ai_label        text,
    ai_score        real,
    model_version   text,
    -- Quyết định đảo ngược kết luận của bộ lọc (FR-5.3)
    overturned_ai   boolean NOT NULL DEFAULT false,
    appeal_deadline timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX decisions_author_idx ON moderation.decisions (author_id, created_at DESC);
CREATE INDEX decisions_created_idx ON moderation.decisions (created_at);
-- Tự xoá chỉ một lần cho mỗi nội dung (subscriber có thể chạy lại)
CREATE UNIQUE INDEX decisions_auto_delete_idx ON moderation.decisions (content_type, content_id) WHERE action = 'auto_delete';

-- Mỗi quyết định chỉ được yêu cầu xem xét lại một lần (FR-4.4)
CREATE TABLE moderation.appeals (
    id          uuid PRIMARY KEY,
    decision_id uuid NOT NULL UNIQUE REFERENCES moderation.decisions (id),
    author_id   uuid NOT NULL,
    reason      text NOT NULL,
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'declined')),
    reviewer_id uuid,
    created_at  timestamptz NOT NULL DEFAULT now(),
    decided_at  timestamptz
);
CREATE INDEX appeals_pending_idx ON moderation.appeals (created_at) WHERE status = 'pending';

-- Nhật ký chỉ ghi thêm: ghi trong cùng transaction với hành động, không có nội dung bài,
-- người dùng chỉ được ghi bằng mã ẩn danh (subject). Không ai sửa hay xoá được, kể cả Admin.
CREATE TABLE moderation.audit_log (
    id            bigserial PRIMARY KEY,
    action        text NOT NULL,
    moderator_id  uuid,
    subject       text NOT NULL,
    content_type  text,
    ai_label      text,
    ai_score      real,
    model_version text,
    rule          int,
    outcome       text NOT NULL,
    overturned_ai boolean NOT NULL DEFAULT false,
    -- Dòng ghi từ event (đổi cài đặt): event giao lại không ghi trùng
    event_id      uuid UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_created_idx ON moderation.audit_log (created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION moderation.audit_log_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'moderation.audit_log is append-only';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER audit_log_no_change BEFORE UPDATE OR DELETE OR TRUNCATE ON moderation.audit_log
    FOR EACH STATEMENT EXECUTE FUNCTION moderation.audit_log_immutable();

INSERT INTO identity.permissions (code, description) VALUES
    ('moderation.appeal.create', 'Yêu cầu xem xét lại quyết định về nội dung của mình'),
    ('moderation.review', 'Xem hàng chờ, duyệt hoặc từ chối nội dung bị giữ lại'),
    ('moderation.remove', 'Gỡ nội dung đã đăng'),
    ('moderation.appeal.review', 'Xem xét yêu cầu xem xét lại'),
    ('moderation.audit.read', 'Xem nhật ký kiểm toán');

INSERT INTO identity.role_permissions (role_id, permission)
SELECT r.id, p.code FROM identity.roles r, identity.permissions p
WHERE p.code = 'moderation.appeal.create'
   OR (p.code IN ('moderation.review', 'moderation.remove', 'moderation.appeal.review') AND r.code IN ('manager', 'admin'))
   OR (p.code = 'moderation.audit.read' AND r.code = 'admin');

-- +goose Down
DELETE FROM identity.permissions WHERE code LIKE 'moderation.%';
DROP SCHEMA moderation CASCADE;
