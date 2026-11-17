# Frontend

Vue 3 single-page app for the Sunshine Apartment Forum. The screens are a port of the approved prototype
(`../docs/prototype/`); `../DESIGN.md` is the authority when they disagree.

## Commands

```sh
npm install
npm run dev        # http://localhost:5180, proxies /api to http://127.0.0.1:8091 (API_URL to change it)
npm run check      # vue-tsc, Vitest, production build (what CI runs)
npm run gen:api    # regenerate src/lib/api/*.d.ts from the backend OpenAPI specs
```

In Docker the `web` service of `../compose.yml` runs the `dev` target of the `Dockerfile`; the `prod` target is
nginx serving `dist/` and proxying `/api` to `app` (`docker/nginx.conf`).

## Stack

| | |
|---|---|
| Vue 3, TypeScript, Vite | `<script setup>` components |
| Vue Router | History mode, Vietnamese paths as in the prototype (`/kenh/:slug`, `/t/:id`, `/kiem-duyet/:case?`, `/quan-tri/…`) |
| TanStack Vue Query | Every server read; mutations call the API client and invalidate keys |
| Pinia | Only the session store (`lib/auth.ts`) |
| openapi-fetch | One typed client per backend module (`lib/api/client.ts`) |
| Reka UI | Popover menus only (account menu, feed filter, queue display options) |
| `@fontsource/be-vietnam-pro` | The brand font, self-hosted |

There is no component library. Styling is the prototype's stylesheet, copied unchanged to `src/app/forum.css`, so
the class names in templates match the prototype markup. Additions the real app needs (Reka's popper wrapper, the
busy spinner) are in `src/app/app.css`. Dialogs are native `<dialog>` elements (`components/AppDialog.vue`), as in
the prototype: focus is trapped, Esc closes, focus returns to the opener.

## Layout

```
src/
  app/           AppLayout (top bar, sidebar, phone bar), router, CSS
  components/    Shared pieces: Icon, Avatar, AppDialog, PageShell, DraftWarning, SaveFailure, ReadOnlyCard…
  composables/   useDraftCheck (pre-send warning)
  features/
    auth/        Sign in, forgot password, set password (invitation and reset links)
    forum/       Feed and channels, thread and comments, compose and edit, search
    me/          Notifications, my requests, appeal, rules, settings, not found
    moderation/  Queue, case view, bulk reject (cases.ts holds the queue logic, with tests)
    admin/       Home, users and roles, AI filter, blocklist, analytics, audit log, export
  lib/
    api/         client.ts and generated *.d.ts (never edit those by hand)
    auth.ts      Session store: current account, permissions, read-only standing
    queries.ts   Shared queries and the notification stream
    forum.ts     Static text and channel data from the prototype and COMMUNITY_RULES.md
    format.ts    Dates, numbers, Vietnamese folding
    types.ts     Short names for the generated schemas
```

## Talking to the API

- Sessions are cookies. `lib/auth.ts` keeps the CSRF token from `/me` or sign-in, and the client middleware adds
  `X-CSRF-Token` to every write. A 401 sends the user to `/dang-nhap?next=…`.
- Errors are problem+json. The client throws `ApiError(status, detail, type, fields)`; `errorText(e)` gives the
  Vietnamese message to show. Network failures have status 0.
- Permissions come from `/me` (`auth.can('moderation.remove')`). They only hide controls; the backend checks
  every call.
- New notifications arrive over Server-Sent Events (`/api/v1/notifications/stream`); `useNotificationStream`
  invalidates the inbox and the queue.

## Conventions

- UI text is Vietnamese and comes from the prototype or `docs/specs/COMMUNITY_RULES.md`. Code comments are Vietnamese.
- Every page heading has `id="focus-t"`; the router focuses it after navigation for screen readers.
- Loading shows skeletons or "Đang tải…"; a failed region shows `LoadError` with a retry and leaves the rest usable;
  a failed send keeps the text and shows `SaveFailure`.
- Use `PageShell` for one-column pages and `AdminPage` for admin pages.
