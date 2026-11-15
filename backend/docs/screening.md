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

Only `scorer.StandIn` exists today. It recognises a few sample words and returns scores marked
"stand-in-0.1 (điểm mẫu)". To connect the real service, add an HTTP scorer (POST the text, decode the
response, map errors to an error return; `/health` can back `ModelOnline`) and pass it as
`screening.Deps{Scorer: …}`. Nothing else changes. With a real scorer the outage simulation endpoint answers
`409 /errors/stand-in-only`.

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
