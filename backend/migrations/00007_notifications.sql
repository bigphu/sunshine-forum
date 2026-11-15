-- Module notifications: hộp thư trong ứng dụng (FR-2.2, FR-4.5). Thêm thông báo phát
-- NOTIFY notifications với account_id để API đẩy qua Server-Sent Events.

-- +goose Up
CREATE SCHEMA notifications;

CREATE TABLE notifications.items (
    id         uuid PRIMARY KEY,
    account_id uuid NOT NULL,
    kind       text NOT NULL,
    -- Khoá chống trùng khi subscriber chạy lại, vd "decision:<id>"
    dedupe_key text NOT NULL,
    body       text NOT NULL,
    link       text NOT NULL DEFAULT '',
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, dedupe_key)
);
CREATE INDEX items_account_idx ON notifications.items (account_id, created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION notifications.notify_new() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('notifications', NEW.account_id::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER items_inserted AFTER INSERT ON notifications.items
    FOR EACH ROW EXECUTE FUNCTION notifications.notify_new();

INSERT INTO identity.permissions (code, description) VALUES
    ('notifications.read', 'Đọc thông báo của mình');

INSERT INTO identity.role_permissions (role_id, permission)
SELECT r.id, 'notifications.read' FROM identity.roles r;

-- +goose Down
DELETE FROM identity.permissions WHERE code LIKE 'notifications.%';
DROP SCHEMA notifications CASCADE;
