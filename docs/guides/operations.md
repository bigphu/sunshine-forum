---
title: Configuration and operations
description: Environment, make targets, checks, background jobs and retention.
---

## Environment

| File | Used by |
|---|---|
| `.env` (from `.env.example`) | `compose.yml`: ports, database name, first admin, `SEED_PASSWORD` |
| `backend/.env.local` (from `.env.local.example`) | Go on the host (`make -C backend run-api`, `run-worker`, tests). `*_FILE` paths are relative to `backend/` |
| `.env` on the server (from `.env.prod.example`) | `compose.prod.yml` |
| `deploy/secrets/*.txt` | Postgres password, the privacy encryption and HMAC keys, the Resend API key. Created by `deploy/scripts/init.sh`; never committed |

Each package owns its variables; the main ones:

| Variable | Default | |
|---|---|---|
| `DB_URL`, `PGPASSWORD_FILE` | | Postgres |
| `PRIVACY_ENC_KEY_FILE`, `PRIVACY_HMAC_KEY_FILE` | | 32-byte keys, base64. Losing them makes personal fields unreadable |
| `HTTP_ADDR`, `HTTP_API_DOCS` | | API address; `true` serves Swagger UI at `/api/docs` |
| `IDENTITY_APP_URL` | `http://localhost:5180` | Base of links in emails |
| `IDENTITY_COOKIE_SECURE` | `true` | Set `false` only for plain-HTTP development |
| `IDENTITY_SESSION_TTL` | `336h` | Session lifetime (14 days) |
| `IDENTITY_INVITE_TTL`, `IDENTITY_RESET_TTL` | `72h`, `24h` | Invitation and reset link lifetimes |
| `IDENTITY_PURGE_AFTER_DAYS` | `90` | Deactivated accounts are deleted after this |
| `EVENTS_PURGE_AFTER_DAYS` | `90` | Delivered event payloads are deleted after this |
| `MAIL_TRANSPORT` | | `smtp` (Mailpit in development) or `resend` (production) |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | | First administrator, created only on an empty database |
| `SCREENING_MODEL_URL` | empty | Model service base URL. Empty: the in-process stand-in. Both compose stacks set it to `http://model:8000` (the mock in development, the trained model in production) |

Filter thresholds, the outage policy, the blocklist and the read-only rule are **not** environment variables:
administrators change them in the app, they apply immediately (Postgres `NOTIFY` clears the screening cache in
every API process), and every change is written to the audit log.

## Make targets

Root `Makefile`:

| Target | |
|---|---|
| `make init` | Secrets and env files, once after cloning |
| `make dev` / `up` / `down` / `restart` / `build` / `ps` | The compose stack |
| `make logs`, `logs-app`, `logs-worker`, `logs-web` | Follow logs |
| `make psql` | A psql shell in the dev database |
| `make seed` | Demo data |
| `make db-reset CONFIRM=yes` | Delete every row and migrate again |
| `make generate` | Backend code (sqlc, oapi-codegen) and frontend API types |
| `make check` | Everything CI runs, for both apps |
| `make images TAG=…` | Production images |
| `make model-image MODEL_TAG=…` | The model service image, from `model/` (needs the trained model file) |

`make -C backend help` lists the Go targets: `migrate`, `migrate-new`, `run-api`, `run-worker`, `test`,
`test-db`, `lint`, `check` and more.

## Checks

```sh
make check
```

runs `sqlc compile`, `gofmt`, `go vet`, golangci-lint (including the module-boundary rules), the Go tests, then
`vue-tsc`, Vitest and the production build. Database tests start Postgres with testcontainers; on Windows, if the
reaper container times out, run them with `TESTCONTAINERS_RYUK_DISABLED=true make -C backend test-db`.

GitHub Actions (`.github/workflows/`) runs the backend and frontend checks on every pull request.

## Background jobs

The worker runs these on a schedule (River periodic jobs):

| Job | Every | Does |
|---|---|---|
| `content.rescan` | 15 s | Scores content held during an outage once the scorer is back |
| `content.purge` | day | Deletes text of removed or deleted content past retention |
| `identity.maintenance` | hour | Expired sessions and links; accounts deactivated more than 90 days ago |
| `notifications.purge` | day | Old notifications |
| `administration.purge_exports` | hour | Exports past their delete-after date |
| `privacy.purge_events` | day | Payloads of delivered events older than `EVENTS_PURGE_AFTER_DAYS` |

## Email

Only invitations and password resets are emailed; every other message lives in the in-app inbox. In development
they go to Mailpit; in production through Resend (`deploy/secrets/resend_api_key.txt`).
