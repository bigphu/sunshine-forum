# Sunshine Apartment Forum

A moderated forum for one building's verified residents. Every post and comment is screened before it goes
public, writers are warned before they submit, managers review what the filter holds back, and every
rejection can be appealed once.

New here? Read [PRODUCT.md](PRODUCT.md) for what the forum is for, then [Getting started](docs/guides/getting-started.md)
to run it. [docs/README.md](docs/README.md) maps every other document.

## What it does

- **Forum**: four channels (management notices, general talk, maintenance, marketplace), posts, two-level
  comments, official replies, pinning, search that works without diacritics.
- **Screening**: a draft check while the writer types, then a blocklist, a language check and a model score on
  submit. Content below the hold threshold is published, above it is held for review. Auto-delete exists but is
  off by default.
- **Moderation**: a priority queue (appeals, high, needs review, blocklist) with hidden text, rule picker, message
  preview, bulk actions and keyboard shortcuts; removal of published content.
- **Appeals and penalties**: one appeal per decision within 30 days, reviewed by a different moderator. Confirmed
  violations add strikes; enough strikes in the window make the account read-only for a while.
- **Administration**: invitations from the tenant register (one at a time or a CSV file), deactivation,
  roles, filter thresholds per channel, outage policy, blocklist, analytics, the append-only audit log, and the
  retraining export.
- **Privacy**: names, emails and units are encrypted at rest; the audit log uses pseudonyms; deactivated
  accounts and old event payloads are purged after 90 days.

The toxicity model is a calibrated TF-IDF + linear SVM trained on ViHSD, served by a small FastAPI service in
`model/`. The backend calls it over HTTP (`SCREENING_MODEL_URL`); in development a mock with sample scores stands
in, so the forum runs without the trained model.

## Repository

| Path | Holds |
|---|---|
| `PRODUCT.md` | What the product is for, who uses it, and the requirements |
| `DESIGN.md` | The design system; the authority for every screen |
| `backend/` | Go API, worker, migrations and seed. Module references in `backend/docs/` |
| `frontend/` | Vue SPA (`frontend/README.md`) |
| `model/` | Git submodule: the toxicity model, its training, and the scoring service (`model/README.md`) |
| `docs/` | Guides, specs, architecture and the prototype (`docs/README.md`) |
| `deploy/` | Host nginx site and `scripts/init.sh`; generated secrets (git-ignored) |
| `compose.yml` | Dev stack: postgres, migrate, app, worker, web, model (mock), mailpit, seed (profile `tools`) |
| `compose.prod.yml` | Single-server production stack |
| `Makefile` | Stack, database, codegen, checks and release images |

## Development

Needs Docker, and for running checks on the host Go 1.26, Node 24 and Python 3.11.

```sh
git clone --recurse-submodules <repo>   # or, in an existing clone: git submodule update --init
make init     # once: submodule, secrets, .env, backend/.env.local
make dev      # build + start everything, follow logs (Ctrl-C leaves it running)
make seed     # demo accounts and content (SEED_PASSWORD in .env)
```

| URL | |
|---|---|
| http://localhost:5180 | Frontend (Vite, hot reload) |
| http://localhost:8091/api/docs | API, Swagger UI for every module |
| http://localhost:8026 | Mailpit (invitation and reset emails) |
| http://localhost:8093 | Mock model service |

`make check` runs everything CI runs for the backend and frontend; the model service's tests run with
`pytest tests` in `model/`. Demo accounts and every other target are in
[Getting started](docs/guides/getting-started.md); releases follow [Deploying](docs/guides/deploying.md).
