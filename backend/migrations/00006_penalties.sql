-- Module penalties: vi phạm đã xác nhận và tạm chỉ đọc (FR-1.3). Chỉ quyết định từ chối,
-- gỡ và tự xoá tạo vi phạm; xem xét lại thành công thì thu hồi.

-- +goose Up
CREATE SCHEMA penalties;

CREATE TABLE penalties.settings (
    id                 int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    strike_limit       int NOT NULL DEFAULT 3 CHECK (strike_limit > 0),
    strike_window_days int NOT NULL DEFAULT 30 CHECK (strike_window_days > 0),
    timeout_days       int NOT NULL DEFAULT 3 CHECK (timeout_days > 0),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
INSERT INTO penalties.settings (id) VALUES (1);

-- decision_id duy nhất: subscriber chạy lại không tạo vi phạm trùng
CREATE TABLE penalties.strikes (
    id          uuid PRIMARY KEY,
    account_id  uuid NOT NULL,
    decision_id uuid NOT NULL UNIQUE,
    rule        int NOT NULL,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX strikes_account_idx ON penalties.strikes (account_id, created_at) WHERE revoked_at IS NULL;

CREATE TABLE penalties.timeouts (
    id          uuid PRIMARY KEY,
    account_id  uuid NOT NULL,
    -- Vi phạm làm vượt giới hạn; mỗi vi phạm tạo tối đa một lần tạm chỉ đọc
    decision_id uuid NOT NULL UNIQUE,
    last_rule   int NOT NULL,
    starts_at   timestamptz NOT NULL,
    ends_at     timestamptz NOT NULL,
    lifted_at   timestamptz
);
CREATE INDEX timeouts_account_idx ON penalties.timeouts (account_id, ends_at DESC);

INSERT INTO identity.permissions (code, description) VALUES
    ('penalties.settings.manage', 'Chỉnh giới hạn vi phạm và thời hạn tạm chỉ đọc');

INSERT INTO identity.role_permissions (role_id, permission)
SELECT r.id, 'penalties.settings.manage' FROM identity.roles r WHERE r.code = 'admin';

-- +goose Down
DELETE FROM identity.permissions WHERE code LIKE 'penalties.%';
DROP SCHEMA penalties CASCADE;
