# Identity module: reference

Accounts exist only by invitation from the tenant registry (FR-1.1). Three fixed roles, sessions stored
in PostgreSQL, CSRF tokens, Argon2id passwords, and email only for invitations and password resets.

## Layout

| Path | Holds |
|---|---|
| `domain/` | `Account`, `Role` (resident, manager, admin), validation (`NormalizeEmail`, `CleanName`, `CheckPassword`), errors, repository interfaces |
| `service/` | Sign-in, sessions, invitations, password reset, admin actions, email rendering (`templates/`), maintenance |
| `repository/` | sqlc queries (`queries/*.sql` → `db/`), encryption of personal fields, outbox events, email jobs |
| `handler/` | `openapi.yaml` → `api/api.gen.go`, cookies, `SessionMiddleware` |
| `job/`, `worker/` | `identity.send_account_email`, `identity.maintenance` |
| `contract/` | `AccountReader` (display names, resident IDs) and event names |

## Wiring

```go
idm, err := identity.New(ctx, identity.Deps{Pool: pool, Outbox: outbox, Jobs: enq, Cipher: cipher, Config: cfg.Identity})
r.Use(idm.SessionMiddleware())   // cmd/server, before any module mounts
err = idm.Bootstrap(ctx)         // first Admin from ADMIN_EMAIL / ADMIN_PASSWORD
err = idm.Mount(r)
events.RegisterWorker(workers, pool, registry, idm.LoadActor) // cmd/worker
idm.RegisterWorkers(workers)     // cmd/worker; needs Deps.Mail
```

`New` loads each role's permissions once: roles are fixed and permissions only change through migrations.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `IDENTITY_SESSION_TTL` | `336h` | Sliding session lifetime |
| `IDENTITY_COOKIE_SECURE` | `true` | Set `false` only for local http |
| `IDENTITY_COOKIE_DOMAIN` | empty | Cookie domain |
| `IDENTITY_APP_URL` | `http://localhost:5180` | Frontend origin for links in emails |
| `IDENTITY_INVITE_TTL` | `72h` | Invitation link lifetime (PRODUCT.md: 3 days) |
| `IDENTITY_RESET_TTL` | `24h` | Reset link lifetime (1 day) |
| `IDENTITY_PURGE_AFTER_DAYS` | `90` | Personal data of deactivated accounts is erased after this |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | empty | First Admin, created when there are no accounts |
| `PRIVACY_ENC_KEY_FILE`, `PRIVACY_HMAC_KEY_FILE` | required | 32-byte keys, base64 (see `internal/privacy`) |

## Sessions and CSRF

- Sign-in sets `sunshine_session` (HttpOnly, SameSite=Lax, Secure in production). The database stores the
  SHA-256 of the token, never the token.
- `SessionMiddleware` runs on every request: it loads the session and the account's **current** role, so a
  role change, a deactivation or an expired session applies on the next request (no JWTs).
- Every POST, PUT, PATCH and DELETE that carries the cookie must send `X-CSRF-Token` with the
  `csrf_token` from `GET /me` (or the sign-in response). A mismatch is `403 /errors/csrf`.
- Sessions slide: `expires_at` moves forward at most once a minute.
- Changing your password signs out your other devices; setting a password from a link signs out all.

## Invitations and password reset

1. Admin `POST /accounts` creates an `invited` account and, in the same transaction, a token row (hash only)
   and a `identity.send_account_email` job carrying the raw token.
2. The worker renders `templates/invite.*` with the link `IDENTITY_APP_URL/dat-mat-khau#token=…` (fragment,
   so the token never reaches server logs) and sends it. Dead links are skipped, permanent mail errors cancel.
3. `POST /auth/password/inspect` shows who the link is for; `POST /auth/password/set` checks the policy
   (8+ characters, not the email), activates the account and signs in.

Resending an invitation or requesting a reset revokes earlier links of the same purpose. `POST
/auth/password/forgot` always answers 202.

## Personal data

Email, name and unit are encrypted with AES-256-GCM (`privacy.Cipher`); `email_hash` (HMAC-SHA-256) is the
lookup key. The account list is sorted in Go with Vietnamese collation because names are encrypted. Accounts
deactivated for 90 days are erased (`status = purged`, name shown as "Cựu cư dân") but keep their ID so old
posts keep an author.

## API (`/api/v1`)

| Method and path | Permission | Notes |
|---|---|---|
| `POST /auth/login` | public, 10 per minute per IP | Sets the cookie, returns `Me` with `csrf_token` |
| `POST /auth/logout` | public | Clears the cookie |
| `POST /auth/password/forgot` | public, 5 per minute | Always 202 |
| `POST /auth/password/inspect` | public | Token in the body |
| `POST /auth/password/set` | public | Signs in |
| `GET /me` | signed in | Account, permissions, CSRF token |
| `PUT /me/password` | signed in | Needs the current password |
| `GET /accounts`, `POST /accounts` | `identity.account.manage` | List, invite |
| `POST /accounts/{id}/reinvite`, `/deactivate`, `PUT /accounts/{id}/role` | `identity.account.manage` | Not on your own account |

## RBAC

Permissions are rows in `identity.permissions`; each module's migration inserts its own codes and grants
them. Services start with `auth.Require(ctx, "<module>.<thing>.<action>")`.

| Permission | Resident | Manager | Admin |
|---|:-:|:-:|:-:|
| `content.read`, `content.write`, `screening.check`, `moderation.appeal.create`, `notifications.read` | ✓ | ✓ | ✓ |
| `content.announce`, `content.pin`, `moderation.review`, `moderation.remove`, `moderation.appeal.review` | | ✓ | ✓ |
| `identity.account.manage`, `screening.settings.manage`, `penalties.settings.manage`, `moderation.audit.read`, `administration.*` | | | ✓ |

## Events

`identity.account_invited`, `identity.account_activated`, `identity.account_deactivated`,
`identity.role_changed` (aggregate `identity.account`, IDs only). No subscribers yet.
