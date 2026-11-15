---
target: prototype/forum.html
total_score: 27
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
target_identity: "file:C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
target_fingerprint: "sha256:232e3bd1b2111d55dea86635f64a6fdef5bb58b571b6047b3c374fe5222f7ec8"
target_path: "C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
timestamp: 2026-11-07T14-38-52Z
slug: prototype-forum-html
---
Method: dual-agent (A: design review sub-agent · B: detector and browser sub-agent)

# Final score: prototype/forum.html

27/40, Acceptable, up from 23 and 25. No P0. Neither reviewer used a real screen reader, phone or on-screen keyboard.

## Design Health Score
| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 3 | Pending, receipt, filter bar and stepper clear; held comments only findable in their thread. |
| 2 | Match System / Real World | 3 | Warm Vietnamese, but "1 giờ" reads as a clock time. |
| 3 | User Control and Freedom | 3 | Drafts survive and Esc works; reply "Hủy" deletes a draft; one-shot appeal has no review step. |
| 4 | Consistency and Standards | 3 | "Minh bạch" and "Quy tắc" share a page; warning and pending share one box style. |
| 5 | Error Prevention | 3 | Warning excellent; "a" is accepted as an appeal reason; composer lacks the privacy reminder. |
| 6 | Recognition Rather Than Recall | 2 | Up-arrow explained only by tooltip; appeal shows the rule title but not its text. |
| 7 | Flexibility and Efficiency | 2 | Sort, filter and drafts exist; no anchors to page sections. Search is scope. |
| 8 | Aesthetic and Minimalist Design | 3 | Disciplined; rail repeats feed posts; urgent thread repeats its action sentence. |
| 9 | Error Recovery | 3 | Errors specific, inline, take focus; appeal reason disappears after submission. |
| 10 | Help and Documentation | 2 | Rules collapsed by default; pending and appeal states don't link to them; emergency numbers not tappable. |
| **Total** | | **27/40** | **Acceptable** |

## Design Specificity Verdict
Design review: specific in behaviour and words, borrowed in form, as the brief asked. The first principle, verified neighbours only, is invisible.

Automated scan: 12 findings, none actionable: 11 from DESIGN.md staleness or the brief's Geist, reported per rule. Browser flat-hierarchy advisory is inflated by hidden headings. No text under 4.5:1 in either theme. No real target failures, no overflow at any tested width, one partial focus-obscured case at 720x450.

Cognitive load: 3 failures, moderate (one thing at a time, minimal choices, working memory).

## Priority Issues
1. [P1] Ages read as clock times. "1 giờ" means both 1 hour and 1 o'clock, beside a schedule. Fix: one helper rendering "1 giờ trước" in a time element. Command: /impeccable clarify
2. [P1] The one-shot appeal misadvises and under-protects. The placeholder suggests "không nhắm vào ai", which Rule 4 already rejects; Rule 4's text and the flagged phrase aren't shown; "a" uses the only appeal; the reason isn't shown afterward. Fix: show rule text and mark the phrase, better hint, minimum length, show the reason. Command: /impeccable clarify
3. [P1] Composer: chips read as filters with no visible "Đăng vào:"; on phones it opens low and "Đăng bài" sits mid-left; near-black selected chip and stepper beside amber; privacy reminder missing from composer and reply. Fix accordingly. Command: /impeccable adapt
4. [P2] Transparency record hard to reach: at y≈984 below 12 collapsed rows, no anchors, rail hidden at 1230px and below; rules collapsed; emergency numbers not tappable. Fix: anchored route, rules open by default, tel: links. Command: /impeccable layout
5. [P2] Gaps for keyboard, screen readers and large text: Undo about 46-52 tab stops away with a 9-second timer; repeated controls without context; bottom nav labels collide at 200% text; comment branches 1.48:1. Command: /impeccable harden

## Persona Red Flags
- Jordan: chips look like filters; up-arrow has no visible label; nothing says members are verified or posts are checked.
- Sam: Undo out of reach; unnamed repeated controls; nav collisions at 200%; faint branches.
- Casey: close and back out of thumb reach; comment box only above comments; 167px of fixed bars.
- Older resident 65+: "1 giờ" ambiguity; 28-30px desktop actions; urgent thread repeats its action; reply "Hủy" discards without asking.

## Minor Observations
- Tablet has no "Đăng bài" outside composers.
- Urgent sample copy asks people to message publicly about elderly relatives, above the privacy hint.
- "Viết lại và đăng" overwrites an existing composer draft.
- A reply draft carries between recipients under the same comment.
- Stale page title after sending an appeal.
- Errors other than the appeal's are lost on re-render.

## Questions to Consider
- Should a building notice lead rather than the poster's name?
- Should an accepted appeal appear in the feed as a post from Ban quản lý?
- Should warned and held residents see more than the same neutral box?
