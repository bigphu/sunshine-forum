# Sunshine Apartment Forum

A moderated forum for one building's verified residents. Every post and comment is screened before it goes
public, writers are warned before they submit, managers review what the filter holds back, and every
rejection can be appealed once.

The product is specified in [PRODUCT.md](PRODUCT.md), [SYSTEM_DESIGN.md](SYSTEM_DESIGN.md),
[COMMUNITY_RULES.md](COMMUNITY_RULES.md) and [DESIGN.md](DESIGN.md). The clickable prototype in `prototype/` is the
reference for every screen.

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

The AI model is **not connected yet**. A stand-in scorer behind `screening/scorer.Scorer` returns sample scores
(model version `stand-in-0.1 (điểm mẫu)`) so every flow can be demonstrated. Plugging in the real model is a
change inside the screening module only; see `backend/docs/screening.md`.

## Repository

Monorepo: the root owns the Docker stack, each app has its own folder and Dockerfile.

| Path | Holds |
|---|---|
| `backend/` | Go API, worker, migrations and seed. Module references in `backend/docs/` |
| `frontend/` | Vue SPA (`frontend/README.md`) |
| `docs/guides/` | Guides for running, operating and extending the system |
| `docs/` | The system documentation (`.docx`) and its UML diagrams |
| `prototype/` | The approved HTML prototype |
| `compose.yml` | Dev stack: postgres, migrate, app, worker, web, mailpit, seed (profile `tools`) |
| `compose.prod.yml` | Single-server production stack |
| `deploy/` | Host nginx site, `scripts/init.sh`; generated secrets (git-ignored) |
| `Makefile` | Stack, database, codegen and checks for both apps |

## Guides

| Guide | For |
|---|---|
| [Getting started](docs/guides/getting-started.md) | Run the stack, demo data and accounts |
| [Architecture](docs/guides/architecture.md) | Services, modules, events, how a post moves through the system |
| [Configuration and operations](docs/guides/operations.md) | Environment, `make` targets, checks, purges |
| [Adding a backend module](docs/guides/writing-a-module.md) | Checklist for a new module |
| [Deploying](docs/guides/deploying.md) | Release tag to server |

## Development

Needs Docker, and for running checks on the host Go 1.26 and Node 24.

```sh
make init     # once: secrets, .env, backend/.env.local
make dev      # build + start everything, follow logs (Ctrl-C leaves it running)
make seed     # demo accounts and content (SEED_PASSWORD in .env)
```

| URL | |
|---|---|
| http://localhost:5180 | Frontend (Vite, hot reload) |
| http://localhost:8091/api/docs | API, Swagger UI for every module |
| http://localhost:8026 | Mailpit (invitation and reset emails) |

Demo accounts and every other target are in [Getting started](docs/guides/getting-started.md).
