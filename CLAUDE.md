# sunshine-apartment-forum

Moderated residents' forum, monorepo in the same shape as StoreIt: `backend/` (Go modular monolith: chi,
oapi-codegen strict server, pgx, River, sqlc, goose; platform packages from `github.com/bigphu/goat` v0.0.2),
`frontend/` (Vue 3) and `model/` (git submodule: the Python toxicity model and its FastAPI scoring service). The
root owns the Docker stack (`compose.yml`, `compose.prod.yml`, `deploy/`, `.env`, `Makefile`). See `README.md`;
`docs/README.md` maps the documentation.

## Specs

- `PRODUCT.md`, `DESIGN.md` (root, because the design tooling reads them there), `docs/specs/SYSTEM_DESIGN.md`,
  `docs/specs/COMMUNITY_RULES.md`, `docs/specs/UI_DESIGN_SPEC.md` and `docs/prototype/` are the source of truth.
  `DESIGN.md` is the authority for screens; the prototype's stylesheet is reused as-is.
- Build only what the first release specifies. Out of scope: Web Push, resident reporting, urgent notices, the
  10-second undo, the Rule 6 pre-send warning, rewrite-and-repost, dark mode, the text-size control, voting.
- `SCREENING_MODEL_URL` selects `scorer.HTTP` (contract: `backend/internal/screening/scorer/model-service.yaml`).
  The dev stack points it at the mock (`cmd/mockmodel`), production at the `model` service built from `model/`
  (`make model-image`). Empty means the in-process `scorer.StandIn`. Change the contract file, the client, the mock
  and `model/src/scoring.py` together. The `model/` submodule has its own repo, branch and commits.

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

## Repository hygiene

- Don't commit binaries (`*.exe`, `bin/`), trained models (`model/models/`), secrets (`deploy/secrets/`, `.env*`
  except the examples) or per-person tool state (`.impeccable/`, `.claude/skills|agents`); all are git-ignored.
- On this Windows machine, run Vitest from a `C:\` path (uppercase drive letter); a lowercase `c:\` working
  directory makes it load twice and fail before any test runs.

## Docs

- `docs/guides/` explains running and operating the system; `backend/docs/*.md` and `frontend/README.md` are the
  developer references. Update them in the same change when behaviour changes.
