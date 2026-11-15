-- name: CreateAccount :one
INSERT INTO identity.accounts (id, email_hash, email_enc, name_enc, unit_enc, role_id, status)
VALUES (@id, @email_hash, @email_enc, @name_enc, @unit_enc, @role_id, @status)
RETURNING *;

-- name: GetAccount :one
SELECT * FROM identity.accounts WHERE id = @id;

-- name: GetAccountByEmailHash :one
SELECT * FROM identity.accounts WHERE email_hash = @email_hash;

-- name: GetAccountsByIDs :many
SELECT * FROM identity.accounts WHERE id = ANY(@ids::uuid[]);

-- Tên đã mã hoá nên không sắp theo tên trong SQL; service sắp sau khi giải mã
-- name: ListAccounts :many
SELECT * FROM identity.accounts WHERE status <> 'purged' ORDER BY created_at;

-- name: ListActiveAccountIDsByRole :many
SELECT a.id FROM identity.accounts a JOIN identity.roles r ON r.id = a.role_id
WHERE a.status = 'active' AND r.code = ANY(@codes::text[]);

-- name: SetPassword :one
UPDATE identity.accounts SET password_hash = @password_hash, status = 'active', updated_at = now()
WHERE id = @id AND status IN ('invited', 'active')
RETURNING *;

-- name: SetRole :one
UPDATE identity.accounts SET role_id = @role_id, updated_at = now()
WHERE id = @id AND status IN ('invited', 'active')
RETURNING *;

-- name: Deactivate :one
UPDATE identity.accounts SET status = 'deactivated', deactivated_at = now(), updated_at = now()
WHERE id = @id AND status IN ('invited', 'active')
RETURNING *;

-- name: TouchSignIn :exec
UPDATE identity.accounts SET last_sign_in_at = now() WHERE id = @id;

-- name: CountAccounts :one
SELECT count(*) FROM identity.accounts;

-- Tài khoản đã khoá quá 90 ngày: xoá dữ liệu cá nhân, giữ ID để bài cũ còn tác giả (NFR-3.3)
-- name: PurgeDeactivated :execrows
UPDATE identity.accounts
SET status = 'purged', email_hash = sha256(id::text::bytea), email_enc = ''::bytea, name_enc = ''::bytea,
    unit_enc = ''::bytea, password_hash = NULL, updated_at = now()
WHERE status = 'deactivated' AND deactivated_at < now() - make_interval(days => @days::int);
