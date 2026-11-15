-- Module administration: thống kê (đọc qua contract của module khác) và bản xuất dữ liệu
-- huấn luyện tối thiểu (US-6.4). Bản xuất không gắn với người dùng hay bài viết, và bị
-- xoá trước khi bản gốc tới hạn 90 ngày.

-- +goose Up
CREATE SCHEMA administration;

CREATE TABLE administration.exports (
    id           uuid PRIMARY KEY,
    created_by   uuid NOT NULL,
    item_count   int NOT NULL,
    -- training | evaluation
    purpose      text NOT NULL DEFAULT 'training' CHECK (purpose IN ('training', 'evaluation')),
    -- JSON Lines: text, label, rule, ai_score, model_version
    data         bytea NOT NULL,
    period_from  timestamptz NOT NULL,
    period_to    timestamptz NOT NULL,
    delete_after timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

INSERT INTO identity.permissions (code, description) VALUES
    ('administration.analytics.read', 'Xem thống kê bộ lọc'),
    ('administration.export.manage', 'Xuất dữ liệu huấn luyện');

INSERT INTO identity.role_permissions (role_id, permission)
SELECT r.id, p.code FROM identity.roles r, identity.permissions p
WHERE p.code LIKE 'administration.%' AND r.code = 'admin';

-- +goose Down
DELETE FROM identity.permissions WHERE code LIKE 'administration.%';
DROP SCHEMA administration CASCADE;
