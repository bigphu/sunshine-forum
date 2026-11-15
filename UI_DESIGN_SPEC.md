# Sunshine Apartment Forum: UI Design Specification

**Status:** draft for review. Visual choices are proposals until you approve them or replace them with a direction from a design tool.

Requirements come from [PRODUCT.md](PRODUCT.md) and [SPEC_CHANGES.md](SPEC_CHANGES.md). Moderation wording comes from [COMMUNITY_RULES.md](COMMUNITY_RULES.md).

## Project context

| Item | Definition |
|---|---|
| **Core purpose** | A verified, moderated hub for announcements, resident discussion, maintenance requests, and marketplace posts. AI checks content before it goes public, and humans have the final word. |
| **Residents** | About 100 verified tenants with mixed technical skill, including elderly and visually impaired residents. They need large text, clear actions, and plain Vietnamese. |
| **Managers** | Moderate at a desk once or twice a day, and on a phone when something can't wait. They need to read each case properly and decide fairly. |
| **System Administrators** | Often the building owner. They manage access, filter settings, analytics, and the audit log. |
| **Platforms** | Responsive web. Resident screens are designed for phone first. Manager and Admin screens are designed for desktop first, with full phone support for moderation. |
| **Language** | Vietnamese only for the first release, with text kept ready to translate. Pages declare `lang="vi"`. |
| **Tech stack** | Vue 3, TypeScript, Vite, Vue Router, Pinia, vue-i18n. |
| **UI library (proposed)** | shadcn-vue on Reka UI with Tailwind CSS. [SYSTEM_DESIGN.md](SYSTEM_DESIGN.md) currently names Reka UI with custom styling, so approving this adds shadcn-vue and Tailwind. |

## 1. Visual Identity & Design Tokens

### 1.1 Aesthetic and vibe

**Design language:** clean, modern community forum. It should feel calm and trustworthy, not corporate or flashy.

| Do | Avoid |
|---|---|
| Clear hierarchy, with strong post titles and comfortable reading text | Decorative gradients, glassmorphism, and glow effects |
| White cards on a soft cool ground, with restrained shadows | Warm cream backgrounds, which read as a generic AI default |
| One primary accent, with colour reserved for meaning | Several tags on every row, or colour used as decoration |
| Visible text labels on navigation and actions | Icon-only actions, except standard controls with accessible names |
| Initials avatars, because the forum has no photo uploads | Stock photos or invented residents |

**Light mode only for the first release.** Dark mode is left out.

### 1.2 Colour palette

Components use the semantic tokens below and never raw hex values.

#### Brand and neutral colours

| Token | Value | Usage |
|---|---|---|
| `primary` (pine green) | `#0F6B4F` | Main actions, selected items, links, focus ring |
| `primary-hover` | `#0B523C` | Hover and pressed state of primary actions |
| `on-primary` | `#FFFFFF` | Text and icons on `primary` |
| `primary-soft` | `#DFF3EA` | Selected row, current navigation item |
| `on-primary-soft` | `#0B523C` | Text on `primary-soft` |
| `secondary` (indigo) | `#4338CA` | Official reply marker and official notice accents only |
| `on-secondary` | `#FFFFFF` | Text on `secondary` |
| `secondary-soft` | `#ECEBFC` | Official marker background |
| `on-secondary-soft` | `#3730A3` | Text on `secondary-soft` |
| `background` | `#F5F6FA` | Page ground |
| `surface` | `#FFFFFF` | Cards, panels, top bar, dialogs |
| `surface-subtle` | `#EEF0F6` | Hover, decision panel, quiet blocks |
| `border` | `#DDE0E8` | Decorative dividers and card edges |
| `control-border` | `#7A8091` | Inputs, checkboxes, outline buttons |
| `text` | `#171A23` | Headings and body text |
| `text-muted` | `#525866` | Metadata and supporting text |

#### Semantic colours

| Meaning | Solid | Soft background | Soft text | Usage |
|---|---|---|---|---|
| `success` | `#15803D` | `#E3F6E9` | `#166534` | Saved, approved, restored |
| `warning` | `#B45309` | `#FDF1D8` | `#7C3F06` | Pending review, flagged text, pre-send warnings, model outage |
| `danger` | `#B91C1C` | `#FCEBEA` | `#991B1B` | Reject, delete, errors, high severity |
| `info` | `#1D4ED8` | `#E4EDFD` | `#1E40AF` | Neutral notices and help |

White text is used on solid semantic colours.

**Usage rules**
- Colour never carries meaning alone. Severity and status also use an icon and text.
- `primary` and `success` are close greens (1.29:1 against each other), so success messages always include a check icon and text.
- Channel tags use one neutral style, not one colour per channel.
- Avatars use four soft tints (`#E8E5F9`, `#DCEFE6`, `#FBE9D5`, `#F7E2E7`), chosen consistently from the account ID, with `#171A23` initials. They carry no meaning.
- Primary and danger never appear as two filled buttons side by side, except in the final confirmation of a rejection.

#### Verified contrast

Calculated with the WCAG formula.

| Pair | Ratio | Pair | Ratio |
|---|---:|---|---:|
| `text` on `surface` | 17.38 | White on `primary` | 6.49 |
| `text` on `background` | 16.09 | White on `secondary` | 7.90 |
| `text-muted` on `surface` | 7.13 | White on `success` | 5.02 |
| `text-muted` on `background` | 6.60 | White on `warning` | 5.02 |
| `text-muted` on `surface-subtle` | 6.25 | White on `danger` | 6.47 |
| `control-border` on `surface` | 3.94 | White on `info` | 6.70 |

Soft pairs measure 6.32–8.45.

### 1.3 Typography

**Font family:** Be Vietnam Pro, self-hosted with its Vietnamese subset, falling back to `system-ui, sans-serif`. Weights used are 400, 500, 600, and 700. Numbers use tabular figures in tables and scores.

| Style | Size (px / rem) | Line height | Weight | Usage |
|---|---|---|---:|---|
| H1 | 32 / 2.000 | 40px | 700 | Page title on large screens |
| H2 | 26 / 1.625 | 34px | 700 | Post and case title |
| H3 | 22 / 1.375 | 30px | 700 | Section title; post title on phone |
| H4 | 18 / 1.125 | 26px | 600 | Card and panel title |
| H5 | 16 / 1.000 | 24px | 600 | Group title, author name |
| H6 | 14 / 0.875 | 20px | 600 | Small group label |
| Body large | 18 / 1.125 | 29px | 400 | Post and flagged comment text |
| Body | 16 / 1.000 | 24px | 400 | Default interface and comment text |
| Body small | 14 / 0.875 | 20px | 400 | Metadata on dense manager screens |
| Caption | 13 / 0.8125 | 18px | 500 | Timestamps and table notes; never for required instructions |

**Rules**
- Body text is at least 16px on resident screens.
- Reading width is 65–75 characters.
- Letter spacing is `-0.02em` on H1–H3 and normal elsewhere. Avoid all-caps Vietnamese body text, because it makes diacritics harder to read.
- Interfaces must stay usable at 200% browser zoom.

### 1.4 Shape and elevation

| Token | Value | Usage |
|---|---|---|
| `radius-sm` | 8px | Small buttons, tags inside dense rows |
| `radius-md` | 12px | Buttons, inputs, list rows |
| `radius-lg` | 16px | Cards, panels |
| `radius-xl` | 20px | Dialogs and main reading card |
| `radius-full` | 999px | Avatars, tags, counts |
| `shadow-sm` | `0 1px 2px rgba(23,26,35,.05)` | Cards at rest |
| `shadow-md` | `0 1px 2px rgba(23,26,35,.05), 0 4px 14px rgba(23,26,35,.07)` | Raised cards, sticky bars |
| `shadow-lg` | `0 8px 28px rgba(23,26,35,.14)` | Dialogs and menus |

## 2. Layout & Spacing Architecture

### 2.1 Breakpoints and grid

| Name | Width | Columns | Gutter | Page margin | Layout |
|---|---|---:|---:|---:|---|
| Phone | 320–639px | 4 | 12px | 16px | One column, bottom navigation for residents |
| Large phone / small tablet | 640–767px | 8 | 16px | 20px | One column with wider content |
| Tablet | 768–1023px | 8 | 16px | 24px | Two panes where useful; side navigation collapses to a menu |
| Desktop | 1024–1439px | 12 | 24px | 24px | Navigation and content; manager queue plus reader |
| Wide | 1440px and up | 12 | 24px | 32px | Content capped at 1440px; optional context column |

Layout is designed down to 320px, and no page scrolls sideways.

### 2.2 Spacing scale (base-4)

| Token | Value | Typical use |
|---|---:|---|
| `space-1` | 4px | Icon and text gap in tight controls |
| `space-2` | 8px | Inside tags; related items |
| `space-3` | 12px | Inside list rows; button groups |
| `space-4` | 16px | Default component padding; phone margin |
| `space-5` | 20px | Compact card padding |
| `space-6` | 24px | Default card padding; grid gutter |
| `space-8` | 32px | Between major sections |
| `space-10` | 40px | Space above a page section |
| `space-12` | 48px | Large empty-state spacing |
| `space-16` | 64px | Page-level separation on wide screens |

Keep related items closer together than unrelated ones, and leave more space above a heading than below it.

### 2.3 Navigation structure

**Residents**
- **Desktop and tablet:** top bar with logo, search if added later, notifications, and account menu; left sidebar with channels.
- **Phone:** top bar with logo and notifications; bottom navigation with Trang chủ, Kênh, Đăng bài, Thông báo, Tài khoản. The channel switcher opens as a listbox dialog.

**Managers**
- **Top bar:** logo, Cần xử lý with a count, Đã xử lý, Thông báo, Về diễn đàn, account menu.
- **Desktop:** grouped queue on the left, reader on the right.
- **Phone:** queue and case are separate screens, with a back action.

**Admins**
- Manager navigation plus a Quản trị section: Người dùng, Bộ lọc AI, Từ khóa chặn, Phân tích, Nhật ký kiểm tra.

**Filters:** the queue and tables use one popover with several attributes, such as channel, type, and date. Active filters appear as removable chips, with a "Xóa bộ lọc" action.

## 3. Component Library Specifications

### 3.1 Shared states

| State | Rule |
|---|---|
| Default | Meets contrast requirements, and the control looks interactive |
| Hover | Colour shift in 120ms; hover alone never reveals required information |
| Active | Darker fill, without moving the layout |
| Focused | 2px `primary` ring with a 2px offset, shown for keyboard focus |
| Disabled | 60% opacity with no pointer action. Add visible text explaining why when the reason isn't obvious |
| Loading | Keeps its width, shows a small spinner, and sets `aria-busy` |

### 3.2 Buttons

**Anatomy:** optional 18px leading icon, label, optional trailing icon. Height is 44px by default and 36px only for pointer-based dense manager tools. Radius is `radius-md`.

| Variant | Default | Hover | Active | Disabled | Use |
|---|---|---|---|---|---|
| **Primary** | `primary` fill, `on-primary` text | `primary-hover` | Darkened `primary-hover` | 60% opacity | One main action per area |
| **Secondary** | `surface` fill, `control-border`, `text` | `surface-subtle` | Darker subtle fill | 60% opacity | Alternative actions; both moderation outcomes start here |
| **Ghost** | Transparent, `primary` or `text` | `surface-subtle` | Darker subtle fill | 60% opacity | Low-priority actions |
| **Danger** | `danger` fill, white text | Darker red | Darker red | 60% opacity | Final confirmation only, such as "Gửi và từ chối" |
| **Icon** | 44px square, transparent | `surface-subtle` | Darker subtle fill | 60% opacity | Standard actions only, with `aria-label` and a tooltip |

All variants show the shared focus ring and loading state.

### 3.3 Inputs and forms

**Anatomy:** visible label above, input, optional hint, error text below. Placeholder text never replaces a label.

- **Text field and textarea**
  - **Default:** 48px high on resident screens, 44px on manager screens, `control-border`, `radius-md`.
  - **Hover:** darker border.
  - **Focused:** `primary` border and focus ring.
  - **Error:** `danger` border, an icon and message below, and `aria-invalid`. Keep what the user typed.
  - **Disabled:** `surface-subtle` fill, with the reason shown nearby.
- **Dropdown (select):** Reka UI select with a drawn chevron and full keyboard support. Use it for short lists of up to about 7 items.
- **Listbox switcher dialog:** used for channels on phone and long lists.
  - Opens as a bottom sheet on phone and a centred dialog on desktop.
  - Each row is at least 44px, with the current item checked. A search field appears only for more than 10 items.
  - Default, hover, selected, and focused rows follow the shared states. Arrow keys move and Enter selects.
- **Radio list:** used to choose a community rule, because the options must stay visible. AI-suggested rules come first, and the others sit behind "Quy tắc khác".
- **Checkbox and switch:** minimum 44px clickable area including the label.
- **Pre-send warning:** a `warning-soft` block directly under the composer, naming the rule when known. It never blocks typing and never counts as a violation.
- **Number and threshold input:** a labelled number field with a value from 0 to 1, plus plain text describing the effect. Save needs confirmation and reports success.

### 3.4 Data display

**Cards**
- **Anatomy:** optional header, content, optional footer. `surface`, `radius-lg`, `shadow-sm`, `space-6` padding.
- **Clickable cards:** the title is the link. Hover adds `shadow-md`, and focus shows the ring.
- **Post card:** title, author avatar and name, time, channel tag, excerpt, comment count, and official marker where relevant.
- **Do not nest** bordered cards inside cards. Use dividers or `surface-subtle` blocks.

**Data tables**
- **Anatomy:** header row, rows, pagination or "Tải thêm", optional bulk action bar.
- **Default row height:** 52px; compact 44px on Admin screens.
- **Row states:** hover uses `surface-subtle`; selected uses `primary-soft` with a checked box; focused shows the ring; disabled uses muted text and a reason.
- **Sorting:** the header is a button and sets `aria-sort`.
- **Phone:** rows become stacked label and value cards. Long tables don't scroll sideways unless they are read-only logs with a visible scroll hint.

**Queue rows**
- **Anatomy:** initials avatar, title, one line of metadata. At most one state marker per row.
- **Selected:** `primary-soft` fill, `primary` ring, and `aria-current="true"`.

**Modals**
- Use only when focus must be protected: permanent or final actions, the channel switcher, removing published content, and appeal forms on phone.
- **Anatomy:** title, content, then actions on the right. Maximum width is 560px; on phone they appear as a bottom sheet.
- **Behaviour:** focus moves inside and is trapped, Esc closes unless work would be lost, and focus returns to the opening control.
- Routine moderation decisions stay inline.

**Tags and markers**
- **Channel tag:** neutral `surface-subtle`, `text-muted`, `radius-full`, 13px.
- **Official reply marker:** `secondary-soft` with a check icon and the text "Phản hồi chính thức".
- **Status marker:** icon plus text, such as "Đang chờ xem xét".

## 4. Key Screens & View Hierarchies

### 4.1 Resident home and channel feed

- **Primary objective:** see official news first, then find or start a discussion.
- **Structure**
  - Top bar.
  - Left channel list on desktop; bottom navigation and a channel switcher on phone.
  - Main column, up to 760px: pinned announcements, then "Tạo bài viết" for allowed channels, then the post feed.
  - Optional right column at 1280px and up: community rules and new replies.
- **Data density:** low to medium. Show 5–7 post cards in the first desktop screen and 3–4 on phone.
- **Visual hierarchy**
  1. Official announcements.
  2. Post titles.
  3. Excerpts.
  4. Author, time, and comment count.
- **Permission rule:** in Tin ban quản lý, residents see comment actions but not "Tạo bài viết".

### 4.2 Post thread and composer

- **Primary objective:** read a post, reply safely, and understand any moderation result.
- **Structure**
  - Back link and channel tag.
  - Post title, author, time, and full text at body-large size.
  - Comments nested up to two levels, with a placeholder for hidden or deleted comments.
  - Composer below the thread on desktop, and in a sheet opened by a fixed "Trả lời" button on phone.
  - Actions for own content (edit, delete). Managers also see "Gỡ nội dung" on residents' content.
- **Data density:** low. Comfortable reading comes first.
- **Visual hierarchy**
  1. Post content.
  2. Official replies.
  3. Other comments.
  4. Actions and metadata.
- **Key states**
  - **Pre-send warning:** appears under the composer with a chance to rephrase.
  - **Pending review:** the author sees their item marked "Đang chờ xem xét", using the message in COMMUNITY_RULES.md. Others don't see it.
  - **Rejected:** shows the rule, deadline, and a request for review.
  - **Read-only account:** the composer is replaced by the end time and reason.
  - **Privacy reminder:** a short note in the composer on Maintenance and Marketplace posts.
  - **Length limit:** from 4,500 characters, the composer shows what remains, up to the 5,000-character limit. Over it, the count turns red and submitting shows an error. The draft is kept and never cut.

### 4.3 Manager moderation dashboard

- **Primary objective:** make a fair decision on each case, then move to the next.
- **Structure**
  - **Desktop left column (360px):** filters, multi-select, and queue groups in this order: Xem xét lại, Mức cao, Cần xem, Từ khóa chặn.
  - **Desktop reader (fluid, up to 860px):**
    - Title and channel tag.
    - Previous message.
    - Flagged comment at body-large size.
    - Decision panel directly beneath the comment.
  - **Wide screens (optional):** a context column for AI scores, author history, and current step.
  - **Phone:** queue screen and case screen. The case shows a back action and all decision controls, including the exact resident message.
- **Decision flow**
  1. One plain sentence explains why the case is here.
  2. Two equal choices appear, such as Từ chối and Cho phép đăng.
  3. Rejecting reveals rule choices and the exact resident message before final confirmation.
  4. Hidden text stays hidden until the manager chooses to reveal it, and the server sends it only then.
  5. A warning appears if the manager decides without reading hidden text.
  6. After the decision, the next row opens, and the resident is notified at once.
- **Data density:** medium in the queue, low in the reader. The first desktop screen shows about 8–10 queue rows.
- **Visual hierarchy**
  1. Flagged comment.
  2. Reason.
  3. Decision.
  4. Thread context.
  5. Queue.
  6. AI scores, which appear in plain language first and in detail on request.
- **Special rules**
  - `hate` flags require the manager to choose Rule 1, 2, or 3.
  - `offensive` suggests Rule 4.
  - Text in other languages shows no score or rule, and doesn't count as a violation.
  - Removing published content needs a rule. It counts as a confirmed violation, can be appealed, and is logged as a false negative.
  - Bulk decisions require a rule and support deselecting individual items.

### 4.4 Admin filter settings and analytics

- **Primary objective:** tune moderation safely and judge whether the AI is working.
- **Structure**
  - Admin navigation on the left, and a menu on phone.
  - **Bộ lọc AI tab:**
    - Global thresholds for auto-publish and quarantine, plus an auto-delete switch, off by default, with its threshold.
    - Per-channel overrides in a table.
    - Model outage policy.
  - **Từ khóa chặn tab:** searchable table with term, linked rule, and actions.
  - **Phân tích tab:** date range; flagged volume over time; AI accuracy; flag rate by label; active warnings.
  - **Nhật ký kiểm tra tab:** read-only, filterable table with no edit controls.
- **Data density:** high, using compact tables on desktop and stacked rows on phone.
- **Visual hierarchy**
  1. Current policy and any model outage.
  2. Unsaved changes.
  3. Settings values.
  4. Supporting analytics and logs.
- **Guardrails**
  - Auto-delete is off by default, and the setting states that it applies only to `offensive`.
  - Changing the outage policy needs confirmation and is logged.
  - Charts include table alternatives and don't rely on colour alone.
  - Sample or stand-in model data is clearly labelled.

## 5. Interaction, States & Accessibility

### 5.1 State management

| State | Visual rule | Content rule |
|---|---|---|
| **Loading, under 300ms** | Show nothing new | Avoid flicker |
| **Loading, page or list** | Skeleton matching the final layout | Set `aria-busy`; never use a spinner in place of a list |
| **Loading, an action** | Small spinner in the button, keeping its width | Label stays visible, or changes to the action in progress |
| **Empty** | Short heading, one sentence, and at most one action | Explain what belongs here and what to do next |
| **Filtered empty** | Same layout | Name the active filter and offer "Xóa bộ lọc" |
| **Inline error** | `danger` text and icon beside the field | State the problem and the fix; keep the user's input |
| **Page or load error** | Message in place of the failed area, with "Thử lại" | Keep the rest of the page usable |
| **Save failure** | Persistent `danger` notice with "Thử lại" | The item stays in place; no action is lost silently |
| **Model outage** | Persistent `warning` banner for Managers and Admins | Explain that posts are held and rescanned automatically under fail closed |
| **Offline connection** | Slim `warning` banner | Keep drafts in the browser where possible |

### 5.2 Motion and feedback

| Interaction | Duration | Easing | Behaviour |
|---|---:|---|---|
| Hover and colour change | 120ms | `ease-out` | Colour only |
| Button press | 100ms | `ease-out` | Fill change, no layout shift |
| Menu, popover, tooltip | 150ms | `ease-in-out` | Fade plus 4px movement |
| Dialog or sheet | 200ms | `cubic-bezier(.16,1,.3,1)` | Fade and rise; exit in 150ms |
| Expand a panel | 180ms | `ease-in-out` | Reveal rule choices and message |
| Next case after a decision | 220ms | `cubic-bezier(.16,1,.3,1)` | The one signature moment: the next case rises in and focus moves to its title |
| Toast or status | Appears in 150ms | `ease-out` | Must not cover the next action |

**Rules**
- Motion only communicates a change of state.
- Under `prefers-reduced-motion`, remove movement and keep instant state changes.
- Actions confirm in place. Avoid toasts when an inline change is clear.

### 5.3 Accessibility

The product requires **WCAG 2.1 Level AA**. The 44px touch target goes beyond AA and is a project rule because of the audience.

| Area | Requirement |
|---|---|
| **Text contrast** | At least 4.5:1, or 3:1 for large text |
| **Non-text contrast** | At least 3:1 for control boundaries, icons, and focus indicators |
| **Focus ring** | 2px `primary`, 2px offset; never removed; never hidden by sticky bars |
| **Touch targets** | At least 44 × 44px, with 8px between adjacent targets |
| **Keyboard** | Every action works by keyboard, in visual order. Include a skip link. |
| **Focus management** | Moves to new page or case titles and returns after dialogs close. Re-rendering never drops focus to the page body. |
| **Screen readers** | Use semantic landmarks and heading order, and label every form field. Announce status with `aria-live` and connect errors with `aria-describedby`. |
| **Flagged and hidden text** | Conveyed in text as well as colour, using `<mark>` plus a hidden label and a named reveal button |
| **Colour use** | Never the only signal of status, score, or severity |
| **Language** | `lang="vi"`, with other languages marked on the relevant content |
| **Zoom and reflow** | Usable at 200% zoom and at 320px wide without sideways scrolling |
| **Testing** | axe in Playwright, keyboard walkthroughs, NVDA or VoiceOver checks, and at least one review with an older resident |

**Optional shortcuts for Managers:** J and K move between cases, H reveals hidden text, R rejects, A allows, and ? opens help. Shortcuts never fire while typing, and every action also has a visible control.

## Decisions to confirm

1. **Primary colour:** decided: golden sun `#FFC53D` from the Sunshine brand, with ink text on it. Pine green and the kit's orange were both dropped. `DESIGN.md` holds the current tokens.
2. **UI library:** shadcn-vue and Tailwind need approval, because they change the current system design.
3. **Dark mode:** decided: left out of the first release, along with the text-size control. Large text is the default for everyone.
4. **Undo window:** decided: left out of the first release. Decisions take effect and notify at once, and an appeal is the way to correct one.
5. **Vietnamese channel names:** the announcements channel is decided as "Tin ban quản lý" (2026-11-08), so "Thông báo" means notifications only. Thảo luận chung and Bảo trì are working names, and `Marketplace` still needs a decision.
6. **Search:** decided: search stays, in the top bar and at `#/tim`, covering posts, notices and comments.
