# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Vue.js frontend and Python model service, with PostgreSQL. The backend isn't tied to a language: whoever builds it can use TypeScript, JavaScript, Go, or another stack that provides the capabilities listed in `docs/specs/SYSTEM_DESIGN.md`. It deploys at `sunshine-forum.bigphu.io.vn` on the VPS shared with StoreIt, and the model service may run on a different machine. Details are in `docs/specs/SYSTEM_DESIGN.md`, a draft awaiting review. The first toxicity model is a linear SVM.

## Users

- **Apartment residents** (verified tenants): read verified news and building announcements, post, join discussions, submit feedback, and engage with neighbours in a safe space. About 100 are expected, including elderly and visually impaired residents.
- **Property managers** (content moderators): broadcast official notices, review flagged posts, manage community inquiries, and rely on automated filtering to moderate efficiently.
- **Apartment owner / system administrator**: the same role, System Administrator, which is often held by the building owner. Manage access rights, verify registrations, configure toxicity sensitivity, monitor analytics, and oversee operations.

## Product Purpose

Sunshine Apartment Forum (full title: Community Communication & Moderation Platform) is a secure, centralized communication hub between property management and residents of Sunshine Apartment. It replaces fragmented, informal channels with one moderated platform for official announcements, resident discussion, maintenance requests, and community support.

Success means announcements reach residents on time, neighbours engage, community feedback is simple to collect, and community guidelines are enforced automatically by AI-driven moderation, with humans able to correct it.

## Positioning

A forum limited to verified tenants of one building, where every post is checked by AI before it goes public, writers are warned before they submit, every AI deletion or manager rejection can be appealed once, and every overturned AI decision is audited.

## Operating Context

- **Writing a post or comment:** the draft is evaluated before submission, and the resident can rephrase without penalty. On submission the backend scans it again before publishing. Edits are rescanned.
- **Moderating:** content in the borderline band is hidden as "Pending review" and appears in the manager's queue. Extreme language is blurred until the manager reveals it. Managers can group similar items and act in bulk, and can remove published content the filter missed.
- **Being penalized:** the resident gets a system message naming the violated rule, plus the duration and how to appeal where they apply, and can submit a structured re-evaluation request, which enters a priority queue.
- **Administering:** admins tune thresholds and blocklists without a deploy or restart, per channel where needed, and use analytics and the audit log to judge the model.
- **Maintenance requests:** residents make them as ordinary posts in the Maintenance channel, not as tracked tickets. Managers respond in the comments.
- Used on desktop, tablet, and mobile.

## Capabilities and Constraints

**Identity and access**
- Access by invitation only, for people on the official tenant registry (US-1.1, FR-1.1). The System Administrator adds each resident from the registry (or imports it once at launch), the system emails a link to set a password, and the Administrator deactivates the account when the tenancy ends. There is no public sign-up form. One account per adult; every account has an email address. The invitation link expires after 3 days; the Administrator can send a new one, which cancels the old link. Password reset links expire after 1 day.
- Role-based access with three levels: Resident, Manager, System Administrator. There is no separate Owner role. Administrator is the most powerful role, but cannot edit the audit log (US-1.2, FR-1.2).
- Automatic read-only timeouts for repeated confirmed violations over a rolling period. Shadow bans are not used. Limits and the rolling period are set by Admins (US-1.3, FR-1.3).
- Only confirmed violations count toward penalties. Pre-send warnings and posts still pending review do not, and a successful appeal removes the strike.

**Content and engagement**
- Users create, read, update, and delete their own posts and comments (US-2.1, US-2.2, FR-2.1).
- Channels: General Announcements, General Discussion, Maintenance, Marketplace (FR-2.3).
- Only Managers and Admins can post in General Announcements; Residents can comment there. Residents can post and comment in all other channels.
- Post bodies and comments are limited to 5,000 characters, the model's input limit. The composer shows what remains near the limit, refuses text over it, and keeps the draft (proposed FR-2.7).
- Reddit-style structure: posts with nested comments, capped at two levels. A hidden or deleted comment leaves a placeholder so its replies still make sense. Voting is not in the specification, and downvotes are excluded.
- Manager and Admin replies carry an official reply marker, so residents can see that management responded. Residents are reminded not to share private details, such as when their home will be empty.
- Real-time alerts and push notifications for official announcements and thread replies (US-2.3, FR-2.2).

**AI toxicity filtering**
- Pre-send evaluation, with warnings that explain the violation. The keyword blocklist runs in the browser, and the model checks the draft on the server after a short typing pause. Drafts are not stored (US-3.1, FR-3.1).
- All submissions and edits pass through the model on the backend before being written as "Published" (US-3.2, US-3.3, FR-3.2).
- Content above the quarantine threshold is saved as "Pending review" and hidden from the public feed (US-3.4, FR-3.3).
- The model will be trained on comments, so posts are scored per sentence, using the highest score. Before launch, it is tested on ViHSD's test set and on a Vietnamese set of building discussion written and labelled by the forum team, including longer posts and ordinary complaints. Both count toward the pass mark.
- The model will handle Vietnamese. Its service normalizes missing diacritics, teencode, and spaced-out letters before scoring. The model has no language detection, so the forum backend checks the language first, and text in other languages goes to quarantine without being scored.
- The model scores `clean`, `offensive`, and `hate`. Labels stay internal, and residents always see a community rule: `offensive` names Rule 4, and `hate` covers Rules 1 to 3. A `hate` flag always goes to quarantine, and a manager chooses the rule. Only `offensive` can be auto-deleted by the model, and auto-delete is off by default, so a manager reviews it first. Blocklist matches store their own rule.

**Moderation and appeals**
- Secure dashboard to view, approve, or reject quarantined content (US-4.1, FR-4.1).
- Managers can remove published content the filter missed, by choosing a rule. The removal counts as a confirmed violation, can be appealed like any other rejection, and is logged as a false negative (FR-5.3, FR-5.4).
- Extreme language is blurred or redacted until a manager clicks to reveal it (US-4.2, FR-4.2).
- Multi-select bulk approval and rejection, with grouping of similar posts (US-4.3, FR-4.3).
- Automatic system message naming the violated rule on rejection or penalty (US-5.1, FR-4.5).
- Structured re-evaluation requests move content to a priority queue (US-5.2, FR-4.4). Only rejected or auto-deleted content can be appealed; content still pending review cannot.
- The appeal reason is required, from 20 to 2,000 characters, so it is long enough to judge and never cut short.
- AI auto-deletions and manager rejections can each be appealed once, within 30 days. An Admin or a different manager decides where possible. That decision is final, and only then is the content "permanently" rejected.

**Administration**
- Rule changes are made by the owner only. The rules stay public as a plain page.
- Adjustable thresholds for auto-publish, quarantine, and auto-delete, plus keyword blocklists, with no deploy or restart (US-6.1, FR-5.1). Auto-delete is off by default, and an Admin can turn it on once the model has been tested on building content.
- Per-channel thresholds (US-6.2, FR-5.2). On General Announcements these apply to resident comments as well as official posts.
- Append-only audit trail of manual moderator actions, tracking overturned AI decisions. It stores only the action, moderator, AI score, model version, rule, outcome, and time, with a pseudonymous user ID. It holds no post text or resident personal data, and no one can edit it, including Admins (US-6.4, FR-5.3).
- Minimal retraining export (US-6.4): no separate long-term store. An Admin exports finalized corrections for a training or evaluation run, and the export is deleted before the originals reach the 90-day purge, so NFR-3.3 is unchanged. Details are in `docs/specs/SYSTEM_DESIGN.md`.
- Analytics on flagged volume, AI accuracy, flag rates, and active warnings (US-6.3, FR-5.4).

**Non-functional constraints**
- Threads and the moderation dashboard render in under 2.0 seconds under normal load (NFR-1.1), including at up to 100 concurrent active users (NFR-2.1).
- Toxicity scoring returns in under 200 ms, for both pre-send and backend checks (NFR-1.2).
- TLS 1.2 or higher in transit, and resident PII encrypted at rest (NFR-3.1).
- Secure HTTP-only SameSite session cookies, with XSS and CSRF prevention (NFR-3.2). JWTs are not used, so role changes and penalties apply on the next request.
- Deleted content is soft-deleted for 90 days, which keeps it available for appeals, then permanently purged. Hard-deleted content and expired tenant accounts are purged from the primary database and analytics logs after 90 days (NFR-3.3).
- 99.5% uptime (NFR-4.1).
- If the model is offline or times out, the admin policy decides what happens, and the submission API does not crash (NFR-4.2). The default is fail closed: posts are quarantined and automatically rescanned when the model returns, so managers do not have to clear the backlog by hand. While the model is offline, the composer tells residents that checks are delayed. If an Admin chooses fail open, the keyword blocklist still runs, and published posts are rescanned on recovery and hidden if flagged. Comments in General Announcements are still held under fail open, because that channel reaches every resident and the blocklist cannot catch impersonation or scams. Announcements are held like any other post. Policy changes are logged.
- Fully responsive across desktop, tablet, and mobile (NFR-5.1).

**Out of scope for the initial release:** online rent payments, utility billing, physical maintenance dispatching, and external visitor booking or access control integrations.

**Left out of the first release to keep it simple:** resident reporting, urgent notices and their email, the 10-second undo on decisions, the Rule 6 pre-send warning, the rewrite and repost route, the public rule change log and weekly figures, dark mode, and the text-size control. Email is used only for invitations and password resets.

**Terminology:** "Pending review" is the quarantine state. "Re-evaluation Request" is the appeal. "Quarantine" means hidden from the public feed awaiting a manager. "Permanently rejected" means rejected after an appeal has been decided.

**Where this record refines the specification (main points; the full wording is in `docs/specs/SPEC_CHANGES.md`)**
- **US-3.2:** "never reaches the public feed" applies only while the filter is running or the system has failed closed.
- **FR-1.3, US-1.3:** read-only timeouts are the only automatic penalty; shadow bans are dropped.
- **FR-2.3:** adds General Discussion and uses Maintenance rather than the scope's Services.
- **New FR-2.7:** posts and comments are limited to 5,000 characters.
- **FR-3.1:** the model checks drafts on the server, so "client/edge" is removed.
- **New FR-3.4, FR-3.5:** the backend checks the language, the model service normalizes text, and the forum team supplies a building test set.
- **FR-4.5:** messages name a community rule, not a model label.
- **FR-5.1:** auto-delete is off by default.
- **FR-4.1, FR-4.4:** "permanently" rejected now means after one appeal.
- **FR-4.1:** managers can also remove published content the filter missed, which gives FR-5.3 its false negatives.
- **NFR-3.3:** the 90-day purge begins with a soft-delete stage, and the audit log holds no content.

**Open decisions**
- **Toxicity model:** the Vietnamese model is not trained yet, and it will use ViHSD's labels. The first model is a linear SVM, which must pass an accuracy test before launch. The pass mark will be discussed later. It must also meet the 200 ms limit in NFR-1.2. The open questions about the model are in `docs/specs/SYSTEM_DESIGN.md`.
- **System design:** `docs/specs/SYSTEM_DESIGN.md` is awaiting review, including the choice of backend stack.
- **Removing published content:** a manager removing published content isn't in the original specification. It's proposed in `docs/specs/SPEC_CHANGES.md` and shown in the prototype and Figma, and it needs the owner's confirmation.

## Brand Commitments

The forum uses the Sunshine brand kit (logo, mark, colours, and tokens) and a Vietnamese-first, modern minimal style inspired by Reddit, Blender Artists, Peerlist, Designer News, and Designer Hangout. It is a safe place for residents where toxicity, scams, and illegal content are prohibited, built on safety, human connection, discussion, contribution, and transparent updates. Profanity rules are strict, but members can appeal, and appeals can improve the rules. Text contrast and accessibility requirements take priority over visual references.

## Evidence on Hand

The project specification is the only source so far; it has not been committed to this repository. The Vietnamese toxicity model is not trained yet. Its chosen training data is ViHSD, a public dataset of Vietnamese social media comments, which is not in this repository. Community rules exist as an unapproved draft in `COMMUNITY_RULES.md`. The Sunshine brand kit is on hand, with the assets used so far in `prototype/brand/`. There is no tenant registry data, testimonial, or usage data on hand. Future work must not fabricate these or present placeholder data as real.

## Product Principles

1. **Verified neighbours only.** Access comes from the tenant registry, because the forum's trust depends on knowing that everyone there lives in the building.
2. **Warn before penalizing.** Give residents the chance to rephrase before a post counts against them.
3. **Explain every decision.** A resident affected by moderation can see which rule applied and can ask a human to look again. Penalties are never hidden from the person they affect.
4. **The AI filters; humans have the final word.** Borderline content goes to a manager, overturned decisions are logged, and that record is used to improve the model.
5. **Protect moderators too.** Reviewing harmful content is work with a cost, so exposure is opt-in and repetitive cases can be handled in bulk.

## Accessibility & Inclusion

WCAG 2.1 Level AA is required, including screen reader compatibility and adequate color contrast for elderly or visually impaired residents (NFR-5.2).

The site is Vietnamese first: the interface is Vietnamese-only for now, with text kept easy to translate later. Pages declare `lang="vi"` for screen readers, and all interface text must render Vietnamese diacritics correctly.
