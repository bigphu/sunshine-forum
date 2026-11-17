# Specification Changes

Changes needed to bring the project specification in line with [PRODUCT.md](../../PRODUCT.md). The specification isn't in this repository, so "Current" quotes the copy pasted during setup. Check it against your latest version.

**Summary:** 18 requirements change, 8 are added, and the Scope section needs updating. All are decided. New IDs are proposals, so change them if they clash with your numbering.

**Withdrawn:** FR-2.6 (urgent notices), US-4.4, and FR-4.6 (resident reporting) were proposed earlier and left out to simplify the first release. Their IDs aren't reused, which is why the numbering has gaps.

## 2.2 Scope

**Channels**

Current:
> Discussion boards categorized by topics (e.g., General Announcements, Services, Marketplace).

Change to:
> Discussion boards categorized by topic: General Announcements, General Discussion, Maintenance, and Marketplace.

**Add to the out-of-scope list:**
> Tracked maintenance tickets. Maintenance requests are ordinary posts in the Maintenance channel.

## US-1: Access, Identity & Penalty Management

**US-1.1 Resident Verification**

Current:
> As an apartment owner, I want to verify new account registrations against an active tenant registry, so that unauthorized external users cannot access the community forum.

Change to:
> As a system administrator (a role often held by the building owner), I want to invite residents from the active tenant registry and deactivate them when their tenancy ends, so that unauthorized external users cannot access the community forum.

Invitation replaces self-registration: the system emails each invited resident a link to set a password, so there is no public sign-up form and no refused-applicant flow. The invitation link expires after 3 days, and sending a new invitation cancels the previous link. Password reset links expire after 1 day.

Why: the three roles don't include Owner.

**US-1.3 Account Penalties**

Current:
> As an apartment manager, I want the system to automatically apply restrictions (e.g., read-only timeouts or shadow bans) to users with repeated toxicity violations, so that bad actors are efficiently handled.

Change to:
> As an apartment manager, I want the system to automatically apply read-only timeouts to users with repeated confirmed toxicity violations, so that bad actors are efficiently handled.

Why: shadow bans conflict with FR-4.5, which requires telling users about penalties.

## FR-1: Identity and Access Management

**FR-1.3**

Current:
> The system must automatically apply read-only restrictions or shadow bans to accounts that repeatedly violate toxicity thresholds over a defined rolling period.

Change to:
> The system must automatically apply read-only restrictions to accounts with repeated confirmed violations over a rolling period. Administrators configure the limits and the period. Shadow bans are not used.

**FR-1.4 (new)**
> Only confirmed violations count toward penalties. Pre-send warnings and content pending review do not count, and a successful re-evaluation request removes the violation.

## FR-2: Content Management & Engagement

**FR-2.1**

Current:
> The system must allow authenticated users to create, read, update, and delete (CRUD) their own forum posts and comments.

Change to:
> The system must allow authenticated users to create, read, update, and delete (CRUD) their own forum posts and comments. Only Managers and Administrators can create posts in General Announcements; Residents can comment there.

**FR-2.3**

Current:
> The system must group discussions into categorized channels (e.g., General Announcements, Maintenance, Marketplace).

Change to:
> The system must group discussions into categorized channels: General Announcements, General Discussion, Maintenance, and Marketplace.

Why: with announcements closed to resident posts, residents need somewhere for the discussion in US-2.1.

**FR-2.4 (new)**
> Posts must support nested comments, capped at two levels. A hidden or deleted comment must leave a placeholder so its replies remain readable. Downvotes are not supported.

**FR-2.5 (new)**
> Replies from Managers and Administrators must carry an official reply marker. The system must remind residents not to share private details in posts.

**FR-2.7 (new)**
> Post bodies and comments must be limited to 5,000 characters. The composer must show the remaining characters near the limit, refuse to submit text over it, and keep the draft.

Why: the toxicity model accepts up to 5,000 characters, so longer text could not be fully scored.

## US-3 and FR-3: AI-Powered Toxicity Filtering

**US-3.2 Pre-Publication Scanning**

Current:
> As an apartment manager, I want all newly submitted posts and comments to be automatically evaluated by an AI toxicity filter before they are published, so that abusive language never reaches the public feed.

Change to:
> As an apartment manager, I want all newly submitted posts and comments to be automatically evaluated by an AI toxicity filter before they are published, so that abusive language does not reach the public feed while the filter is running or the system has failed closed.

Why: NFR-4.2 allows Administrators to choose fail open, so "never" can't be guaranteed.

**FR-3.1**

Current:
> FR-3.1 (Pre-Send Evaluation): The system must evaluate draft content on the client/edge before submission, returning real-time warnings to the user if the text violates community guidelines.

Change to:
> FR-3.1 (Pre-Send Evaluation): The system must evaluate draft content before submission, returning real-time warnings to the user if the text may violate community guidelines. The keyword blocklist runs in the browser, and the model evaluates the draft on the server after a short typing pause. Drafts are not stored.

Why: the model team confirmed the model can't run in the browser.

**FR-3.4 (new)**
> The toxicity model must evaluate Vietnamese text. The model service must normalize missing diacritics, teencode, and spaced-out letters before scoring. The forum backend must check the language first and send text in other languages to "Pending review" without scoring it.

Why: the model has no language detection, so it could score abuse in another language as clean.

**FR-3.5 (new)**
> The system must score posts sentence by sentence and use the highest score. Before launch, model accuracy must be tested on ViHSD's test set and, separately, on a Vietnamese test set of building discussion written and labelled by the forum team, including longer posts and ordinary complaints. Both results count toward the pass mark.

Why: the model's training data is comment-based, and the model team has no data for posts or complaints.

## US-4 and FR-4: Moderation Workflow

**US-4.1 Moderation Dashboard**

Current:
> As an apartment manager, I want a centralized dashboard displaying all quarantined content, so that I can efficiently read, approve, or permanently delete flagged posts.

Change to:
> As an apartment manager, I want a centralized dashboard displaying all quarantined content, so that I can efficiently read, approve, or reject it.

**FR-4.1**

Current:
> The system must provide a secure dashboard for Managers to view, approve, or permanently reject quarantined posts.

Change to:
> The system must provide a secure dashboard for Managers to view, approve, or reject quarantined content. Content is permanently rejected only after a re-evaluation request has been decided. Managers must also be able to remove published content the filter missed, by choosing a rule. The removal counts as a confirmed violation and can be appealed, and the system logs it as a false negative.

Why: FR-5.3 tracks false negatives, and without resident reporting this is the only way to record them.

**FR-4.4**

Current:
> The system must allow penalized Residents to submit a structured "Re-evaluation Request", moving their rejected content into a priority moderation queue.

Change to:
> The system must allow Residents to submit a structured "Re-evaluation Request", moving their rejected or auto-deleted content into a priority moderation queue. Content still pending review cannot be appealed. Rejected or auto-deleted content can be appealed once, within 30 days. An Administrator or a different Manager decides where possible, and that decision is final.

**FR-4.5**

Current:
> The system must automatically send a formatted system message detailing the specific violated rule when a user's content is rejected or an account is penalized.

Change to:
> The system must automatically send a formatted system message detailing the specific violated rule when a user's content is rejected or an account is penalized. The message must name a community rule, not a model label: `offensive` names Rule 4, and a manager chooses among Rules 1 to 3 for `hate`. Managers must choose a rule before rejecting content, including in bulk.

## US-5: Transparency & Appeals

**US-5.2 Appeals System**

Current:
> As an apartment resident, I want to submit a re-evaluation request for my quarantined or rejected post, so that a human manager can correct any AI misclassifications (false positives).

Change to:
> As an apartment resident, I want to submit a re-evaluation request for my rejected or auto-deleted post, so that a human can correct any wrong decisions.

Why: appeals now also cover manager rejections, not only AI mistakes. Pending content is already waiting for a human, so it isn't appealable.

## US-6 and FR-5: System Administration

**US-6.4 Override Audit Log**

Current:
> As a system administrator, I want an uneditable audit log of every time a human moderator overturns an AI decision, so that I can use this data to retrain the model and patch its blind spots.

Change to:
> As a system administrator, I want an uneditable audit log of every time a human moderator overturns an AI decision, and a way to export finalized corrections, so that the model can be retrained and its blind spots patched.

Why: the audit log no longer holds post text.

**FR-5.1**

Current:
> The system must provide an administrative interface to modify the numerical confidence thresholds for auto-publishing, quarantining, and auto-deleting without requiring a server restart.

Change to:
> The system must provide an administrative interface to modify the numerical confidence thresholds for auto-publishing, quarantining, and auto-deleting without requiring a server restart. Auto-deleting is off by default. An Administrator can turn it on, and it applies only to the `offensive` label.

Why: a linear SVM sees word patterns, not meaning, so removing content with no human check is the riskiest use of it. Until it has been tested on building content, a manager reviews this content first.

**FR-5.3**

Current:
> The system must log an immutable audit trail of all manual moderator actions, specifically tracking when a Manager overturns an AI decision (false positive/negative tracking).

Change to:
> The system must log an immutable audit trail of all manual moderator actions, specifically tracking when a Manager overturns an AI decision (false positive/negative tracking). The trail stores the action, moderator, AI score, model version, rule, outcome, and time, with a pseudonymous user ID. It holds no post text or resident personal data, and no one can edit it, including Administrators.

**FR-5.5 (new)**
> The system must allow an Administrator to export finalized corrections for a training or evaluation run, 30–90 days after the decision. Exports exclude private-information cases and author-deleted content, contain no user or post IDs, and have phone numbers, emails, and similar patterns replaced automatically. The export must be deleted before the originals reach the 90-day purge.

## Non-functional Requirements

**NFR-3.2 (Session Security)**

Current:
> The system must use secure, HTTP-only, SameSite cookies or short-lived encrypted JWTs for authentication, actively preventing Cross-Site Scripting (XSS) and Cross-Site Request Forgery (CSRF).

Change to:
> The system must use secure, HTTP-only, SameSite session cookies for authentication, actively preventing Cross-Site Scripting (XSS) and Cross-Site Request Forgery (CSRF). Role changes and penalties must apply on the next request.

**NFR-3.3 (Data Retention)**

Current:
> Hard-deleted content and expired tenant accounts must be completely purged from the primary database and analytics logs after 90 days to comply with standard privacy retention policies.

Change to:
> Deleted content is soft-deleted for 90 days, which keeps it available for appeals. Deleted content and expired tenant accounts must then be completely purged from the primary database, analytics logs, and system event records to comply with standard privacy retention policies. The audit trail is retained because it holds no content or directly identifying data.

**NFR-4.2 (Graceful Degradation)**

Current:
> In the event the AI toxicity model goes offline or times out, the system must fail open or fail closed based on a configurable Admin policy (e.g., automatically quarantining all posts until the AI is restored) rather than crashing the submission API.

Change to:
> In the event the AI toxicity model goes offline or times out, the system must fail open or fail closed based on a configurable Admin policy rather than crashing the submission API. The default is fail closed: posts are quarantined and automatically rescanned when the model returns. Under fail open, the keyword blocklist still runs, and published posts are rescanned on recovery and hidden if flagged. Comments in General Announcements remain quarantined under either policy. Policy changes are logged.

**NFR-5.3 (new)**
> The platform is Vietnamese first. The interface is Vietnamese-only for the initial release, with text kept easy to translate later. Pages must declare their language for screen readers, and all interface text must render Vietnamese diacritics correctly.
