---
title: Architecture
description: Services, modules, events, and how a post moves through the system.
---

The full design, with UML views, is in `docs/specs/SYSTEM_DESIGN.md` and `docs/architecture/` (the system documentation and its diagrams).
This page is the short version for developers.

## The stack

| Service | Role |
|---|---|
| `postgres` | The only datastore (Postgres 17): data, sessions, the event outbox and River's job tables |
| `migrate` | Applies goose and River migrations before the app starts |
| `app` | The HTTP API (`cmd/server`). Swagger UI at `/api/docs` in development |
| `worker` | `cmd/worker`: delivers outbox events to subscribers, sends emails, rescans after outages, purges |
| `web` | The Vue SPA. Vite in development; nginx in production, serving the SPA and proxying `/api` to `app` |
| `model` | The model service. In development, the mock (`cmd/mockmodel`) with sample scores; in production, the trained SVM from the `model/` submodule (`make model-image`) |
| `mailpit` | Development mail inbox |
| `seed` | Demo data, behind the `tools` profile |

In production only `web` is published, on `127.0.0.1:${WEB_PORT}`; host nginx terminates HTTPS
(`deploy/nginx/sunshine-forum.conf`).

## Backend: a modular monolith

One Go program split into modules under `backend/internal/`. Platform packages (config, logger, database,
server, middleware, web, errs, auth, events, jobs, mail) come from goat v0.0.2, a port of StoreIt's platform.

| Module | Owns |
|---|---|
| `identity` | Accounts (encrypted personal fields), invitations, sessions and CSRF, roles and permissions, password reset |
| `screening` | Rules, blocklist, language check, the scorer plug, thresholds, outage policy, the draft check |
| `content` | Channels, posts, comments, pinning, search, the rescan after an outage |
| `moderation` | The queue, decisions, removals, appeals, the audit log |
| `penalties` | Strikes and read-only timeouts |
| `notifications` | The in-app inbox and its Server-Sent Events stream |
| `administration` | Analytics and the retraining export |

Inside a module the layers only depend downwards:

| Package | Holds |
|---|---|
| `domain` | Entities, rules, errors, permission codes |
| `service` | Use cases. Every public call starts with `auth.Require` |
| `repository` | sqlc queries, transactions, outbox appends, job enqueues |
| `handler` | `openapi.yaml`, the generated strict server, `Mount` |
| `contract` | What other modules may use: reader interfaces, DTOs, event types |
| `subscriber`, `job`, `worker` | Event handlers and River jobs, run by `cmd/worker` |

Modules never read each other's tables. They call each other only through `contract` interfaces, or react to
events. depguard enforces this, tests included.

Shared packages that are not modules: `internal/privacy` (AES-GCM field encryption, HMAC lookup hashes,
pseudonyms, purging old event payloads), `internal/vntext` (folding Vietnamese for search and the blocklist),
`internal/dbmigrate`, `internal/testdb`.

## Sessions instead of tokens

Sessions are rows in Postgres; the browser holds an HTTP-only `sunshine_session` cookie (SameSite=Lax) and sends
`X-CSRF-Token` on every write. The session middleware reloads the account's role and permissions on every request,
so a role change, a deactivation or a read-only timeout applies on the very next request. goat's JWT and bearer
middleware are not used.

## How a post moves

1. **Draft check.** While the writer types, the SPA checks the blocklist and phone numbers locally and calls
   `POST /screening/check` after a pause. Warnings never count as violations.
2. **Submit.** `content` calls `screening.Screen` for the title and body. The result is *publish*, *hold* or (only
   when enabled) *auto-delete*, with the label, score, model version and sentence offsets. The post is stored with
   that status and a `content.held` or `content.published` event goes into the outbox in the same transaction.
3. **Queue.** `moderation` subscribes to `content.held` and opens a case. Managers see it grouped by priority.
4. **Decision.** Approve or reject writes the decision, the audit row and `moderation.decision_recorded` in one
   transaction. Subscribers then publish or reject the content (`content`), count a strike (`penalties`) and tell
   the writer (`notifications`).
5. **Appeal.** The writer appeals once within 30 days. A different moderator decides; accepting restores the
   content and removes the strike, through the same event path.

The backend talks to the model service over HTTP with a 200 ms deadline; the contract is
`backend/internal/screening/scorer/model-service.yaml`. If the scorer is down, the outage policy decides: *fail closed* (default) holds new content, *fail open* publishes
what passes the blocklist. Either way the worker rescans held content every 15 seconds once the scorer is back.
Comments in the management channel are always held during an outage.

## Events

Every event, its publisher and its subscribers are listed in `backend/docs/events.md`. Delivery is at least once,
so subscribers are idempotent (the audit log's `event_id` is unique, notifications carry a dedupe key).

## Frontend

Vue 3 with TypeScript, Vue Query and Pinia. API types are generated from the backend OpenAPI specs
(`npm run gen:api`). Styling is the prototype's CSS, unchanged. See `frontend/README.md`.
