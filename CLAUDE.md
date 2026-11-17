# sunshine-apartment-forum

Moderated residents' forum, monorepo in the same shape as StoreIt: `backend/` (Go modular monolith: chi,
oapi-codegen strict server, pgx, River, sqlc, goose; platform packages from `github.com/bigphu/goat` v0.0.2) and
`frontend/` (Vue 3). The root owns the Docker stack (`compose.yml`, `compose.prod.yml`, `deploy/`, `.env`, `Makefile`).
See `README.md`.

## Specs

- `PRODUCT.md`, `SYSTEM_DESIGN.md`, `COMMUNITY_RULES.md`, `DESIGN.md` and `prototype/` are the source of truth.
  `DESIGN.md` is the authority for screens; the prototype's markup and CSS are reused as-is.
- Build only what the first release specifies. Out of scope: Web Push, resident reporting, urgent notices, the
  10-second undo, the Rule 6 pre-send warning, rewrite-and-repost, dark mode, the text-size control, voting.
- The trained model is not connected. `SCREENING_MODEL_URL` selects `scorer.HTTP` (contract:
  `backend/internal/screening/scorer/model-service.yaml`); the dev stack points it at the mock (`cmd/mockmodel`).
  Empty means the in-process `scorer.StandIn`. Change the contract file, the client and the mock together.

## Backend

- Commands (run in `backend/`): `make test`, `make test-db` (Docker-backed tests), `make check` (sqlc-compile,
  fmt, vet, lint, test), `make generate` (sqlc + oapi-codegen; never hand-edit `*.gen.go` or `repository/db/`).
  `make run-api` / `make run-worker` run on the host with `backend/.env.local`.
- If testcontainers' reaper times out on Windows, run DB tests with `TESTCONTAINERS_RYUK_DISABLED=true`.
- Modules: identity, screening, content, moderation, penalties, notifications, administration. References in
  `backend/docs/<module>.md`; events in `backend/docs/events.md`. goat's API reference is its `docs/platform.md`.
- Rules that bite most often:
  - Sessions live in Postgres (cookie `sunshine_session`, `X-CSRF-Token` on unsafe methods). goat's `jwt` and
    bearer-token middleware are not used. `SessionMiddleware` runs at the root router.
  - Services start with `auth.Require`. Transactions, outbox appends and job enqueues live in repositories.
  - Modules talk only through `contract` packages and events; depguard enforces it, including tests.
  - A moderation decision, its audit row and its event are written in one transaction; content changes status
    in a subscriber. Appeals live in moderation for the same reason.
  - Periodic River jobs must not use `UniqueOpts{ByArgs: true}`, or they never run again after completing.
  - Personal fields are encrypted with `internal/privacy`; never log or put them in events.
- Code comments are in Vietnamese; keep that style. Files use LF; with Python on Windows open files with `newline=''`.

## Frontend

- `frontend/`: Vue 3 + TypeScript + Vue Query + Pinia + Reka UI (menus only). Reference in `frontend/README.md`.
- Styling is the prototype's CSS (`src/app/forum.css`, copied verbatim; additions go in `src/app/app.css`).
  Palette is gold on navy. Dialogs are native `<dialog>` (`AppDialog`), as in the prototype.
- Commands (in `frontend/`): `npm run dev` (:5180, proxies `/api` to :8091), `npm run check` (vue-tsc, Vitest,
  build), `npm run gen:api` after changing a backend OpenAPI spec (commit `src/lib/api/*.d.ts`).
- UI text is Vietnamese, taken from the prototype and COMMUNITY_RULES.md. Comments in Vietnamese.

## Docs

- `docs/guides/` explains running and operating the system; `backend/docs/*.md` and `frontend/README.md` are the
  developer references. Update them in the same change when behaviour changes.
