---
title: Adding a backend module
description: A checklist for a new module in the modular monolith.
---

Use `penalties` (small) or `moderation` (events, appeals, audit) as the pattern. goat's `docs/platform.md` is the
reference for the platform packages.

## Checklist

1. **Layout**: `backend/internal/<module>/` with `domain`, `service`, `repository`, `handler`, `contract`, and a
   root package exposing `New(Deps)` and a `Module` with `Mount`, `Subscribe`, `RegisterWorkers`,
   `PeriodicJobs` and `APIDoc` as needed.
2. **Migration**: `make -C backend migrate-new name=<module>_<change>`. Create the module's schema, insert its
   permission codes into `identity.permissions` and grant them to the roles, **including admin**.
3. **Queries**: SQL in `repository/queries/*.sql`, then `make generate`.
4. **API**: `handler/openapi.yaml` (reuse `api/common.yaml` for `Problem` and `ID`), an `oapi.yaml` for codegen,
   then `make generate` and `npm --prefix frontend run gen:api`. Mount with
   `{web.ValidateRequests(spec, base), <ForOperations overrides>}`; the last entry runs first. Wrap operation
   lists in `web.MustOperations`. Sessions are already on the context from the root router.
5. **Services**: every public method starts with `auth.Require(ctx, "<module>.<action>")`.
6. **Transactions**: in repositories, with `database.WithTx`. Append outbox events and enqueue jobs in the same
   transaction. Services never see a `pgx.Tx`.
7. **Other modules**: depend on their `contract` interfaces through `Deps`, or subscribe to their events. Add the
   module to the depguard rules in `backend/.golangci.yml`.
8. **Events**: define payload types in `contract/events.go`, register handlers in `Subscribe`, and make every
   handler idempotent. List them in `backend/docs/events.md`.
9. **Config**: the module owns its environment names and a `Validate()`. Each binary lists only the blocks it uses
   in `cmd/<bin>/config.go`.
10. **Wiring**: `cmd/server` (routes, API docs), `cmd/worker` (subscribers, jobs), and `cmd/seed` if demo data
    needs it.
11. **Personal data**: encrypt it with `internal/privacy`, never put it in events or logs, and give it a purge.
12. **Tests**: unit tests next to the code; database tests as `*_db_test.go` using `internal/testdb`.
13. **Docs**: `backend/docs/<module>.md`, and the guides if people see the change.

## Conventions

- Code comments and package docs are in Vietnamese.
- Files use LF line endings.
- Periodic River jobs must not set `UniqueOpts{ByArgs: true}`; a completed unique job blocks every later run.
