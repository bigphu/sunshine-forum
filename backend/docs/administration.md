# Administration module: reference

Filter analytics (FR-5.4) and the minimal retraining export (US-6.4). Settings, accounts and the audit log
belong to screening, penalties, identity and moderation; the Admin pages call those APIs directly.

## Analytics (`GET /admin/analytics?days=30`)

Read through contracts: submitted and flagged counts plus 14 days of flagged-per-day (content), decision and
appeal counts (moderation), active strikes and read-only accounts (penalties). `precision` is
`rejected / (rejected + approved)` among held items a manager decided. "Flagged" means held by the filter
(blocklist, language, outage, or an `offensive`/`hate` label) regardless of the later decision.

## Retraining export

SYSTEM_DESIGN §3: no long-term store and no change to NFR-3.3.

- **Eligible:** finalized decisions that overturned the AI (approvals of flagged content, removals of content
  the model published, accepted appeals of auto-deletions), decided 30–89 days ago so the appeal period has
  closed, excluding Rules 5–7 and content the author deleted or that was already purged.
- **Each line** (JSON Lines): `text`, final `label` (`clean`, `offensive`, `hate`), `rule`, `ai_score`,
  `model_version`. No user or post IDs. Emails, phone numbers and number plates are replaced (`domain.Scrub`).
- **Deletion:** `delete_after` is the oldest included decision plus 90 days; `administration.purge_exports`
  removes expired exports hourly. Admins can delete earlier.

## API (`/api/v1`)

| Method and path | Permission |
|---|---|
| `GET /admin/analytics` | `administration.analytics.read` |
| `GET/POST /admin/exports`, `GET/DELETE /admin/exports/{id}` | `administration.export.manage` |
