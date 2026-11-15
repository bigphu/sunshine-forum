# Events

Modules never write each other's tables. A change is committed together with its event in
`platform.events` and one River job per subscriber (goat `events.Outbox`, transactional outbox). The worker
delivers each job with the actor who caused the change (`identity.LoadActor`). Subscribers must be
idempotent: River retries, and a job can run twice.

Payloads hold IDs and codes only, never text or personal data, and `privacy.purge_events` deletes events
after 90 days.

| Event | Published by | Subscribers |
|---|---|---|
| `content.held` | content (create, edit, rescan) | `moderation.track_content`: open a case |
| `content.published` | content (create, approve, appeal, rescan) | `moderation.track_content`: close an open case; `notifications.notify`: comments, replies, official posts |
| `content.withdrawn` | content (author deletes) | `moderation.track_content`: close the case |
| `content.auto_deleted` | content (create, edit) | `moderation.track_content`: record the system decision |
| `moderation.decision_recorded` | moderation (decide, remove, auto-delete) | `content.apply_moderation`: change status; `penalties.count_strikes`: strike and timeout; `notifications.notify`: message |
| `moderation.appeal_decided` | moderation | `content.apply_moderation`: restore; `penalties.count_strikes`: revoke; `notifications.notify`: message |
| `penalties.read_only_started` | penalties | `notifications.notify`: message |
| `penalties.read_only_lifted` | penalties | none yet |
| `identity.account_*`, `identity.role_changed` | identity | none yet |

Both binaries (and `cmd/seed`) build the same registry: the API needs it to know which jobs to enqueue,
the worker to know which code to run. Register every module's `Subscribe` in all three.

Adding a subscriber: give it a stable name (`<module>.<what>`), register it in the module's `subscriber/`
package, call `Subscribe` from `cmd/server`, `cmd/worker` and `cmd/seed`, and make the handler idempotent
(unique keys or "only if the current status is …" updates).
