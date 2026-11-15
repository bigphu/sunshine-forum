---
target: full prototype after admin, account, palette and states
total_score: 29
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
target_identity: "file:C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
target_fingerprint: "sha256:8271b731efffe8ea71b814539a0a5f9e4b44f57ff8e8b8e4faabb4cba9fd10ff"
target_path: "C:\\d-drive\\assignments\\intelligent-system\\sunshine_apartment_forum\\prototype\\forum.html"
timestamp: 2026-11-08T14-33-16Z
slug: prototype-forum-html
---
Method: dual-agent (A: design review sub-agent · B: detector and browser sub-agent)

## Design Health Score: 29/40 (Good)

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 3 | Skeletons, spinners, offline strip, held receipt and appeal timeline are strong. On phone the notice strip truncates the title. |
| 2 | Match System / Real World | 3 | Natural Vietnamese. But "Thông báo" has four meanings, notice compose shows US dates, and admins see raw hate/offensive labels. |
| 3 | User Control and Freedom | 3 | Undo, drafts, per-notice hide and delete confirm are in place. Deleting a post also removes neighbours' replies, including official answers. |
| 4 | Consistency and Standards | 2 | Rule names come in three forms; "Hiển thị" is used three ways; "Hủy" is styled two ways; the dark "Khẩn cấp" band is pastel; "Đăng xuất" has a back-arrow icon. |
| 5 | Error Prevention | 3 | "Vẫn gửi" is the gold primary in the warning state, and native date inputs risk a day/month swap on an irreversible email. |
| 6 | Recognition Rather Than Recall | 3 | The rules panel disappears below 1200px. The 113/114/115 buttons on home have no labels. |
| 7 | Flexibility and Efficiency | 3 | Suggested rules aren't pre-selected, and single-key shortcuts can't be turned off. |
| 8 | Aesthetic and Minimalist Design | 3 | The banner takes about 22% of the first viewport just to show the page title "Bảng tin". The right rail repeats on every page. |
| 9 | Error Recovery | 3 | The appeal length hint is shown twice. |
| 10 | Help and Documentation | 3 | "Aa" is never explained. |

## Design Specificity Verdict
- **Authored where it matters:** the printed notice, the "Khi bạn đăng" consequences list and the moderation loop are specific to this building.
- **Borrowed elsewhere:** the frame is a Reddit-light chassis, and the banner is oversized.
- **Main gap:** fit for the audience. The compact, desktop-first density doesn't suit older residents on phones.
- **Detector:**
  - **File scan:** 1 false-positive font warning and 22 token-drift advisories (14 font sizes, 5 radii, 3 colours).
  - **URL scans:** 24, all clean apart from the false-positive repeated-meta warning on the queue.
  - **Browser:** overlay clean on 3 routes. Found a real bug the detector missed: Admin › Người dùng scrolls sideways by 24px at 375px.

## Priority Issues
1. [P1] Default text is too small for residents, bottom-nav labels ignore the text-size setting, and "Aa" has no label. Command: /impeccable typeset, /impeccable adapt.
2. [P1] On phone the notice card hides Ảnh hưởng and the strip truncates the title. Command: /impeccable adapt.
3. [P1] Posts that impersonate management carry no resident-side signal; the official marker isn't unmistakable in feed and search. Command: /impeccable harden.
4. [P2] The pre-send warning makes "Vẫn gửi" the primary and lists three outcomes. Command: /impeccable clarify.
5. [P2] "Thông báo" names four different things. Command: /impeccable clarify.

## Persona Red Flags
- **Alex (manager):** suggested rules aren't pre-selected; the shortcut button has an icon but no label; the badges count different things.
- **Sam (screen reader / keyboard):** single-key shortcuts fail WCAG 2.1.4; the outcome announcement after a decision and the inert background behind dialogs need checking.
- **Casey (one-handed phone user):** warning actions sit below the fold; the banner costs about 100px; "Báo cáo" takes its own row; the strip title is truncated.
- **Bà Lan (72, low vision, phone, large text):**
  - text is small and nav labels are fixed at 13px;
  - "Aa" means nothing to her;
  - "Thông báo" is ambiguous;
  - the phone notice card doesn't say who is affected;
  - the fake BQL fee post looks real;
  - reporting offers 7 rules and no "not sure" option;
  - the emergency numbers have no labels.

## Minor Observations
- US-format dates in notice compose.
- The "Ngưỡng chung" (global threshold) setting is dead weight, since every channel sets its own.
- A rejected appeal shows the same gold checks as an accepted one.
- Dark "Khẩn cấp" looks pastel.
- Admin users scrolls sideways at 375px.
- English model labels lead the analytics.
- "Đăng xuất" uses a back-arrow icon.
- The preview card still shows a dismiss button.
- "Comment test" data is left in the hub.
- The hub shows an action styled as a status.
- The appeal hint is shown twice.
- Every admin user row repeats the same button.
- mod-undo.png captured the server's directory listing instead of the app.
- DESIGN.md's type ramp has drifted from the CSS.

## Questions to Consider
- Should text size be set per role?
- Should Rule 7 impersonation get a stricter path?
- Could the banner carry today's notice?
- Can an author delete a post that carries management's answer?
