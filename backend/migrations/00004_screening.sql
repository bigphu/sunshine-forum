-- Module screening: ngưỡng (một dòng duy nhất), ngưỡng theo kênh, danh sách từ khoá.
-- Đổi cài đặt phát NOTIFY screening_settings để API làm mới bản cache mà không cần
-- khởi động lại (FR-5.1).

-- +goose Up
CREATE SCHEMA screening;

CREATE TABLE screening.settings (
    id                    int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    hold_threshold        real NOT NULL DEFAULT 0.5,
    auto_delete_enabled   boolean NOT NULL DEFAULT false,
    auto_delete_threshold real NOT NULL DEFAULT 0.95,
    outage_policy         text NOT NULL DEFAULT 'fail_closed' CHECK (outage_policy IN ('fail_closed', 'fail_open')),
    -- Chỉ dịch vụ chấm điểm tạm (stand-in) dùng: giả lập sự cố để thử NFR-4.2
    stand_in_offline      boolean NOT NULL DEFAULT false,
    updated_at            timestamptz NOT NULL DEFAULT now()
);
INSERT INTO screening.settings (id) VALUES (1);

-- channel_id trỏ tới content.channels nhưng không có khoá ngoại: module không ràng buộc bảng của nhau
CREATE TABLE screening.channel_thresholds (
    channel_id            uuid PRIMARY KEY,
    hold_threshold        real NOT NULL,
    -- NULL: dùng ngưỡng tự động xoá chung
    auto_delete_threshold real
);

CREATE TABLE screening.blocklist (
    id          uuid PRIMARY KEY,
    term        text NOT NULL,
    -- Viết thường, bỏ dấu: khớp cả khi gõ không dấu
    term_folded text NOT NULL UNIQUE,
    rule        int NOT NULL CHECK (rule BETWEEN 1 AND 7),
    created_by  uuid,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE FUNCTION screening.notify_settings() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('screening_settings', '');
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER settings_changed AFTER INSERT OR UPDATE OR DELETE ON screening.settings
    FOR EACH STATEMENT EXECUTE FUNCTION screening.notify_settings();
CREATE TRIGGER channel_thresholds_changed AFTER INSERT OR UPDATE OR DELETE ON screening.channel_thresholds
    FOR EACH STATEMENT EXECUTE FUNCTION screening.notify_settings();
CREATE TRIGGER blocklist_changed AFTER INSERT OR UPDATE OR DELETE ON screening.blocklist
    FOR EACH STATEMENT EXECUTE FUNCTION screening.notify_settings();

INSERT INTO identity.permissions (code, description) VALUES
    ('screening.check', 'Kiểm tra bản nháp trước khi gửi'),
    ('screening.settings.manage', 'Chỉnh ngưỡng, danh sách từ khoá và chính sách khi mô hình gặp sự cố');

INSERT INTO identity.role_permissions (role_id, permission)
SELECT r.id, p.code FROM identity.roles r, identity.permissions p
WHERE p.code = 'screening.check' OR (p.code = 'screening.settings.manage' AND r.code = 'admin');

-- +goose Down
DELETE FROM identity.permissions WHERE code LIKE 'screening.%';
DROP SCHEMA screening CASCADE;
