# Content module: reference

Channels, posts, two-level comments, the official marker, pinning and search.

## Rules

- Four channels are seeded by the migration (slug: `thong-bao`, `thao-luan`, `bao-tri`, `marketplace`). Only
  accounts with `content.announce` post in `official_only` channels; residents comment there (FR-2.3).
- Titles 1–200 characters, bodies 1–5,000 characters counted as characters, not bytes (FR-2.7).
- Every create and edit is screened (`screening.Screener`) before it is saved (US-3.2, US-3.3). Posts are
  screened title and body separately.
- Writing checks `penalties.StandingReader` first: a read-only account gets `403 /errors/read-only`.
- `official` is set when the author holds `content.announce`. A resident post whose title or opening looks
  like a management notice comes back with `imitates_official = true` so the UI adds a disclaimer.
- Comments are two levels. A reply to a reply attaches to the top-level comment.
- Unpublished posts are visible to their author only. Hidden or deleted comments that still have visible
  replies come back as placeholders without text.
- Deleting is soft (`status = deleted`). Deleted, rejected, removed and auto-deleted content is purged after
  `CONTENT_PURGE_AFTER_DAYS` (90). Comments that still have replies keep an empty row as the placeholder.

## Status changes

| From | To | By |
|---|---|---|
| new | `published` / `pending` / `auto_deleted` | screening result |
| `pending` | `published` / `rejected` | moderation decision (subscriber) |
| `published` | `removed` | manager removal (subscriber) |
| `rejected`, `removed`, `auto_deleted` | `published` | accepted appeal (subscriber) |
| `published`, `pending` | `deleted` | author |
| held by an outage | screening result | rescan job |

Every write emits its event in the same transaction (see events.md). Moderation never writes content
tables: it records a decision and `subscriber/` applies it, so the thread updates a moment after the decision
(when the worker delivers the event).

## Jobs

| Job | Schedule | Does |
|---|---|---|
| `content.rescan` | every 15 s | Re-screens `needs_rescan` content once `ModelOnline`. After an outage nothing is auto-deleted; violations are held. |
| `content.purge` | daily | 90-day purge |

## Search

`search_text` stores title and body folded with `vntext.Fold`. `GET /search?q=` folds the query the same way
and matches with `LIKE`, so "thang may" finds "thang máy". For better ranking, SYSTEM_DESIGN suggests
`unaccent` + `pg_trgm` or a search engine later.

## API (`/api/v1`)

| Method and path | Permission |
|---|---|
| `GET /channels`, `GET /posts?channel=&cursor=`, `GET /posts/{id}`, `GET /search?q=`, `GET /me/pending` | `content.read` |
| `POST /posts`, `PATCH/DELETE /posts/{id}`, `POST /posts/{id}/comments`, `PATCH/DELETE /comments/{id}` | `content.write` (own content) |
| `PUT /posts/{id}/pin` | `content.pin` |

Feed paging is keyset on `(created_at, id)`; the cursor is opaque. Pinned posts come on the first page only.

## Contract

`contract.Reader` (`Items`, `Channel`, `Stats`) for moderation, notifications and administration, plus the
event payloads in `contract/events.go`.
