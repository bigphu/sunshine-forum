---
title: Getting started
description: Run the stack, load demo data, sign in with each role.
---

## Run everything in Docker

Needs Docker. For checks on the host you also need Go 1.26 and Node 24.

```sh
make init     # once: secrets in deploy/secrets/, .env, backend/.env.local
make dev      # build + start everything, follow logs (Ctrl-C leaves it running)
make seed     # demo accounts and content
```

| URL | |
|---|---|
| http://localhost:5180 | Frontend (Vite, hot reload) |
| http://localhost:8091/api/docs | API, Swagger UI for every module |
| http://localhost:8026 | Mailpit: invitation and password-reset emails land here |
| http://localhost:8093 | Mock model service (`/health`, `/mock/state`) |

Ports come from `.env` (`WEB_PORT`, `APP_PORT`, `DB_PORT`, `MAILPIT_PORT`, `MODEL_PORT`). The forum has its own Postgres on
5441 so it can run next to StoreIt.

Set `ADMIN_EMAIL` and `ADMIN_PASSWORD` in `.env` before the first start to get an administrator account on an
empty database. `make seed` creates one anyway.

## Run the Go apps on the host

Useful when debugging the backend. Start Postgres and Mailpit in Docker, then the API and the worker on the host:

```sh
docker compose up -d postgres mailpit
make -C backend migrate
make -C backend run-api      # 127.0.0.1:8091
make -C backend run-worker   # events, emails, rescans, purges
make -C backend run-mockmodel  # optional: the mock model service on :8093 (set SCREENING_MODEL_URL in .env.local)
npm --prefix frontend run dev   # :5180, proxies /api to :8091
```

Stop the `app`, `worker` and `web` containers first if they are running; they use the same ports.

## Demo data

`make seed` fills an empty database. It only adds content when the feed is empty, so running it twice does nothing.
To start over: `make db-reset CONFIRM=yes`, then `make seed`. Every account uses `SEED_PASSWORD` from `.env`.

| Email | Name | Role |
|---|---|---|
| `admin@sunshine.test` | Chủ tòa nhà (mẫu) | Administrator |
| `manager@sunshine.test` | Ban quản lý (mẫu) | Manager |
| `lan@sunshine.test` | Nguyễn Thị Lan (mẫu), A-1203 | Resident |
| `minh@sunshine.test` | Trần Văn Minh (mẫu), B-0807 | Resident |
| `hoa@sunshine.test` | Lê Thu Hoa (mẫu), A-0405 | Resident |

The content covers the main cases: a pinned management notice with an official reply, posts in every channel,
a comment the stand-in scorer holds as hate, and a marketplace post held by the blocklist ("vay nhanh", "lãi cực
thấp"). Start the worker so the seeded events reach the queue and the notification inbox.

## A five-minute tour

1. Sign in as **lan**. Write a comment containing an insult: the draft check warns before you send. Send it
   anyway: it is held and shows as pending, and it appears under **Yêu cầu của tôi**.
2. Sign in as **manager**. Open **Kiểm duyệt**: the held comment is under *Mức cao* with its text hidden. Reveal
   it, reject it under Rule 1, and see the message the writer will get.
3. Back as **lan**: a notification explains the decision. Open it and request a re-evaluation.
4. As **admin** (a different moderator than the one who decided), accept or decline the appeal in the queue.
5. Take the model offline: `curl -X PUT localhost:8093/mock/state -d '{"offline":true}'`. New posts are held
   under the outage policy and the composer says checking is slow. Send `'{}'` to bring it back; the worker
   rescans the held posts within 15 seconds. (On a host run without `SCREENING_MODEL_URL`, use the
   *Giả lập bộ lọc ngừng hoạt động* switch in **Quản trị → Bộ lọc AI** instead.)
6. **Quản trị → Nhật ký kiểm tra** shows every decision and setting change with pseudonymous user IDs.
