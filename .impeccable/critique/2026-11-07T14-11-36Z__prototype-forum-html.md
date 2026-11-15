---
target: prototype/forum.html
total_score: 25
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 4
target_identity: "file:C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
target_fingerprint: "sha256:70cb2a1d4422f1f0b7e33484a6b3de09a9556c518ff687ecb651751fe48c26fa"
target_path: "C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
timestamp: 2026-11-07T14-11-36Z
slug: prototype-forum-html
---
Method: dual-agent (A: design review sub-agent · B: detector and browser sub-agent)

# Re-score: prototype/forum.html

Score moved from 23/40 to 25/40, still Acceptable. Neither reviewer used a real screen reader or phone.

## Status of the first critique's issues
- Held-content feedback: fixed.
- Keyboard use: mostly fixed; focus and error gaps remain (issue 4).
- Urgent notices and small text: partly fixed; urgency disappears inside the notice (issue 1).
- Moderation recovery: partly fixed; some reassurance missing (issue 3).
- Long text and filter bugs: fixed.

## Design Health Score
| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 3 | Held and appeal states strong; on phones, sort and filter show no visible change. |
| 2 | Match System / Real World | 3 | Natural language; up-arrow meaning and "Marketplace" are the gaps. |
| 3 | User Control and Freedom | 2 | Undo and composer drafts work; reply drafts wiped; held content can't be withdrawn. |
| 4 | Consistency and Standards | 2 | Report in a menu on posts, inline on comments; held comments and replies confirm differently. |
| 5 | Error Prevention | 3 | Warning excellent but never says rewriting is free. |
| 6 | Recognition Rather Than Recall | 2 | Icon-only actions; channel buttons read as filters; composer fields have no visible edge. |
| 7 | Flexibility and Efficiency | 2 | Sort and filter exist; no way to find your own posts or appeal. |
| 8 | Aesthetic and Minimalist Design | 3 | Calm overall; "Đăng bài" twice; amber overused. |
| 9 | Error Recovery | 3 | Errors move focus and clear on typing; rewrite route prefilled; mistaken report can't be undone. |
| 10 | Help and Documentation | 2 | Good rules page; nothing explains AI checks before a warning fires. |
| **Total** | | **25/40** | **Acceptable** |

## Design Specificity Verdict
Design review: specific in behaviour, generic in composition, which follows the brief. The logo as management's signature is the strongest brand decision. Weakest in the slot under a resident's name, where nothing says these people share a building.

Automated scan: 12 findings. Ten are DESIGN.md staleness, one is Geist (brief requires it). Browser flat-hierarchy still advisory. Text contrast: no failures in either theme apart from one large-text hover state at 4.41:1, which passes for its size.

Cognitive load: 3 failures, moderate (chunking, one thing at a time, minimal choices).

## Priority Issues
1. [P1] Urgency disappears inside the notice. Opening the urgent notice shows no urgent label and the action is in paragraph 2; dismissing the banner strips the label everywhere. Fix: separate "urgent" from "banner closed"; show label and action line in the thread. Command: /impeccable harden
2. [P1] The phone's first screen shows no neighbours. First resident post at y=800 behind the bottom nav; pins ignore sort and filter. Fix: collapse non-urgent pins, put urgent above composer, hide pins while filtering. Command: /impeccable adapt
3. [P1] Reassurance missing at high-stakes moments. Warning lacks the approved rewrite line; reporting management doesn't mention the administrator; notification offers an appeal after one is sent; appeal guarantee is muted. Fix: approved sentences; update notification. Command: /impeccable clarify
4. [P1] Keyboard and error-state gaps. Focus can sit under sticky bars; focus drops to body after Undo and Copy link; comment, reply and report errors don't set invalid state; composer error doesn't show on the field; theme toggle removes the appeal error. Fix: scroll padding, real focus targets, consistent error handling. Command: /impeccable audit
5. [P2] Amber means five different things. Urgent, warnings, pending, active filter and brand button; warning and urgent backgrounds are 1.04:1. Fix: keep the tint for urgent; neutral surface with strong border for moderation states. Command: /impeccable colorize

## Persona Red Flags
- Jordan: taps "Bảo trì" expecting a filter and gets a composer; doesn't know the up-arrow means useful.
- Sam: strong structure; "Thêm tùy chọn" and "Hữu ích" buttons sound identical; focus can sit under the bars.
- Casey: targets fine; filter feedback off-screen; 20.5% permanent sticky bars.
- Older resident 65+: feed notice works well; urgency vanishes after tapping; 7 rules must be opened one by one.

## Minor Observations
- Switching replies wipes a typed reply.
- A held comment shows the same amber paragraph twice.
- Tablet thread has no "Đăng bài".
- Dark urgent logo tile is near-black with the orange mark.
- "như khi gặp hàng xóm ở sảnh chung cư" isn't on the rules page.

## Questions to Consider
- Could "Tòa B" under a name make neighbours feel like neighbours without breaching Rule 5?
- When a rule changes because residents appealed, why is that a sidebar card and not news in the feed?
- Should the appeal page acknowledge the broken lift before asking the resident to argue their case?
