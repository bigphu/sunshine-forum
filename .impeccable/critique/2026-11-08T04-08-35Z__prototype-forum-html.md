---
target: prototype/forum.html
total_score: 29
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 2
target_identity: "file:C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
target_fingerprint: "sha256:cfa3761155f7a963f96eff32e15f819f09c8c787f4442ecda06e9c2acd0dce7b"
target_path: "C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
timestamp: 2026-11-08T04-08-35Z
slug: prototype-forum-html
---
Method: dual-agent (A: design review sub-agent · B: detector and browser sub-agent)

# Re-score: prototype/forum.html (Reddit light-mode style, artwork banner)

29/40, up from 23, 25 and 27. No P0, two P1. Contrast, overflow and target sizes measured by B; A's contrast figures were calculated from tokens. No real screen reader or Tab walk.

## Design Health Score
| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 3 | Receipt, live region, filter bar good; appeal tracker state is visual only; counts include own held comment. |
| 2 | Match System / Real World | 3 | Natural Vietnamese; "Hữu ích 12" on a water notice reads oddly; rules icon reads as "toggle panel". |
| 3 | User Control and Freedom | 3 | Undo, Thu gọn, Esc, Xóa lọc, drafts work; no withdraw for held comments; composer stays open across pages. |
| 4 | Consistency and Standards | 3 | Warning and pending look identical; emergency buttons styled two ways; "Yêu cầu của tôi" vs "Yêu cầu xem xét lại". |
| 5 | Error Prevention | 3 | Strong warning and appeal guards; send button unchanged during a warning; Báo cáo 8px from Hữu ích. |
| 6 | Recognition Rather Than Recall | 3 | Mostly labelled; icon-only rules button; inert "T" avatar looks like a button. |
| 7 | Flexibility and Efficiency | 2 | Search, skip link, sort, filter; no text-size control; search excludes comments. |
| 8 | Aesthetic and Minimalist Design | 3 | Calm; 220px banner and per-post report cost space. |
| 9 | Error Recovery | 3 | Errors focus and clear; appeal error doesn't say how many characters remain. |
| 10 | Help and Documentation | 3 | Good contextual help; no first-visit orientation; "Hữu ích" unexplained. |
| **Total** | | **29/40** | **Good** |

## Design Specificity Verdict
Specific in content, generic in the visual shell (the requested Reddit look). Banner is the weakest specific element: rotated mark per channel, hard to tell channels apart, 220px on every channel visit.

Automated scan: 19 findings, all against stale DESIGN.md; one real problem (bottom-nav labels 12px at 360px and below). Browser scan: wrapping advisories only (stragglers, one unbalanced title, two long rules lines).

Banner: title at least 16.3:1, description at least 12.7:1 (analytic composite); no art overlaps the text at any tested width or channel.

Measured: one light text failure ("Đóng" in the receipt, 4.36:1); no dark text failures; no overflow at 11 widths; no target under 44px on phones; all forms handle errors and survive theme changes.

## Priority Issues
1. [P1] Below 1200px the post button jumps above the content (tablet: between banner and urgent notice; threads: above the notice); focus order mismatches visual order. Fix: keep the right column after the feed; compact Đăng bài beside sort and filter. Command: /impeccable layout
2. [P1] The pre-send warning looks like neutral information (same grey box as pending; send button unchanged). Fix: stronger structure for warnings; relabel to "Vẫn gửi" while a warning shows. Command: /impeccable clarify
3. [P2] Phone flow gaps: report submit below the fold; composer "Thu gọn" drops focus to body on phones; comment box half hidden behind the bottom nav at 720x450. Fix: sticky sheet actions, visible focus target, scroll room. Command: /impeccable adapt
4. [P2] Báo cáo has the same weight as positive actions on every post, the urgent notice included. Fix: move it to the thread or make it quiet text on the right. Command: /impeccable quieter
5. [P2] Flat controls rely on 1.15:1 fills (secondary buttons, sort switch, resting search, action pills); receipt "Đóng" 4.36:1. Fix: subtle visible edge on resting controls; darker "Đóng". Command: /impeccable polish

## Persona Red Flags
- Jordan: rules icon reads as "toggle panel"; inert avatar; "Hữu ích" unexplained.
- Sam: focus order mismatch below 1200px; appeal tracker step not announced; grey warning easy to miss at magnification.
- Casey: report submit off-screen; Báo cáo next to Hữu ích; search and theme in the hardest corner.
- Older resident: orange bar above the notice on tablets; comment box before existing answers; 12-13px nav labels.

## Minor Observations
- Phone pinned row cuts "15/12".
- Appeal details squeeze into two columns on phones; no character counter.
- Own posts can be marked "Hữu ích"; composer stays open across pages.
- Search results skip h1 to h3; both rules pages share a title.
- Sample appeal deadline doesn't match the removal age.

## Questions to Consider
- Should the banner shrink or show something real?
- Should reporting be as easy as appreciating?
- Should comments come before the reply box?
