---
name: Sunshine Apartment Forum
description: A moderated forum for one building's verified residents, where notices, rules and moderation decisions sit in plain sight.
colors:
  sun-300: "#FFDB80"
  sun-500: "#FFC53D"
  sun-600: "#F0AE14"
  sun-700: "#8A5A00"
  ink-950: "#0E1116"
  night-950: "#0B1324"
  ink-600: "#5A6472"
  ink-500: "#7A8699"
  edge: "#AEB5BF"
  line: "#E2E5EA"
  band: "#EDEFF2"
  soft: "#F6F7F9"
  white: "#FFFFFF"
  danger: "#B3261E"
  danger-soft: "#FBE9E7"
  approve: "#0F766E"
  approve-soft: "#E3F4EF"
  highlight: "#E8F0FE"
  highlight-line: "#8DB0EE"
  warn-text: "#1E3A8A"
  avatar-blue: "#2563EB"
  avatar-teal: "#0F766E"
  avatar-rust: "#C2410C"
  avatar-violet: "#7C3AED"
  banner-glow: "#2E4170"
  banner-mid: "#15223F"
  banner-subtitle: "#D3DBEA"
typography:
  display:
    fontFamily: "Be Vietnam Pro, system-ui, sans-serif"
    fontSize: "2.75rem"
    fontWeight: 800
    lineHeight: "3.25rem"
    letterSpacing: "-0.025em"
  headline:
    fontFamily: "Be Vietnam Pro, system-ui, sans-serif"
    fontSize: "2.125rem"
    fontWeight: 700
    lineHeight: "2.625rem"
    letterSpacing: "-0.02em"
  title:
    fontFamily: "Be Vietnam Pro, system-ui, sans-serif"
    fontSize: "1.3125rem"
    fontWeight: 700
    lineHeight: "1.8125rem"
    letterSpacing: "-0.01em"
  body:
    fontFamily: "Be Vietnam Pro, system-ui, sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 400
    lineHeight: "1.625rem"
    fontFeature: "tnum"
  label:
    fontFamily: "Be Vietnam Pro, system-ui, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 600
    lineHeight: "1.375rem"
rounded:
  xs: "6px"
  sm: "8px"
  md: "12px"
  field: "14px"
  lg: "16px"
  xl: "18px"
  card-phone: "20px"
  dialog: "26px"
  full: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  2xl: "28px"
  3xl: "36px"
components:
  button-primary:
    backgroundColor: "{colors.sun-500}"
    textColor: "{colors.ink-950}"
    rounded: "{rounded.full}"
    padding: "0 18px"
    height: "44px"
  button-primary-hover:
    backgroundColor: "{colors.sun-600}"
  button-secondary:
    backgroundColor: "{colors.band}"
    textColor: "{colors.ink-950}"
    rounded: "{rounded.full}"
    padding: "0 18px"
    height: "44px"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.white}"
    rounded: "{rounded.full}"
    padding: "0 18px"
    height: "44px"
  button-ghost:
    textColor: "{colors.sun-700}"
    rounded: "{rounded.full}"
    padding: "0 18px"
    height: "44px"
  decision-approve:
    backgroundColor: "{colors.white}"
    textColor: "{colors.ink-950}"
    rounded: "{rounded.full}"
    height: "48px"
  decision-approve-hover:
    backgroundColor: "{colors.approve-soft}"
  decision-reject:
    backgroundColor: "{colors.white}"
    textColor: "{colors.ink-950}"
    rounded: "{rounded.full}"
    height: "48px"
  decision-reject-hover:
    backgroundColor: "{colors.danger-soft}"
  action-chip:
    backgroundColor: "{colors.band}"
    textColor: "{colors.ink-950}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "0 16px"
    height: "44px"
  input-field:
    backgroundColor: "{colors.white}"
    textColor: "{colors.ink-950}"
    rounded: "14px"
    padding: "12px 14px"
    height: "46px"
  search-field:
    backgroundColor: "{colors.band}"
    textColor: "{colors.ink-950}"
    rounded: "{rounded.full}"
    height: "44px"
  card:
    backgroundColor: "{colors.white}"
    rounded: "{rounded.lg}"
    padding: "20px 22px"
  queue-row-current:
    backgroundColor: "{colors.white}"
    rounded: "14px"
    padding: "12px"
  channel-tag:
    backgroundColor: "{colors.band}"
    textColor: "{colors.ink-600}"
    rounded: "{rounded.full}"
    padding: "0 10px"
    height: "28px"
  avatar:
    backgroundColor: "{colors.avatar-blue}"
    textColor: "{colors.white}"
    rounded: "{rounded.full}"
    size: "40px"
---

# Design System: Sunshine Apartment Forum

## Overview

**Creative North Star: "The Open Notice Board"**

This is one building's notice board and the conversation around it, kept in plain sight. The page is white, the writing is ink, and titles are strong enough to scan from across the room. Sun gold marks the places where a resident can act. The half-disc mark stands for management's own voice. Moderation lives in the same open layout as everything else: which rule applied, what happens next, and who decides.

The visual world is the Sunshine brand kit, recoloured on 2026-11-08 from the kit's orange to a golden sun on cool ink and navy neutrals, set in Be Vietnam Pro, in the style of Reddit's light mode: a white page, posts as an open list split by hairlines, grey pill buttons, and a light grey side panel. Only forms and page sections sit in bordered cards. Each channel has a Steam-library-style artwork banner: a deep navy field with the gold half-disc mark arranged differently per channel. The resident journey and the manager moderation dashboard share one shell, one set of tokens and one component kit. A prototype role switch moves between them.

Density is a reading forum, with one text size for everyone: root 100%, body 17px. Bottom-nav labels are at least 13px and can wrap to two lines. Every target is at least 44px. The confirmed rejections are card-grid dashboards, decorative gradients and stripes, a punitive moderation tone, the earlier pine green look, and orange on near-black (the owner rejected it because it recalls an adult site's branding).

**Key Characteristics:**
- Sun gold on ink and navy neutrals; one warm signal on a cool page.
- One family, Be Vietnam Pro, with strong titles over quiet grey metadata.
- Hairline-divided lists on a flat page; cards only where something is filled in or decided.
- Management posts carry the half-disc mark and "Chính thức", and can be pinned.
- Vivid avatars and colour-coded decision icons make people and outcomes easy to tell apart.
- Light only and one text size for the first release.

## Colors

One warm brand signal against a cool ink scale, with a small set of project additions for states the kit does not cover: danger, approve, highlight and avatars.

### Primary
- **Sun Gold** (`sun-500`): primary buttons, the unread dot, the channel banner artwork, and the mark. Text on it is always ink-950.
- **Pressed Sun** (`sun-600`): primary button hover.
- **Sun Text** (`sun-700`): every link and sun-coloured text (channel names, "Xem Quy tắc 5", back links), and the current-row ring in the moderation queue.
- **Pale Sun** (`sun-300`): text selection.

### Neutral
- **Ink 950** (`ink-950`): text, the focus ring, the toast background, and all text on sun gold.
- **Night 950** (`night-950`): the deepest point of the channel banner.
- **Slate** (`ink-600`): muted metadata, hints, timestamps, dt labels.
- **Field Grey** (`ink-500`): input borders, comment-tree branches, and the faint scrollbar thumb (45% alpha).
- **Edge** (`edge`): pill button outlines.
- **Line** (`line`): hairlines between rows and sections.
- **Band** (`band`): grey pill buttons, search field, segmented controls, hover fill.
- **Soft** (`soft`): side panel, queue panel, pinned rows, previous-message blocks.
- **White** (`white`): page and cards.
- **Banner Night** (`banner-glow` to `banner-mid` to `night-950`, with `banner-subtitle` text): the radial navy field behind the gold channel banner art; always dark.

### Project Additions
- **Alarm Red** (`danger`): errors, the reject icon disc, "Gửi và từ chối", "Gửi và gỡ", and the Mức cao icon. `danger-soft` tints the reject button on hover or when open.
- **Approve Teal** (`approve`): the approve, allow or accept icon disc only. `approve-soft` tints that button on hover or when open.
- **Highlight Blue** (`highlight`, `highlight-line`): a deep-linked comment, a highlighted rule, and search hits. Its text partner is `warn-text`. It is a "you are here" marker, never a signal of importance.
- **Avatar Set**: four saturated hues (blue, teal, rust, violet) with white initials. The hue is picked by hashing the name.

### Named Rules
**The One Warm Signal Rule.** Sun gold means "act here". It never decorates empty space, and text on it is always ink-950.

**The Outcome Colour Rule.** Teal and red appear only as the small icon discs and soft tints of decision buttons, plus red for errors. Both outcome buttons stay the same size and weight.

**The Highlight Is Not An Alert Rule.** Pale blue marks where the reader landed (a linked comment, a rule, a search hit). It never signals importance.

## Typography

**Display and Body Font:** Be Vietnam Pro (with system-ui, sans-serif)

**Character:** A single humanist sans with full Vietnamese diacritic support. Weight carries hierarchy: 800 to 700 for titles and 400 for reading, with 600 reserved for labels and controls. Numerals are tabular everywhere.

The root is 100%, so 1rem is 16px and body text 17px. Bottom-navigation labels are at least 13px.

### Hierarchy
- **Display** (800, 2.75rem / 3.25rem, -0.025em): channel banner titles only, white on the ink artwork. Short channel banners use 2.25rem.
- **Headline** (700, 2.125rem / 2.625rem, -0.02em): page titles ("Hàng chờ kiểm duyệt", "Quy tắc cộng đồng"). The thread title is 1.875rem.
- **Title** (700, 1.3125rem / 1.8125rem): post titles in the feed, card headings (1.25rem), and case titles (1.375rem).
- **Body** (400, 1.0625rem / 1.625rem): posts, comments and forms. Flagged content in a moderation case is shown larger (1.1875rem / 1.875rem). Page prose is capped at 58ch.
- **Label** (600, 0.9375rem / 1.375rem): buttons, chips, metadata, group headings and dt labels. Smaller labels go down to 0.8125rem (queue markers, count pills, channel tags).

### Named Rules
**The Weight Before Size Rule.** Step hierarchy with weight first (400 to 600 to 700). Add size only for titles, and never shout with uppercase or letter-spaced kickers.

**The Balanced Heading Rule.** Headings use `text-wrap: balance` and prose uses `pretty`, so no line ends on a single word.

## Layout

A three-part shell: a sticky frosted top bar (60px, 56px on phone), a 264px collapsible sidebar, and a main area of up to 1160px (1440px for the moderation dashboard) with 36px side padding.

- **Resident pages:** a feed column next to a 320px side panel (post button, channel note, the seven rules, emergency numbers). Below 1200px the side panel goes, and "Đăng bài" moves into the top bar.
- **Text pages:** rules, requests, appeals, notifications and search use an 800px column of stacked cards with a 16px gap.
- **Moderation dashboard:** uses a container query, not a viewport breakpoint. At 840px or wider it is a sticky queue column (320–420px, toolbar on top, its own scroll) beside a reader of up to 860px. Narrower, the queue and the case are separate screens with a "Hàng chờ" back link.
- **Phone (820px and below):** one column, 14px gutters, a compact banner, and a five-tab frosted bottom bar. Dialogs become bottom sheets.

The spacing rhythm is base 4: 4, 8, 12, 16, 20, 28 and 36px. Groups are tight (8–12px) and sections more generous (16–20px).

### Named Rules
**The 44px Floor Rule.** Every tap target is at least 44px tall. Small radios and checkboxes sit inside rows that are at least 44px, and inline text links in metadata use padding on touch screens.

## Elevation & Depth

Mostly flat. Depth comes from tonal layering (white on soft) and hairlines. Real shadows are kept for things that float.

### Shadow Vocabulary
- **Hairline ring** (`box-shadow: 0 0 0 1px var(--line)`): cards and panels at rest.
- **Lifted** (`0 2px 6px rgba(14,17,22,.06), 0 14px 36px rgba(14,17,22,.10)`): menus, dialogs, the sidebar toggle.
- **Current row** (`0 1px 2px rgba(14,17,22,.06), 0 6px 16px rgba(14,17,22,.08), inset 0 0 0 1.5px sun-700`): the case open in the moderation queue.

### Named Rules
**The Earned Shadow Rule.** A surface casts a shadow only when it floats (a menu, dialog or toast). Lists and cards at rest use a hairline.

## Shapes

Soft, friendly geometry: pills for anything you press, 16px rounded cards, 26px dialogs and bottom sheets, and circles for avatars and icon buttons. Comment trees use 2px branches with a 14px rounded elbow to each reply. The brand mark's half-discs repeat as oversized banner art.

## Components

### Buttons
- **Shape:** full pills (999px), 44px tall, 18px side padding, 600 weight. They press down to 98% scale.
- **Primary:** sun gold with ink text, one per area ("Đăng bài", "Gửi bình luận"). Hover is `sun-600`.
- **Secondary:** band fill with a 1.5px `edge` outline. Hover blends toward field grey.
- **Ghost:** no fill, link colour ("Hủy", "Đặt lại").
- **Danger:** alarm red, only on a final irreversible send ("Gửi và từ chối", "Gửi và gỡ").
- **Toggled / on:** an ink fill with page-colour text (an active filter, "Chọn nhiều", an open rejection).
- **Decision pair:** two equal 48px pills on the card colour, each led by a 30px filled icon disc (a teal check, a red cross). They tint toward teal or red on hover and when expanded.

### Chips
- **Action chips:** "2 bình luận", "Hữu ích", "Trả lời". Band fill, 1px edge ring, 44px. Pressed state is sun gold. A manager's "Gỡ nội dung" is always quiet muted text at the row's end, never a filled chip.
- **Channel tag:** band fill, muted 0.8125rem text, 28px.
- **Count and marker pills:** small band pills for counts. Queue markers are an icon plus bold text.

### Cards / Containers
- **Corner Style:** 16px (20px on phone), dialogs 26px.
- **Background:** white, with soft or band for blocks inside a card. A bordered card never sits inside another bordered card.
- **Shadow Strategy:** a hairline ring at rest (see Elevation).
- **Internal Padding:** 20px 22px (18px on phone).

### Inputs / Fields
- **Style:** card background, a 1.5px field-grey border, 14px radius, 12px 14px padding, 46px minimum height. Labels always sit above the field.
- **Focus:** a 3px ink outline offset 3px, shared by every focusable element.
- **Error:** a 2.5px red border, a red message with a filled "!" disc, and `aria-invalid`. What the user typed is kept.
- **Pre-send warning:** a card with a 2px ink inset ring, a filled ink alert disc and the rule named. "Sửa lại" sits directly under the message as the gold primary, then one line, "Sửa lại sẽ không bị tính là vi phạm.", and two disclosures ("Nếu vẫn gửi thì sao?", the rule text). The form's send button loses its gold and becomes the secondary "Vẫn gửi".

### Navigation
- **Top bar:** frosted glass with the logo, a centred grey pill search, and 44px circular icon buttons (bell with a sun count dot, avatar with a ring).
- **Sidebar:** 44px rows with 20px icons; the current row has a band fill and 700 weight. It ends with the prototype role switch, and its scrollbar shows only on hover or focus.
- **Phone bar:** five frosted tabs with 24px icons and 13px labels; the current tab has a band fill.
- **Tab pill:** a segmented band pill whose current tab is a raised card ("Cư dân" and "Vai trò"; "Mới nhất" and "Sôi nổi").

### Announcements
A management post is an ordinary post with the management mark and "Chính thức" beside the name. Managers write it on the normal compose page: channel, title, body, then "Đăng bài". A pinned post shows a "Đã ghim" line in its channel, and a soft row at the top of Bảng tin.

### Removing Published Content
On a resident's published post or comment, managers see a quiet "Gỡ nội dung" control. It opens a dialog with the rule list, the exact resident message, "Hủy", and the danger button "Gửi và gỡ". The content leaves the thread, and a toast confirms it.

### Moderation Queue (signature)
A soft-grey panel with a toolbar on top: "Hiển thị" (sort, density, channel, saved per device), "Chọn nhiều", and a round keyboard button. Groups follow the spec order, each with an icon, a name, a count pill and a sticky header. Rows have a 32px vivid avatar, a title of up to three lines, a meta line (type · channel · time), and at most one marker. Rows are split by inset hairlines that disappear around the hovered or current row. Decisions use the decision pair, reveal rule choices and the exact resident message, and confirm in a quiet inline strip that names the outcome and says the resident was notified.

### Account Screens
Sign-in, set-password and forgot-password pages drop the app shell. They show a logo, then one 460px card on the soft page, with a 1.75rem h1, one muted line, 50px fields and a full-width primary button. Password fields carry an inline "Hiện" toggle (aria-pressed). New passwords show a live checklist that turns ok-teal as each rule is met. Errors appear under each field, and focus goes to the first one. The avatar in the top bar opens an account menu (Cài đặt, Yêu cầu của tôi, Đăng xuất); on narrow phones the menu spans the width under the top bar. Settings are cards of definition lists, sub-headings and **switches**: 48×28 pills, field-grey when off and ink when on, inside 56px label rows. Rules the resident cannot change, such as invitation and password email, are written as a check/cross list rather than disabled switches. A **read-only card** (soft fill, edge hairline, clock icon) replaces every composer while an account is read-only, naming the end date and reason and linking to the violations. Own posts and comments get quiet "Sửa" and "Xóa" actions. Edits are re-checked and marked "đã sửa"; a deleted comment leaves a placeholder.

### System States
Every data area has the states in UI_DESIGN_SPEC §5.1. **Loading** shows skeletons shaped like the final layout: band-grey blocks with a slow 1.4s pulse (still under reduced motion), and the main region sets `aria-busy` with a hidden "Đang tải…" status. Nothing new appears for loads under 300ms. A **load error** replaces only the area that failed (the feed list, a thread's comments, the queue, a page body or admin data) with a card: red alert icon, a bold "Không tải được …" line, one muted sentence, and "Thử lại". Banners, posts and panels around it stay usable. An **action in progress** keeps the button's width and swaps its label for a 16px ring spinner with "Đang gửi" or "Đang lưu" (aria-busy). A **send or save failure** adds a persistent danger-tint notice with a 1.5px danger outline inside the form, above its buttons. It says the text is still there and offers "Thử lại"; the input is never cleared. **Offline** adds a slim band-grey strip under the top bar (sticky, with an alert icon) saying drafts are kept on the device, and sends fail with the offline wording until the connection returns.

### Admin Screens
Admin pages sit under a "Quản trị" sidebar group (a "Quản trị" tab and an index page on phone), in a 1080px column with a "Quản trị" back link, an h1 and one muted lead line. Lists are hairline **data tables** inside a card: 44px rows, 13px muted column heads, primary text with a muted second line, and the row action right-aligned. On phones each row stacks, with a fixed-width muted label before each value and no label on the name. Status uses pills: ok-tint "Đang hoạt động", band pills with an icon for invited and read-only accounts, and an outlined muted pill for deactivated ones. Filters are the same chip row as the composer's channel picker. Settings use 96px number fields with a "/100" unit. A **threshold strip** (soft and band, labelled) shows where content is published or held, with the exact ranges written underneath. Auto-delete is a switch, off by default. Turning it on adds a danger-tint third part to the strip, a score field and a per-channel column. Unsaved settings collect in an ink **save bar** that sticks to the bottom (above the phone bar) and names the count; errors are written in the bar and mark the fields. Charts are neutral `field`-grey columns or bars with 4px rounded ends, recessive gridlines, a hover value, a written `aria-label` summary and a "Xem dạng bảng" table. The audit log carries no edit controls. Rows where the AI was overturned get an alert pill, and user IDs are pseudonymous.

Circles of 40px (34px in comments, 32px in the queue) in the four avatar hues, with a single initial. Management uses the brand mark on a white circle with a hairline, which doubles as the official-voice marker next to "Phản hồi chính thức".

### Dialogs And Toasts
Dialogs are 560px wide with a 26px radius and a blurred scrim, and become bottom sheets with a grab handle on phone. Toasts are an ink pill that never covers the next action. The moderation confirmation is inline rather than in a toast.

## Do's and Don'ts

### Do:
- **Do** make rephrasing the easy path: in any warning, "Sửa lại" is the primary and sending anyway is secondary.
- **Do** mark every management post "Chính thức" beside the name, and add "Bài của cư dân, không phải tin chính thức của ban quản lý." to resident posts whose title or opening imitates management.
- **Do** keep text on sun gold ink-950, and use `sun-700` for sun-coloured text.
- **Do** make every target at least 44px, and keep radios and checkboxes inside 44px rows.
- **Do** keep the two moderation outcomes the same size, coloured only by their icon disc and hover tint.
- **Do** keep scrollbars thin and faint, and show panel scrollbars only on hover or keyboard focus.
- **Do** cap page prose at 58ch and balance headings.
- **Do** label sample data ("Dữ liệu mẫu", "Điểm mẫu") wherever numbers or scores appear.

### Don't:
- **Don't** use the earlier pine green, amber-tinted notice cards, or decorative stripes and gradients on surfaces.
- **Don't** nest a bordered card inside a card. Use a soft or band block, or a ruled divider.
- **Don't** fill the "Gỡ nội dung" control or make it louder than the other actions.
- **Don't** use teal or red as large fills outside errors and the final danger button.
- **Don't** add uppercase kickers or eyebrows above headings.
- **Don't** shrink below the 100% root for body text or below 13px for any label.
