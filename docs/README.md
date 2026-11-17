# Documentation

Where to look, by what you need. `PRODUCT.md` and `DESIGN.md` stay at the repository root because the design
tooling reads them from there.

## Run and operate the system (`guides/`)

| Guide | For |
|---|---|
| [Getting started](guides/getting-started.md) | Run the stack, demo data and accounts, a five-minute tour |
| [Architecture](guides/architecture.md) | Services, modules, events, how a post moves through the system |
| [Configuration and operations](guides/operations.md) | Environment, `make` targets, checks, background jobs, retention |
| [Adding a backend module](guides/writing-a-module.md) | Checklist for a new module |
| [Deploying](guides/deploying.md) | First install and each release on the server |

Developer references live next to the code: `backend/docs/` (one file per module, plus events),
`frontend/README.md`, and `model/README.md`.

## What the product must do (`specs/`)

| File | Holds |
|---|---|
| [../PRODUCT.md](../PRODUCT.md) | Purpose, users, requirements (FR/NFR), first-release scope |
| [specs/SYSTEM_DESIGN.md](specs/SYSTEM_DESIGN.md) | Architecture decisions, the model contract (§5), hosting |
| [specs/COMMUNITY_RULES.md](specs/COMMUNITY_RULES.md) | The seven rules and every moderation message, word for word |
| [specs/UI_DESIGN_SPEC.md](specs/UI_DESIGN_SPEC.md) | Screen-by-screen behaviour, states and accessibility |
| [specs/SPEC_CHANGES.md](specs/SPEC_CHANGES.md) | Where this project departs from the original specification |
| `specs/copy-deck-vi.xlsx` | Vietnamese interface copy |

## How it looks (`../DESIGN.md`, `prototype/`)

[../DESIGN.md](../DESIGN.md) is the design system and the authority for screens. `prototype/` is the approved
clickable HTML prototype (`forum.html`, `moderation-dashboard.html`); the app reuses its stylesheet.

## Architecture documentation (`architecture/`)

`Sunshine_Forum_System_Documentation.docx` with its UML diagrams in `architecture/diagrams/` (editable `.drawio`,
plus `.png` and `.svg` exports): context, module views, data model, component-and-connector and deployment.
