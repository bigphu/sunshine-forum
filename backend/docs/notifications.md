# Notifications module: reference

The in-app inbox and a Server-Sent Events stream (FR-2.2). Email is only used by identity (invitations,
resets). Web Push is not built.

## What creates a notification

| Event | Recipient | Message |
|---|---|---|
| `moderation.decision_recorded` (reject, remove, auto_delete) | author | "Nội dung của bạn trong kênh … đã bị … vì vi phạm Quy tắc N: … Bạn có thể yêu cầu xem xét lại một lần trước ngày …" |
| `moderation.appeal_decided` | author | Accepted or declined wording from COMMUNITY_RULES.md Part 2 |
| `penalties.read_only_started` | account | Read-only wording with end time and last rule |
| `content.published`, an official post | every active resident except the author | "Thông báo mới từ ban quản lý: …" |
| `content.published`, a comment | post author, parent comment author (never the writer) | "… đã bình luận / đã trả lời …" |

Messages always name a community rule, never a model label (FR-4.5). Wording lives in `domain/domain.go`.
Each row has a `dedupe_key` (unique per account), so redelivered events do not duplicate. Restored content
(`first_time = false`) does not notify followers again.

## Real time

A trigger sends `NOTIFY notifications, '<account id>'` on every insert. `Module.Listen` (cmd/server) feeds an
in-process `Broker`; `GET /api/v1/notifications/stream` (hand-written in `handler/stream.go`, outside the
OpenAPI spec because the strict server cannot flush per event) sends `event: notification` and the client
reloads `GET /notifications`. A comment line every 25 seconds keeps proxies from closing the stream. The host
nginx site must disable buffering on this path (`deploy/nginx/sunshine-forum.conf`).

With more than one API replica, each replica listens on its own connection, so this keeps working.

## API (`/api/v1`)

| Method and path | Permission |
|---|---|
| `GET /notifications` | `notifications.read` (50 newest and the unread count) |
| `POST /notifications/read` | `notifications.read` |
| `GET /notifications/stream` | signed in |

`notifications.purge` deletes rows older than `NOTIFICATIONS_PURGE_AFTER_DAYS` (90), daily.
