-- name: CreatePasswordToken :exec
INSERT INTO identity.password_tokens (token_hash, account_id, purpose, expires_at)
VALUES (@token_hash, @account_id, @purpose, @expires_at);

-- Link mới thay link cũ: lời mời gửi lại huỷ link trước
-- name: RevokePasswordTokens :exec
UPDATE identity.password_tokens SET used_at = now()
WHERE account_id = @account_id AND purpose = @purpose AND used_at IS NULL;

-- name: GetValidPasswordToken :one
SELECT * FROM identity.password_tokens
WHERE token_hash = @token_hash AND used_at IS NULL AND expires_at > now();

-- name: UsePasswordToken :execrows
UPDATE identity.password_tokens SET used_at = now()
WHERE token_hash = @token_hash AND used_at IS NULL AND expires_at > now();

-- name: PendingInviteExpiry :one
SELECT expires_at FROM identity.password_tokens
WHERE account_id = @account_id AND purpose = 'invite' AND used_at IS NULL
ORDER BY expires_at DESC LIMIT 1;

-- name: PruneTokens :execrows
DELETE FROM identity.password_tokens WHERE expires_at < now() - interval '30 days';
