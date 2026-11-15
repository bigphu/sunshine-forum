---
target: prototype/forum.html
total_score: 23
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 5
target_identity: "file:C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
target_fingerprint: "sha256:4106325b4edb79b87844d4bb063b475e7e7ac911010c574fbe9c3123eb3e0255"
target_path: "C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
timestamp: 2026-11-07T11-42-11Z
slug: prototype-forum-html
---
Method: dual-agent (A: design review sub-agent · B: detector and browser sub-agent)

# Critique: prototype/forum.html

The main defects in Priority Issues 1, 2 and 5 were checked against the source. Both reviewers worked mostly from DOM measurements and earlier captures because the browser pane was hidden. Neither ran a real screen reader or phone.

## Design Health Score

| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of System Status | 2 | The pending message lands off-screen on phones, and a held post's own thread doesn't show it. |
| 2 | Match System / Real World | 3 | The Vietnamese is natural, but "Hữu ích" never appears on screen and "Sôi nổi 1" is unexplained. |
| 3 | User Control and Freedom | 2 | "Hủy" and "Để sau" discard drafts without warning; no edit or delete on own posts. |
| 4 | Consistency and Standards | 3 | Report is in a menu on posts but inline on comments; posts get a toast and comments don't. |
| 5 | Error Prevention | 3 | The warning is strong, but it vanishes on re-render and Rule 6 holds content with no warning. |
| 6 | Recognition Rather Than Recall | 2 | The warning cites "Quy tắc 1" with no title or link; channel buttons are icons only on phones. |
| 7 | Flexibility and Efficiency | 1 | No search; the filter carries silently between channels. |
| 8 | Aesthetic and Minimalist Design | 3 | Calm overall, but the right column repeats the next three feed items. |
| 9 | Error Recovery | 2 | Clear messages, but old errors linger and a deleted post has no route to rewrite it. |
| 10 | Help and Documentation | 2 | Good rules page, but no link from the warning, pending or report moments. |
| **Total** | | **23/40** | **Acceptable: significant improvements needed** |

## Design Specificity Verdict

Design review: half grounded. The behaviour belongs to this product; the composition doesn't, which follows the Peerlist brief. Specific to Sunshine: pre-send warning, pending copy, hidden-comment placeholder, official marker built from the logo, "Trước / Nay" rule record. Missing: the building. A neighbour is only an initial and "Cư dân X". The rule-change card disappears below about 1210px.

Automated scan: 12 findings (3 warnings, 9 advisories), all reported from forum.html. 11 compare against the stale DESIGN.md; regenerating it should clear most, though the tiny text sizes are a real problem (issue 3). 1 flags Geist as overused; keep it, since it follows the Peerlist brief and its Vietnamese characters render correctly. Browser scan adds a skipped heading level on home and a flat type hierarchy (1.07:1 against a 1.25:1 target). Text contrast: about 412 combinations with no failures in either theme; one hover state calculated at 4.26:1.

Visual overlays: injection worked, but the helper stopped and the tab closed, so no overlay is visible.

Cognitive load: 3 failures, moderate (chunking, minimal choices, working memory). Decision points over 4 options: report dialog 7 (mandated), left nav 6, bottom nav 5, phone channel dialog 5, rules page 7 + 5 accordions.

## Overall Impression
A calm, believable forum with excellent moderation writing. The biggest weakness is that the reassurance often doesn't reach the resident when it matters. The biggest opportunity is making held content, removed content and urgent notices as carefully handled as the comment tree.

## What's Working
- Comment tree: precise and readable down to phone size; the inline reply form sits on its branch.
- Moderation copy: "Đây chưa phải là một lần vi phạm" and "Một người thật sẽ đọc lại" reassure without scolding; the warning sits right before send.
- Logo as official identity: the logo avatar, "Phản hồi chính thức" and "Ban quản lý đã trả lời" make the brand carry meaning.

## Priority Issues

1. [P1] Held-content feedback doesn't reach the resident. After a comment is held, the box empties and no toast appears; the pending message rendered at y≈1290 on a phone with the page at the top; a held post's thread shows no pending state. Design review rated this P0; rated P1 because the comment does submit. Fix: lasting receipt under the comment box, scroll to the new comment, pending block in held threads, a time expectation. Command: /impeccable harden

2. [P1] Keyboard use is broken in specific places. Skip link replaces the page with "Không tìm thấy" (router treats #mc as a route). Forward Tab starts inside main, so the skip link, logo and left nav come last or need Shift+Tab. Focus lost after dismissing the urgent notice, composer cancel and submit, and post-menu report. Menus use role="menu" with no arrow keys. Composer fields have about 1.4:1 focus cue; field borders 1.23:1. Fix: skip link focuses #mc without routing; move focus only on real route changes; return focus to openers; plain disclosure buttons; darker borders and a real focus style. Command: /impeccable audit

3. [P1] Urgent notices and small text undersell things for older residents. "Khẩn" is 11px; a water shut-off title is the same 15px as a chess invitation; the pinned card omits the action and gives no date; the notice is absent on channel, notification and rules pages; dismiss has no undo; meta text reaches 10-13px; px sizing ignores browser font settings; tablet action buttons are 28-30px. Fix: dedicated banner on every page, 18px+ title, date, action line, undo; rem type with about a 14px floor; size targets by pointer type. Command: /impeccable typeset

4. [P1] Moderation recovery has holes at the most stressful moments. Theme toggle wiped a typed appeal reason and "Để sau" discards it; warning disappears on re-render, has no rule link or consequence line; Rule 6 holds silently; errors persist after fixing; report dialog doesn't say content stays visible and a 3.5s toast is the only confirmation; removed posts have no rewrite route. Fix: keep drafts and warning state across re-renders; rule title, link and consequence in the warning; clear errors on typing; "still visible" line and lasting "Đã báo cáo" state; rewrite route on the appeal page. Command: /impeccable clarify

5. [P1] Long text and filter state break the feed. A 300-character unbroken string overflows by 2,256px on desktop and 3,145px on phones; a filter set on home carries into Marketplace, which falsely says "Chưa có bài viết nào"; on phones the last post's "Báo cáo" sits under the bottom nav. Fix: overflow-wrap:anywhere on user text; reset the filter per channel or show "Đang lọc · Xóa lọc"; open the menu upward near the bottom. Command: /impeccable harden

## Persona Red Flags

Jordan (first-timer): "Thông báo" and "Thông báo chung" are nearly identical; "Quy tắc" and "Minh bạch" go to the same page; the up-arrow has no visible meaning; nothing explains the AI filter until a warning fires.

Sam (screen reader and keyboard): skip link destroys the page; focus stranded after reporting; 57 tab stops on home; headings skip h1 to h3; thread h1 smaller than h2; errors not linked to fields.

Casey (one-handed phone): pending message off-screen; fixed bars take 167px, 37-45% of a short screen; two bells with badges; unlabelled channel icons.

Older resident, 65+ (project-specific): "Khẩn" easy to miss; no action line or date in the feed; an accidental X removes the urgency for good; "Trả lời" is small muted text; a nested official reply has no reply button and no explanation.

## Minor Observations
- Dark theme: orange tab outline and amber button on near-black are closest to the "hub" look. Light theme is clear of it.
- Uppercase tabs crowd Vietnamese diacritics and break sentence case.
- Rules page: changes and figures come before the rules; emergency numbers come last.
- "Bình luận đã ẩn" in bold in the name slot reads like a person.
- Thread back goes to the channel, not where you came from; returning loses scroll.
- Home and rules scroll sideways by 4-5px at around 1210px wide with a scrollbar.
- Report radios measure about 15×18px on phones; three rows are 39px tall.

## Questions to Consider
- Cover the logo: does anything say these people share a building? Could "Tòa B" beside a name build more trust than any colour, without breaking Rule 5?
- The resident with the deleted lift post still has a broken lift. Why is a formal appeal the only door?
- Should a water shut-off share the rhythm of a chess invitation? Where does the Peerlist clone stop serving a 70-year-old?
