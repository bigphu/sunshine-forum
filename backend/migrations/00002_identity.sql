-- Module identity: tài khoản (chỉ tạo qua lời mời), ba vai trò cố định, quyền,
-- phiên đăng nhập lưu trong DB và link mời / đặt lại mật khẩu.
-- Email, tên và căn hộ được mã hoá trong ứng dụng (NFR-3.1); email_hash là HMAC để tra cứu.

-- +goose Up
CREATE SCHEMA identity;

CREATE TABLE identity.roles (
    id          uuid PRIMARY KEY,
    code        text NOT NULL UNIQUE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT ''
);

-- Mã quyền của mọi module; module khác thêm mã của mình bằng migration riêng
CREATE TABLE identity.permissions (
    code        text PRIMARY KEY,
    description text NOT NULL DEFAULT ''
);

CREATE TABLE identity.role_permissions (
    role_id    uuid NOT NULL REFERENCES identity.roles (id) ON DELETE CASCADE,
    permission text NOT NULL REFERENCES identity.permissions (code) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission)
);

CREATE TABLE identity.accounts (
    id              uuid PRIMARY KEY,
    email_hash      bytea NOT NULL UNIQUE,
    email_enc       bytea NOT NULL,
    name_enc        bytea NOT NULL,
    unit_enc        bytea NOT NULL,
    role_id         uuid NOT NULL REFERENCES identity.roles (id),
    status          text NOT NULL CHECK (status IN ('invited', 'active', 'deactivated', 'purged')),
    password_hash   text,
    last_sign_in_at timestamptz,
    deactivated_at  timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX accounts_role_idx ON identity.accounts (role_id) WHERE status = 'active';

-- Cookie chỉ giữ token thô; DB giữ SHA-256 của nó. csrf_token gửi lại qua header X-CSRF-Token
CREATE TABLE identity.sessions (
    token_hash   bytea PRIMARY KEY,
    account_id   uuid NOT NULL REFERENCES identity.accounts (id) ON DELETE CASCADE,
    csrf_token   text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE INDEX sessions_account_idx ON identity.sessions (account_id);
CREATE INDEX sessions_expires_idx ON identity.sessions (expires_at);

CREATE TABLE identity.password_tokens (
    token_hash bytea PRIMARY KEY,
    account_id uuid NOT NULL REFERENCES identity.accounts (id) ON DELETE CASCADE,
    purpose    text NOT NULL CHECK (purpose IN ('invite', 'reset')),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX password_tokens_account_idx ON identity.password_tokens (account_id);

INSERT INTO identity.roles (id, code, name, description) VALUES
    ('00000000-0000-7000-8000-000000000001', 'resident', 'Cư dân', 'Đọc, đăng bài và bình luận trong các kênh dành cho cư dân.'),
    ('00000000-0000-7000-8000-000000000002', 'manager', 'Quản lý', 'Đăng thông báo chính thức, kiểm duyệt nội dung và xem xét yêu cầu xem xét lại.'),
    ('00000000-0000-7000-8000-000000000003', 'admin', 'Quản trị viên', 'Mọi quyền của Quản lý, cộng quản lý tài khoản, bộ lọc, thống kê và nhật ký.');

INSERT INTO identity.permissions (code, description) VALUES
    ('identity.account.manage', 'Mời, khoá tài khoản và đổi vai trò');

INSERT INTO identity.role_permissions (role_id, permission) VALUES
    ('00000000-0000-7000-8000-000000000003', 'identity.account.manage');

-- +goose Down
DROP SCHEMA identity CASCADE;
