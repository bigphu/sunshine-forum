-- name: GetSettings :one
SELECT * FROM penalties.settings WHERE id = 1;

-- name: UpdateSettings :one
UPDATE penalties.settings
SET strike_limit = @strike_limit, strike_window_days = @strike_window_days, timeout_days = @timeout_days, updated_at = now()
WHERE id = 1
RETURNING *;

-- decision_id duy nhất: event giao lại không tạo vi phạm trùng
-- name: InsertStrike :execrows
INSERT INTO penalties.strikes (id, account_id, decision_id, rule) VALUES (@id, @account_id, @decision_id, @rule)
ON CONFLICT (decision_id) DO NOTHING;

-- name: RevokeStrike :one
UPDATE penalties.strikes SET revoked_at = now() WHERE decision_id = @decision_id AND revoked_at IS NULL
RETURNING account_id;

-- name: CountActiveStrikes :one
SELECT count(*) FROM penalties.strikes
WHERE account_id = @account_id AND revoked_at IS NULL
  AND created_at > now() - make_interval(days => @window_days::int);

-- name: InsertTimeout :execrows
INSERT INTO penalties.timeouts (id, account_id, decision_id, last_rule, starts_at, ends_at)
VALUES (@id, @account_id, @decision_id, @last_rule, now(), @ends_at)
ON CONFLICT (decision_id) DO NOTHING;

-- name: ActiveTimeout :one
SELECT * FROM penalties.timeouts
WHERE account_id = @account_id AND lifted_at IS NULL AND ends_at > now()
ORDER BY ends_at DESC LIMIT 1;

-- name: LiftTimeouts :execrows
UPDATE penalties.timeouts SET lifted_at = now()
WHERE account_id = @account_id AND lifted_at IS NULL AND ends_at > now();

-- name: Stats :one
SELECT
    (SELECT count(*) FROM penalties.strikes s WHERE s.revoked_at IS NULL
        AND s.created_at > now() - make_interval(days => @window_days::int)) AS active_strikes,
    (SELECT count(DISTINCT t.account_id) FROM penalties.timeouts t WHERE t.lifted_at IS NULL AND t.ends_at > now()) AS read_only;

-- Tài khoản đang tạm chỉ đọc, hết hạn muộn nhất của mỗi người (trang Người dùng, Phân tích)
-- name: ListActiveTimeouts :many
SELECT DISTINCT ON (account_id) account_id, ends_at, last_rule FROM penalties.timeouts
WHERE lifted_at IS NULL AND ends_at > now()
ORDER BY account_id, ends_at DESC;
