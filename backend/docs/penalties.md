# Penalties module: reference

Confirmed violations (strikes) and read-only timeouts (FR-1.3). Shadow bans are not used.

- Only `reject`, `remove` and `auto_delete` decisions create a strike. Pre-send warnings and pending content
  never count. `penalties.strikes.decision_id` is unique, so a redelivered event cannot double-count.
- When active strikes within the rolling window reach the limit, a timeout starts for `timeout_days` and
  `penalties.read_only_started` is emitted (notifications sends the message from COMMUNITY_RULES).
- An accepted appeal revokes its strike; if the count drops below the limit, active timeouts are lifted
  (`penalties.read_only_lifted`).
- `content` asks `StandingReader.Standing` before every write, so a timeout applies on the next request.

| Setting | Default |
|---|---|
| Strike limit | 3 |
| Rolling window | 30 days |
| Read-only duration | 3 days |

## API (`/api/v1`)

| Method and path | Permission |
|---|---|
| `GET /me/standing` | signed in |
| `GET/PUT /penalties/settings` | `penalties.settings.manage` |
