# System Design

**Status:** draft, awaiting review before the initial UI design begins. The first model is chosen but not yet trained, so anything that depends on its results is provisional.

Product requirements live in [PRODUCT.md](../../PRODUCT.md), and rules and moderation messages live in [COMMUNITY_RULES.md](COMMUNITY_RULES.md). The full backend architecture, as UML views in the Views and Beyond format, is Part A of [the system documentation](docs/Sunshine_Forum_System_Documentation.docx).

This design covers the simplified first release: a forum, the AI filter, and three roles. The backend isn't tied to a language.

## 1. Architecture

```text
Browser (Vue)
   |  HTTPS, session cookie, Server-Sent Events
   v
Host nginx (TLS, shared with StoreIt)
   |
   v
Backend (API and Worker modes)  -------------->  Python model service
   |                              HTTP, timeout under 200 ms
   v
PostgreSQL (forum data, audit log, background jobs)
```

- Vue talks only to the backend.
- The backend calls the Python model service, and only the backend writes to the database.
- The model service is separate, so if it fails, the backend can still apply the outage policy (NFR-4.2). It may run on another machine (see "Hosting").
- This is one backend and one model service, not microservices. The backend is one codebase that runs in two modes: API for requests, and Worker for background jobs.

## 2. Tech stack

| Layer | Choice | Why |
|---|---|---|
| Frontend | Vue 3, TypeScript, Vite, Vue Router, Pinia, vue-i18n | Single-page app, because the forum is login-only and doesn't need SEO. vue-i18n keeps text ready to translate. |
| Accessible components | Reka UI, with custom styling | Handles keyboard and screen reader behaviour for WCAG AA without imposing a visual style. |
| Backend | Open: TypeScript, JavaScript, Go, or another stack, built as one modular application | Whoever builds the backend chooses, as long as it provides the capabilities below. |
| Model service | Python with FastAPI and scikit-learn for the SVM; ONNX Runtime only if it is later replaced by a transformer | Scikit-learn is the simplest fit for a linear SVM. See Section 5. |
| Database | PostgreSQL | See Section 4. |
| Background jobs | A job queue on PostgreSQL, plus a transactional outbox | Rescans after outages, 90-day purges, penalty expiry, and notifications. No separate queue server is needed. |
| Real-time alerts | Server-Sent Events plus Web Push | Alerts only flow from server to user, so WebSockets aren't needed. |
| Sign-in | Secure HTTP-only, SameSite session cookies stored in PostgreSQL, with CSRF tokens | Unlike signed tokens, sessions can be revoked immediately, and role or penalty changes apply on the next request. |
| Deployment | Docker Compose on the VPS shared with StoreIt, behind the host's nginx | 99.5% uptime allows about 3.6 hours of downtime a month, so this can work with tested backups and monitoring. See "Hosting" below. |
| Testing | Vitest, Playwright with axe, backend integration tests against a real PostgreSQL, pytest, k6 | axe covers automated WCAG checks only, so manual testing is still needed. k6 tests the 100-user and speed targets. |

Current versions have not been checked. Choose the latest stable releases at setup.

**Left out for now:** Kubernetes, Kafka, Redis, and Nuxt. They aren't needed at this scale and can be added if testing shows a need.

### Backend: what any stack must provide

The options below are examples to evaluate, not decisions. Check current versions and maintenance status before choosing.

| Capability needed | TypeScript or JavaScript | Go |
|---|---|---|
| HTTP framework with middleware and streaming | NestJS, Fastify, or Express | chi |
| Request validation and OpenAPI | Zod or Ajv, with an OpenAPI generator | oapi-codegen |
| SQL with transactions and migrations | Prisma, Drizzle, or Kysely, with node-postgres | pgx, sqlc, goose |
| Job queue on PostgreSQL | pg-boss or Graphile Worker | River |
| Transactional outbox | Own table and worker loop | Own table |
| Password hashing | An Argon2 library | An Argon2 library |
| Field encryption | Node crypto: AES-256-GCM, with HMAC-SHA-256 for lookups | Standard library crypto |
| Server-Sent Events | Framework support or a plain response stream | Standard library |
| Web Push | web-push | webpush-go |
| Email | Provider SDK or HTTPS API | Provider SDK or HTTPS API |
| Language detection | franc or similar | whatlanggo or similar |

Whichever language detection library is chosen must be tested on Vietnamese without diacritics.

**Modules**
- **identity:** invitations, password reset, sessions, CSRF protection, users, and roles.
- **content:** channels, posts, two-level comments, official replies, and the draft check.
- **screening:** the blocklist, language check, model call, thresholds, and outage policy.
- **moderation:** the queue, decisions, bulk actions, redaction, and removing published content.
- **appeals:** one re-evaluation request per decision, and its review.
- **penalties:** confirmed violations, the rolling count, and read-only timeouts.
- **notifications:** the in-app inbox, Server-Sent Events, Web Push, and email.
- **administration:** settings, analytics, the audit trail, and the retraining export.
- **privacy:** field encryption and the 90-day purge.
- **platform:** HTTP plumbing, data access, jobs, the outbox, configuration, and logging.

**Rules that hold in any stack**
- **Sign-in:** load the session and user on every request, and don't keep permissions in a token. A read-only timeout must apply on the next request.
- **Audit log:** write audit rows in the same transaction as the moderation action, not through an event subscriber. Subscribers run later and can be delivered more than once.
- **90-day purge:** event and outbox rows must hold IDs only, never post text or personal data, and the purge job must clean them too.
- **Model calls:** put a hard timeout on every call, using an abort signal or equivalent.
- **Blocking work:** in Node.js, password hashing and purges must not block request handling.
- **Logs:** never log post text, drafts, or personal data.

**Settings to get right**
- **Trusted proxies:** set these to the forum's Docker network, which host nginx connects through. If they're empty, every resident appears to come from one IP address and they all share one rate limit.
- **API docs:** keep any docs page off in production, because it would be public.
- **Streaming:** the Server-Sent Events route needs no write timeout, and streaming through the middleware chain needs testing.
- **Rate limits:** apply them to sign-in, password reset, and pre-send checks.

**If Go is chosen:** [goat](https://github.com/bigphu/goat) was the earlier proposal. It describes itself as an early prototype, has no tests, and requires Go 1.26 or later. Pin the version, write integration tests for `database`, `jobs`, and `events`, and don't use its `jwt` package, because its tokens keep permissions until they expire. Its `platform.events` table also keeps events permanently, so the purge job must clean it.

### Hosting

The forum runs at `sunshine-forum.bigphu.io.vn` on StoreIt's VPS: 4 vCPU, 4 GB RAM, 40 GB SSD, Ubuntu 24.04. The limits below come from the [StoreIt runbook](https://claude.ai/artifact/Ag9pe1fNXWoLuwdBXdmPrN#setup).

**Sharing the server**
- **TLS:** host nginx already holds ports 80 and 443, so the forum adds its own nginx site instead of running Caddy.
- **Separate stack:** the forum gets its own Compose project and its own PostgreSQL container, so its purges, backups, and upgrades don't touch StoreIt's data. It costs some extra memory.
- **Port and network:** StoreIt uses `127.0.0.1:8080` and the `172.30.0.0/24` network, so the forum needs a different port and subnet.
- **Server-Sent Events:** the forum's nginx site must turn off response buffering and allow long read timeouts on the stream, or alerts arrive late or the connection drops.

**Provider network limits**

| Limit | Effect on the forum |
|---|---|
| Inbound traffic from outside Vietnam is blocked | Residents abroad can't open the site, and uptime monitors abroad report it as down. Monitor from Vietnam or from the server. |
| Let's Encrypt can't reach port 80 | Certificates use the Cloudflare DNS method, as StoreIt's do. Keep the `sunshine-forum` DNS record as DNS only, so rate limits see each resident's real IP address. |
| Outbound SMTP is blocked | Invitation and password reset emails go through Resend's API. |
| Docker Hub is blocked | Images come through `mirror.gcr.io`, which only keeps popular images. |

**Model service on another machine**

The SVM (Section 5) should fit on the shared server, so another machine is only likely if it is replaced by something larger. The backend reads the service's address from configuration, so this doesn't change the code. If it does run elsewhere:
- **Speed:** the 200 ms limit (NFR-1.2) includes the network trip both ways. Measure this from the VPS before relying on it, because it leaves less time for scoring.
- **Reachability:** the VPS must be able to reach that machine, given the provider's blocks. Test the actual port from the VPS, and if the machine is at home or behind a router, plan a stable private connection such as a VPN.
- **Privacy:** post text and drafts now leave the server. Use TLS and service authentication, keep the service off the public internet, and make sure it doesn't log the text.
- **Availability:** a second machine and a network link can each trigger the outage policy (NFR-4.2). Under the default fail closed, every new post waits in quarantine until the model returns, so that machine's uptime directly affects the forum.

**Risks**
- **Memory:** StoreIt uses 400–700 MB, and building images takes about 1.5 GB more for a minute or two. The forum adds the backend and PostgreSQL, and the backend's memory use depends on the stack chosen. The SVM should need only a few hundred MB (Section 5). None of this has been measured. A larger model may need its own machine.
- **Shared downtime:** a server failure, or a StoreIt build that uses up memory, also affects the forum. Avoid running builds at busy times.
- **Disk:** about 25 GB was left for data before the forum, and both apps' data, images, and dumps now share it.
- **Backups:** StoreIt's backup script covers only its own database. The forum needs its own nightly dump, copied off the server.

## 3. How the design meets specific requirements

| Requirement | Approach |
|---|---|
| Immutable audit log (FR-5.3) | Append-only table, with update and delete permissions removed from the app's database role. Rows are written in the same transaction as the moderation action. |
| Encrypted personal data (NFR-3.1) | Encrypt fields in the application, with a keyed lookup hash for anything that must be searched, such as email. |
| Settings without a restart (FR-5.1) | Store thresholds in PostgreSQL and use `LISTEN/NOTIFY` to refresh the backend's cached copy. |
| 200 ms scoring (NFR-1.2) | The backend enforces a timeout under 200 ms, and a timeout triggers the outage policy. Whether the model can meet this is unconfirmed. |
| Model outage (NFR-4.2) | The backend applies the Admin policy. The default is fail closed, and background jobs rescan on recovery. |
| Pre-send check (FR-3.1) | Run the blocklist instantly in the browser, and send the draft to the model after a short typing pause (debounced). Drafts are not stored. |
| Retraining data (US-6.4) | Minimal export, described below. No separate store and no change to NFR-3.3. |
| False negatives (FR-5.3) | A manager removes published content by choosing a rule. This records a confirmed violation and logs a false negative, and the resident can appeal it. Without resident reporting, this is the only way to record what the filter missed. |
| Notifications (FR-2.2) | In-app notifications must work on their own. Email carries only invitations and password resets. Web Push on iOS only works if the resident adds the site to their Home Screen, which many elderly residents may not do. Before relying on push, confirm the server can reach the browsers' push services. |

The approved FR-3.1 wording in [SPEC_CHANGES.md](SPEC_CHANGES.md) has the model run on the server, because the model team confirmed it can't run in the browser.

**Retraining export**
- **Trigger:** an Admin runs it when the model team needs data. It reads from the content table, because the audit log holds no text.
- **What it includes:** overturned AI decisions, including published content a manager removed, 30–90 days after the decision, so the appeal period has closed.
- **What it leaves out:** Rule 5 cases and content the author has deleted.
- **What each sample holds:** the text, final label, rule, AI score, and model version. It has no user or post IDs. Phone numbers, emails, and similar patterns are replaced automatically.
- **Deletion:** the model team uses the file for a training or evaluation run, then deletes it before the originals' 90 days are up.
- **Trade-offs:** examples don't build up over time, and old evaluations can't be rerun. To capture every correction, export at least every 60 days. Deletion relies on the model team following this process.

## 4. Data storage: PostgreSQL, no NoSQL

**Decision:** use PostgreSQL as the only database for now.

**Expected volume:** about 100 residents are expected. Even if all of them wrote 10 posts or comments a day, which is a deliberately high estimate, that is about 365,000 rows a year, likely under 1 GB including indexes. That is well within what one modest PostgreSQL server can handle.

**Why relational fits**
- **Linked updates:** rejecting a post changes its status, adds a strike, may apply a timeout, writes to the audit log, and queues a notification. These can all succeed or fail together in one transaction.
- **Connected data:** users, roles, channels, posts, appeals, and audit records reference each other, and the moderation queue and analytics query across them.
- **Flexible fields:** `JSONB` columns can hold data whose shape may change, such as model scores per label.

**How to keep it fast**
- **Comments:** store `post_id` and `parent_id`, and index `(post_id, created_at)`. With nesting capped at two levels, one simple query loads a whole thread.
- **Paging:** use keyset pagination instead of `OFFSET`.
- **Feeds and the queue:** use partial indexes, such as on `(channel_id, created_at) WHERE status = 'published'`, with a separate one for the moderation queue.
- **Counts:** store comment counts on the post instead of counting rows on every page load.
- **Connections:** use a connection pool.
- **Audit log:** partition by month if it grows large.

**When to add something else** (PostgreSQL stays the main database in every case)

| If this happens | Add |
|---|---|
| Residents can upload images or files | S3-compatible object storage; PostgreSQL stores only the links |
| Vietnamese search isn't good enough | OpenSearch or Meilisearch, holding a copy of the data |
| Load tests show repeated reads are the bottleneck | Redis cache |
| Read traffic outgrows one server | A read replica |

The specification doesn't currently require search. If it's added, PostgreSQL has no built-in Vietnamese language support, though `unaccent` and `pg_trgm` can handle basic search, including text typed without diacritics.

**How to confirm it:** before launch, fill a test database with several years of simulated data and run the k6 tests against it to check the 2-second page load target (NFR-1.1).

## 5. Model integration

**Status: the first model is decided, and the model team answered the design questions on 2026-11-09 (below). The model has not been delivered, so the stand-in stays in use.**

Until the model is ready, the backend calls a stand-in scoring service that follows the same contract and returns scores clearly marked as fake. This lets the warnings, quarantine queue, and appeals be built and demonstrated.

### First model: linear SVM

**Status: accepted. The course allows an SVM, and the project only needs a simple classifier. It is not yet trained or tested.**

- **Design:** a linear SVM on TF-IDF character and word n-grams, trained on ViHSD (below), which scores `clean`, `offensive`, and `hate`.
- **Input:** text goes through the FR-3.4 normalization, then is scored sentence by sentence, using the highest score (FR-3.5).
- **Scores:** raw SVM scores are not probabilities, so calibrate them on held-out data for the Admin thresholds (FR-5.1), and recheck calibration after every retrain.
- **Why start here:** it should score well within 200 ms on a CPU and need only a few hundred MB, so it should fit on the shared server. Its weights can show which words drove a score, and it is cheap to retrain. These are estimates to measure.
- **Known weakness:** it mostly sees word patterns, not meaning, so expect it to miss polite hostility and sarcasm, and to flag ordinary complaints that use insult-like words. A wider "Pending review" band can offset this, at the cost of more manager work.
- **Keeping it replaceable:** it uses the same service contract as any other model, including `model_version`, so a transformer can replace it without changing the backend.

**Before launch:** test precision and recall per label on Vietnamese text it hasn't seen, including longer posts and ordinary complaints. The model team will only measure ViHSD's test set, so the forum team tests longer posts and complaints on its own set (see the decisions below). Set the pass mark before seeing the results. If it fails, first improve the training data and features. A small transformer such as PhoBERT is the fallback, provided it still meets the 200 ms limit.

### Training data: ViHSD

[ViHSD](https://huggingface.co/datasets/uitnlp/vihsd) has 33,400 Vietnamese Facebook and YouTube comments, split 7:1:2 into train, dev, and test, each labelled `CLEAN`, `OFFENSIVE`, or `HATE`. Names are replaced with placeholders, and teencode and slang are kept. Access is gated, so each person needs a signed-in Hugging Face account. [The authors' paper](https://arxiv.org/abs/2103.11528) has the definitions and results used here.

**Label mapping (decided):** the model keeps ViHSD's labels, and [COMMUNITY_RULES.md](COMMUNITY_RULES.md) has the full rule map.
- **`OFFENSIVE`:** harassing or vulgar content with no specific target. This names R4, and it is the only model label the auto-delete threshold (FR-5.1) applies to, when an Admin has turned it on.
- **`HATE`:** abusive content aimed at an individual or a group. This mixes R1, R2, and R3, so it goes to quarantine whatever its score, and a manager chooses the rule, as FR-4.5 already requires. The model never auto-deletes it.
- **Effect:** every `HATE` flag needs a manager, and pre-send warnings for it are more general.

**What the paper means for testing**
- **Imbalance:** the data is heavily skewed to `CLEAN`, so accuracy is misleading. Use per-label precision and recall, plus macro-F1.
- **Benchmark:** the paper's best model, a BERT variant, reached 86.88% accuracy but only 62.69% macro-F1. It reports no SVM baseline, so expect the SVM's results to need measuring, and it is unlikely to beat this by much.
- **Your own test set:** ViHSD comes from public social media, so also test on hand-written examples of building discussion, including ordinary complaints.

**Still to confirm**
- **Licence:** the model team and the Hugging Face page both say MIT, but read the terms shown when requesting access. The paper names no dataset licence.
- **Pass mark:** the target precision and recall for each label, to be discussed later.

### Model team answers

Received 2026-11-09. Their general note: ViHSD has only the labels `CLEAN`, `OFFENSIVE`, and `HATE`, so the specification should follow those three.

| # | Topic | Answer | Effect on this design |
|---|---|---|---|
| 1 | Outputs | Scores for `clean`, `offensive`, and `hate` only. They read the rule in FR-4.5 as "offensive or hate". | Matches the label mapping. See the decisions for FR-4.5. |
| 2 | Scores | Calibrated 0–1 and comparable. | The Admin thresholds can use them directly (FR-5.1). |
| 3 | Speed | Under 200 ms on CPU, no GPU needed. Measurements to follow. | Supports the shared server, but not proven yet. |
| 4 | Pre-send check | Cannot run in the browser. FR-3.1 should say the client calls an API before sending. | Matches the FR-3.1 change. The browser calls the backend API, and the backend calls the model. |
| 5 | Text preparation | The service handles it, including word segmentation. No language detection. Limit of 5,000 characters. | The backend handles other languages and long text. See the decisions below. |
| 6 | Locating the problem | Yes, by sentence or phrase. Sentence-level accuracy in long posts is unverified. | Treat highlights as a guide. Decisions use the overall score. |
| 7 | Service contract | The proposed format is accepted, with labels `clean`, `offensive`, and `hate`, plus `/health` and `model_version`. | See the agreed format below. Error codes are not settled. |
| 8 | Delivery | Docker and FastAPI, run by the team. No zero-downtime version swap. | A swap counts as an outage: the NFR-4.2 policy holds content and rescans it afterwards. Swap at quiet times. |
| 9 | Timeline | To be announced. | Forum work continues on the stand-in. |
| 10 | Accuracy | Per-label precision and recall on ViHSD's test set. Nothing separate for longer posts or ordinary complaints, because there is no data. | The forum team tests these. See the decisions below. |
| 11 | Starting thresholds | Sent after training. | Based on ViHSD only, so review them against building content. |
| 12 | Training data | ViHSD (UIT), MIT licence, labels as defined in the paper. | Matches this section. |
| 13 | Retraining | Manual, by the Data Scientist, once there is enough new data, using decisions moderators overturned. Personal data is removed automatically on export. | Keeps the minimal export in Section 3. Frequency and the deletion deadline are unconfirmed. |

### Decisions from these answers

Decided 2026-11-09.

- **Other languages (FR-3.4):** the model has no language detection, so a Vietnamese-only model could score foreign-language abuse as clean. A small language check in the backend runs before scoring and sends non-Vietnamese text to "Pending review" without scoring it. The library is still to be chosen, and it must not treat Vietnamese without diacritics as another language.
- **Length limit:** the model takes 5,000 characters, and posts have no limit yet. Post bodies and comments are limited to 5,000 characters, so nothing goes unscored (proposed FR-2.7). The backend enforces this as well as the composer, counts characters rather than bytes, and scores a post's title and body in separate requests.
- **FR-4.5 wording:** the model team reads the violated rule as a model label. The seven community rules stay, because "hate" alone does not tell a resident what they did. `offensive` names Rule 4, and a manager chooses among Rules 1 to 3 for `hate`.
- **Auto-delete (FR-5.1):** off by default. A linear SVM sees word patterns, not meaning, so removing content with no human check is the riskiest use of it. Until an Admin turns it on, `offensive` content above the hold threshold goes to a manager. Turn it on only after testing on ViHSD and the building test set.
- **Building test set:** the model team cannot measure longer posts or ordinary complaints, so the forum team hand-writes and labels a small Vietnamese set of building discussion. It counts toward the pass mark. Its size and owner are still to be set.

### Still to ask the model team

- **Speed:** measurements for a long post scored sentence by sentence, with up to 100 active users, plus memory use and start-up time.
- **Scores:** do the three scores add up to 1? If they do, a text that is both offensive and hateful splits its score between labels, which affects the thresholds.
- **Text preparation:** does the service cover missing diacritics, teencode, and spaced-out letters, as FR-3.4 requires?
- **Locating:** does the response give sentences only, or exact phrases? The prototype masks phrases.
- **Errors:** the error codes, and what happens above 5,000 characters.
- **Running it:** who "the team" is in production, and who swaps versions.
- **Retraining:** how often they need an export, whether they delete it before the 90-day limit, and whether one building's corrections are enough to retrain or only to tune thresholds.

### Agreed response format

Accepted by the model team on 2026-11-09. The service also exposes `/health`. Scores below are examples.

```json
{
  "model_version": "2026.11.1",
  "labels": { "clean": 0.05, "offensive": 0.40, "hate": 0.91 },
  "sentences": [{ "start": 0, "end": 42, "top_label": "hate", "score": 0.91 }]
}
```

## 6. Open items

- **Model team:** the remaining questions in Section 5, and the delivery date.
- **Backend stack:** choose it from the options in Section 2.
- **Language check:** choose the library, and test it on Vietnamese without diacritics.
- **Building test set:** its size and who writes it.
- **Training data:** read ViHSD's access terms.
- **Pass mark:** the SVM's accuracy targets, to be discussed later.
- **Model hosting:** the shared server if the SVM fits, which still has to be measured. Otherwise, where the other machine is.

The model items above are paused (see Section 7).

## 7. Current focus: forum core

Model work is set aside so the core forum can be built first. The forum uses the stand-in scoring service from Section 5 until a trained model replaces it.

**Paused:** training the SVM, checking ViHSD's access terms, the pass mark and accuracy testing, the remaining Section 5 questions, and measuring whether the model fits on the VPS.

**What the core relies on from the model**
- **Response format:** `model_version`, `clean`, `offensive`, and `hate` scores, and per-sentence scores.
- **Outcomes:** `offensive` names Rule 4, and `hate` goes to quarantine for a manager to choose the rule.
- **Failure handling:** a timeout or error triggers the outage policy (NFR-4.2). The stand-in can fail on purpose, so fail closed and the rescan on recovery can be tested without a model.
- **Address:** the model service's address stays in configuration, so moving it to another machine doesn't change the core.

**Build order**
1. **Foundations:** the backend project, PostgreSQL, migrations, Docker Compose, and a health check.
2. **Sign-in:** invitations, session cookies, CSRF protection, and roles.
3. **Forum:** channels, posts, two-level comments, and the official reply marker.
4. **Screening:** the stand-in service, the blocklist, the language check, thresholds, and the outage policy.
5. **Moderation:** the quarantine queue, decisions, removing published content, the audit log, penalties, and appeals.
6. **Administration:** settings refreshed without a restart, analytics, the audit view, and the retraining export.
7. **Notifications:** in-app alerts, Server-Sent Events, invitation and password reset emails, then Web Push.
8. **Privacy:** field encryption and the 90-day purge.
9. **Deployment:** the forum's nginx site, certificate, backups, and monitoring on the shared VPS.

The Vue frontend and OpenAPI spec grow alongside steps 2 to 7.
