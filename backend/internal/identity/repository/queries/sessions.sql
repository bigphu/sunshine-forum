-- name: CreateSession :exec
INSERT INTO identity.sessions (token_hash, account_id, csrf_token, expires_at)
VALUES (@token_hash, @account_id, @csrf_token, @expires_at);

-- Phiên còn hạn của một account đang hoạt động, kèm vai trò hiện tại: đọc ở mỗi request
-- nên đổi vai trò hay khoá tài khoản có hiệu lực ngay ở request kế tiếp
-- name: GetActiveSession :one
SELECT s.token_hash, s.account_id, s.csrf_token, s.last_seen_at, s.expires_at, a.role_id
FROM identity.sessions s JOIN identity.accounts a ON a.id = s.account_id
WHERE s.token_hash = @token_hash AND s.expires_at > now() AND a.status = 'active';

-- Gia hạn trượt; chỉ ghi khi lần ghi trước đã cũ để không ghi ở mọi request
-- name: ExtendSession :exec
UPDATE identity.sessions SET last_seen_at = now(), expires_at = @expires_at
WHERE token_hash = @token_hash;

-- name: DeleteSession :exec
DELETE FROM identity.sessions WHERE token_hash = @token_hash;

-- name: DeleteAccountSessions :exec
DELETE FROM identity.sessions WHERE account_id = @account_id;

-- name: DeleteOtherSessions :exec
DELETE FROM identity.sessions WHERE account_id = @account_id AND token_hash <> @keep;

-- name: PruneSessions :execrows
DELETE FROM identity.sessions WHERE expires_at < now();
