# Moderation module: reference

The review queue, decisions (also in bulk), removing published content, re-evaluation requests (appeals)
and the append-only audit log.

Appeals live here rather than in a separate module (SYSTEM_DESIGN lists "appeals" on its own) because an
appeal decision must write its audit row in the same transaction, and the audit log belongs to moderation.

## Tables

| Table | Holds |
|---|---|
| `moderation.cases` | One open case per held item (`content_type`, `content_id`) |
| `moderation.decisions` | approve, reject, remove, auto_delete; rule, moderator, AI label, score, model version, `overturned_ai`, appeal deadline |
| `moderation.appeals` | One per decision (unique), reason 20–2,000 characters |
| `moderation.audit_log` | Append-only. A trigger rejects UPDATE, DELETE and TRUNCATE. No content, pseudonymous subjects (`privacy.Cipher.Pseudonym`) |

Moderation never reads content tables: item text, author and AI result come from `content.Reader`.

## Flows

**Held content.** `content.held` → subscriber opens a case (idempotent). `content.published`,
`content.withdrawn` close it as withdrawn (the author fixed or deleted it, or the rescan cleared it).

**Decide** (`POST /moderation/decisions`). Each item: lock the open case, insert the decision, close the case,
insert the audit row and emit `moderation.decision_recorded`, all in one transaction. Items whose case is gone
are skipped, so two managers deciding at once cannot double-decide. Rejecting requires a rule, also in bulk
(FR-4.5). Approving something the filter flagged (model label or blocklist) sets `overturned_ai`.

**Remove** (`POST /moderation/removals`). Only published content. A decision without a case, a rule, a 30-day
appeal deadline, and `overturned_ai` when the model had scored it (a false negative, FR-5.3).

**Auto-delete.** `content.auto_deleted` → the subscriber records a system decision (no moderator) with the
appeal deadline.

**Appeals.** Allowed once per reject, remove or auto-delete decision, within 30 days, by its author. The
manager who made the decision cannot review it (`403 /errors/same-moderator`); another manager or an Admin
does. Accepting emits `moderation.appeal_decided` with `accepted = true`: content restores the item and
penalties revokes the strike. The result is final.

## Audit actions

`approve`, `reject`, `remove`, `auto_delete`, `appeal_accepted`, `appeal_declined`. Outcomes: `published`,
`rejected`, `removed`, `auto_deleted`, `restored`, `permanently_rejected`.

## API (`/api/v1`)

| Method and path | Permission |
|---|---|
| `GET /moderation/queue` | `moderation.review` (appeals need `moderation.appeal.review`) |
| `POST /moderation/decisions` | `moderation.review` |
| `POST /moderation/removals` | `moderation.remove` |
| `POST /moderation/appeals/{id}/review` | `moderation.appeal.review` |
| `POST /appeals`, `GET /me/decisions` | `moderation.appeal.create` |
| `GET /audit?before=&overturned_only=` | `moderation.audit.read` |

## Contract

Event names and payloads, `Reader.Stats` and `Reader.Corrections` (finalized decisions that overturned the
AI, for the retraining export; Rules 5–7 excluded).
