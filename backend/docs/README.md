# Backend reference

Developer references for the Go backend. People-facing guides are in `docs/guides/`.

| Document | Covers |
|---|---|
| [identity.md](identity.md) | Accounts, invitations, sessions, CSRF, roles and permissions, password reset |
| [screening.md](screening.md) | Blocklist, language check, the model plug, thresholds, outage policy, draft check |
| [content.md](content.md) | Channels, posts, comments, search, rescan after outages |
| [moderation.md](moderation.md) | Queue, decisions, removals, appeals, audit log |
| [penalties.md](penalties.md) | Strikes and read-only timeouts |
| [notifications.md](notifications.md) | Inbox, Server-Sent Events, message wording |
| [administration.md](administration.md) | Analytics and the retraining export |
| [events.md](events.md) | Every event, who publishes it and who reacts |

The platform packages (`config`, `logger`, `database`, `server`, `middleware`, `web`, `errs`, `auth`,
`events`, `jobs`, `mail`) come from [goat](https://github.com/bigphu/goat) v0.0.2, the published port of
StoreIt's `internal/platform`. Its `docs/platform.md` is the API reference. goat's `jwt` and bearer-token
`auth.Middleware` are not used: sessions live in PostgreSQL (see identity.md).

Shared packages that are not modules: `internal/privacy` (field encryption, pseudonyms, event purge),
`internal/vntext` (Vietnamese text folding), `internal/dbmigrate` (goose + River migrations),
`internal/testdb` (Postgres for tests).
