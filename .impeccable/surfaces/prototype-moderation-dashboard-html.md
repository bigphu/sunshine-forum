---
version: 1
slug: "prototype-moderation-dashboard-html"
primary_target: "prototype/moderation-dashboard.html"
related_targets: []
---

# Moderation dashboard

**Scope and mode:** clickable prototype of the manager moderation dashboard; Operate.
**Audience and job:** property managers, at a desk once or twice a day and on a phone for urgent items, making careful single decisions about posts by neighbours.
**Proof and content:** sample data only, always labelled. No real residents, scores, or rules text beyond the draft community rules.
**Unresolved:** whether rejecting has a 10-second undo (prototyped as a proposal), whether AI can highlight exact phrases (whole sentence for now), Vietnamese channel names.

## Direction contract

THESIS: A modern community-forum look, held to restraint, where one case is read as a conversation with the verdict docked right beneath it. It refuses pill-covered rows, decorative gradients, and any gap between evidence and decision.

OWN-WORLD: Soft cool-neutral ground, as in the first reference image, floating white rounded cards with soft shadows, bold tight case titles, grouped rounded queue sections with uppercase headers and counts, soft-tinted initials avatars, one indigo accent for selection and main actions, one channel tag, amber for warning, red for danger. Be Vietnam Pro throughout, following the user's reference images. No gradients or glass.

STORY: The manager sees grouped work by priority, reads the comment in context, understands one plain sentence about why it is here, chooses whether to view hidden text, picks an outcome, reads exactly what the resident will be told, then moves to the next row with undo available.

FIRST VIEWPORT: Slim top bar with logo mark and tabs. Left: 360px column of grouped queue cards. Right: a white rounded card with bold title, channel tag, threaded previous message, the flagged comment, then a stone decision panel directly beneath it holding the reason and two equal outcome buttons.

FORM: Category standard, inbox and thread, styled toward the user's two reference images with shadcn-style tokens in plain CSS. Critique composition 1, chosen by the user. Signature interaction: outcome first, then rule and exact resident message, with hidden text requested only on reveal. seed:e404a9deb09cc45f

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
