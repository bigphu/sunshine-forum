# Screening module: reference

Decides whether text is published, held for review, or (only when an Admin enabled it) auto-deleted.
`content` calls it before every save, including edits. COMMUNITY_RULES.md Part 3 is the rule map.

## Pipeline (`service.Screen`)

1. **Blocklist.** Terms are folded (lowercase, no diacritics, `vntext.Fold`) and matched as whole phrases, so
   text typed without diacritics still matches. A match holds the content with the term's rule.
2. **Language.** `domain.IsVietnamese`: any Vietnamese diacritic means Vietnamese; text without diacritics is
   foreign only when it has clearly more English stopwords than Vietnamese ones. Foreign text is held
   unscored, names no rule and counts as no violation (FR-3.4). The library choice in SYSTEM_DESIGN is still
   open; replace this heuristic behind the same function and keep `domain_test.go` passing.
3. **Model**, with a 200 ms timeout (`service.ModelTimeout`, NFR-1.2). Scores per sentence, highest wins.
4. **Thresholds** (`domain.Decide`): `hate` at or above the hold threshold is always held so a manager picks
   Rule 1, 2 or 3, and wins over `offensive`. `offensive` names Rule 4 and is held, or auto-deleted at the
   auto-delete threshold when enabled. Below the threshold it is published.

Post titles and bodies are scored separately and the worse result wins (`Worse`).

**Outage (NFR-4.2).** Any scorer error or timeout applies the policy: `fail_closed` (default) holds the
content; `fail_open` publishes it after the blocklist. Either way `NeedsRescan` is set and content's rescan
job scores it again when the model answers. Comments in Thông báo chung are held even under fail open.

## The model plug (`scorer/`)

`scorer.Scorer` returns the contract agreed with the model team (SYSTEM_DESIGN §5): `model_version`,
calibrated `clean` / `offensive` / `hate` scores and flagged sentences with character offsets.

Which scorer runs depends on `SCREENING_MODEL_URL` (`screening.Config`):

| Setting | Scorer | Used for |
|---|---|---|
| empty | `scorer.StandIn`, in process. Scores marked "stand-in-0.1 (điểm mẫu)" | Host runs, tests, production until the model ships. Admins can simulate an outage from the filter page |
| a URL | `scorer.HTTP`, calling the model service | The dev stack (pointing at the mock), and production once the real service is deployed |

`scorer.HTTP` speaks the contract in [`scorer/model-service.yaml`](../internal/screening/scorer/model-service.yaml):
`POST /score {"text"}` and `GET /health`. It keeps connections alive, honours the 200 ms deadline, and
rejects anything outside the contract: a missing label, a score outside 0–1, an empty `model_version`, or
sentence offsets outside the text. Every failure becomes an error, so the outage policy applies. `ModelOnline` and
the status endpoint call `/health` and report the service's `model_version`.

With a real scorer the outage simulation endpoint answers `409 /errors/stand-in-only`; simulate outages on the
service instead.

**Mock service (`cmd/mockmodel`).** Implements the same contract with StandIn's sample scores and version
"mock-0.1 (điểm mẫu)". The dev stack runs it as the `model` service, and `make run-mockmodel` runs it on the host at
`127.0.0.1:8093`. Its test runs it against `scorer.HTTP`, so the mock and the client cannot drift apart. To simulate
trouble without stopping it:

```sh
curl -X PUT localhost:8093/mock/state -d '{"offline":true}'   # /score and /health answer 503
curl -X PUT localhost:8093/mock/state -d '{"delay_ms":300}'   # slower than the 200 ms deadline
curl -X PUT localhost:8093/mock/state -d '{}'                 # back to normal
```

Held content is rescanned by the worker within 15 seconds of the service answering again.

**Real service.** The FastAPI service lives in the `model/` submodule (`service/app.py`, `Dockerfile`). It scores
sentence by sentence with the calibrated SVM and passes the same contract checks. Production runs it as the `model`
service (`make model-image`). Without the trained model, `python src/toy.py` in `model/` builds a toy model to run it
locally; to try it in the dev stack, run its image on the `sunshine-forum_default` network and point
`SCREENING_MODEL_URL` at it.

## Settings without a restart (FR-5.1)

Settings, per-channel thresholds and the blocklist live in `screening.*`. Triggers send
`NOTIFY screening_settings`; `Module.Watch` (run by server and worker) invalidates the in-memory snapshot.
The snapshot also expires after 30 seconds in case a notification is missed.

| Setting | Default |
|---|---|
| Hold threshold | 0.5 |
| Auto-delete | off (PRODUCT.md: until the model is tested on building content) |
| Auto-delete threshold | 0.95 |
| Outage policy | `fail_closed` |

## Draft check (FR-3.1)

`POST /screening/check` returns warnings with the exact wording from COMMUNITY_RULES.md Part 2: the
rule-specific warning for blocklist matches, the Rule 4 warning for `offensive`, the general Rules 1–3 warning
for `hate`, and the Rule 5 warning for phone numbers. Drafts are not stored and warnings never count. It is
rate-limited to 30 requests, refilling 5 per second, per IP. `GET /screening/blocklist` gives the browser the
terms so it can warn instantly while typing.

## API (`/api/v1`)

| Method and path | Permission |
|---|---|
| `GET /rules` | public |
| `POST /screening/check`, `GET /screening/status`, `GET /screening/blocklist` | `screening.check` |
| `GET/PUT /screening/settings`, `PUT /screening/channels/{id}/threshold`, `POST /screening/blocklist`, `DELETE /screening/blocklist/{id}`, `PUT /screening/stand-in` | `screening.settings.manage` |

## Contract

`contract.Screener` (`Screen`, `Worse`, `ModelOnline`), the outcome and reason constants, `MaxChars`
(5,000, FR-2.7) and `Rules`, the seven community rules every module uses.
