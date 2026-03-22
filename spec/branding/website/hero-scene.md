# Hero Scene Design

Design spec for the Synchestra landing page hero section — the signature brand moment where red curtains animate open to reveal the orchestra on stage, gathered around a workbench studying blueprints for a theater they're going to build together.

## Layout

The hero section fills the viewport height (`100vh`). It is split into two visual zones:

- **Upper 60% — white backdrop:** Headline, subheadline, and CTA buttons. Text appears as if on the back wall of the stage.
- **Lower 40% — beige stage floor:** The illustration lives here, with a dark edge line (`border-top`) at the boundary. Background color is CSS (`#f5f0e8` → `#ebe4d6` gradient), not part of the illustration.

Curtains are CSS elements framing the sides, animated independently from the illustration.

## Scene Narrative

The architect stands at a solid woodworking workbench at center stage, showing the orchestra a blueprint of the theater they're going to build together. The stage is subtly under construction — a few planks leaning against something, a sawhorse in the background — but the focus is on the characters, not the construction.

This is the "rallying the team" moment: a leader sharing a vision, a diverse group deciding whether to buy in.

### Viewpoint

Slightly elevated audience POV, looking at the stage from approximately the 5th row center.

### Scene Zones

**Zone 1 — At the workbench (center):**
- The Architect stands at the near side of the workbench, facing the audience at a 3/4 angle, pointing at the blueprint. His baton is nearby but not in use — tucked under his arm, resting on the bench, or sticking out of a vest pocket. He's in architect mode, not conductor mode.
- The Cellist is seated at the workbench, leaning forward to study the blueprint
- The Flutist stands at the workbench, one hand on the blueprint, asking a question
- The Guitarist stands behind the workbench, leaning on it with both hands, grinning

**Zone 2 — Not yet at the table:**
- The Accordionist sits in his chair a few steps from the workbench, arms crossed, head tilted — the skeptic
- The Violinist walks toward the workbench with violin under arm — the last to join

**Zone 3 — In his own world (off to one side):**
- The Drummer is setting up his percussion kit (cajón, cymbal on a stand, djembe), tapping a rhythm absentmindedly — not part of the huddle yet

**Construction hints (subtle, background):**
- A sawhorse, a few planks leaning against something, maybe a toolbox on the floor
- The stage itself is under renovation — enough to suggest "building something" without cluttering the scene

### The Blueprint

Unrolled on the workbench. Drawn in blueprint blue (`#1a5ea0`) with light blue fill (`#eef2f8`). Shows a theater floor plan — the thing they're going to build together. This is the only colored element on the workbench.

### Curtains

Red theater curtains (`#c42b2b` / `#d94040` highlights) are **CSS elements, not part of the illustration**. They animate independently.

**Shape:** Triangular — wider at the top (~12–15% of viewport width each), tapering inward as they go down, trailing off the bottom edge of the hero section. No visible bottom hem. This creates a natural viewport that widens as the visitor scrolls, allowing the curtains to gracefully exit without a hard cutoff between hero and content sections.

**Implementation:** CSS `clip-path` polygon shapes on absolutely positioned curtain elements, animated with `transform: translateX()`.

## Animation Choreography

A layered reveal sequence telling a micro-story: **waiting → leader arrives → ensemble activates → message lands**.

### Beat 0 — Initial State (0ms)
- Curtains cover the entire hero section, meeting at center
- Nothing visible behind them — just theater red filling the viewport
- Subtle fabric texture/fold gradient on the curtain surfaces

### Beat 1 — Curtains Part (0–1500ms)
- Curtains sweep outward from center to their final triangular resting position
- `transform: translateX()` with `cubic-bezier(0.25, 0.1, 0.25, 1)` easing
- Because of the triangular shape, the bottom-center of the scene peeks through first, then the reveal expands upward and outward
- The stage scene becomes progressively visible

### Beat 2 — Musicians Materialize (1200–2000ms, staggered fade)
- Musicians fade in at **20–25% opacity** — more subdued than their normal resting state
- Present but dormant — instruments held, not played
- The stage feels populated but uncoordinated
- 100ms stagger between each musician

### Beat 3 — Architect Arrives (1800–2300ms, fade + scale)
- The Architect fades in and scales from `0.95` to `1.0` — a subtle "stepping forward"
- `opacity: 0 → 1` + `transform: scale(0.95) → scale(1)`
- The gold baton catches a brief CSS shimmer/glow (200ms)
- This is the emotional peak — the moment it all clicks

### Beat 3b — Musicians Come Alive (2100–2500ms)
- Musicians' opacity shifts from 20–25% up to their full 30–35%
- Quick stagger, immediately following the architect's entrance
- The ensemble responds to the conductor's presence — coordination brings the pieces to life
- This beat reinforces the core message: agents are ready, but need orchestration

### Beat 4 — Headline Lands (2200–2500ms, fade-up)
- Text slides up ~20px and fades in: `opacity: 0 → 1` + `transform: translateY(20px) → translateY(0)`
- Subheadline follows 150ms after headline
- Appears right as the architect settles — message crystallizes with the visual

### Beat 5 — CTAs Appear (2400–2600ms, fade)
- Simple fade-in, no motion
- Total sequence: ~2.6 seconds from load to fully interactive

### Reduced-Motion Fallback
When `prefers-reduced-motion` is active:
- All elements render in their final state immediately — no animation
- Curtains are already in parted triangular position
- All figures at full target opacity
- Headline and CTAs visible instantly

### Technical Note
Beats 2–5 animate CSS overlay elements positioned over regions of the static AI-generated illustration. The illustration itself does not animate. Invisible divs aligned with the architect and musician regions control opacity transitions to create the reveal effect.

## Headline Copy & CTAs

### Headline
**"Every agent knows its part."**
- JetBrains Mono, 700 weight, Deep Navy `#1a3a6a`
- The metaphor works on two levels: musical part (the score) and role/responsibility
- Declarative, punchy — per brand voice spec
- Positioned in the upper-center text-safe zone of the illustration

### Subheadline
**"Spec-driven coordination for AI-assisted development."**
- Inter, 400 weight, dark gray `#333333`
- One line that explains what Synchestra actually is
- "Spec-driven" leads because it's the differentiator

### CTAs (side by side, centered below subheadline)

| Button | Style | Action |
|---|---|---|
| Star on GitHub | Primary — filled navy `#1a3a6a`, JetBrains Mono 500, GitHub star icon | Links to GitHub repo |
| See how it works | Secondary — outlined navy `#1a3a6a`, Inter 600 | Smooth-scrolls to first content section |

"Star on GitHub" rather than just "GitHub" gives a specific action and signals open-source immediately.

## Responsive Behavior

### Desktop (>1024px)
- Hero fills the viewport height (`100vh`)
- Illustration fills the lower 40% stage floor area (`object-fit: contain`, `object-position: center bottom`)
- Curtains take ~12–15% width each at top, tapering off-screen
- Headline centered in the upper white backdrop area

### Tablet (768–1024px)
- Hero remains full viewport
- Illustration may crop slightly at edges — curtains absorb this gracefully
- Font sizes scale down proportionally
- CTAs stack vertically if needed

### Mobile (<768px)
- Hero remains full viewport
- Mobile-specific illustration variant: architect + workbench + 2-3 closest musicians (cellist, flutist, guitarist). Drummer and outer characters cropped.
- Curtains become narrower or hidden on very small screens (<480px)
- Headline stacks to 2 lines naturally
- CTAs stack vertically, full-width

### Illustration Composition Constraint
The workbench and architect must live in the center 60% of the image. Edge-cropping on mobile must never lose the protagonist. The drummer (zone 3, off to one side) can be cropped on mobile.

## Illustration Style

The hero illustration follows the **Sempé-inspired pencil character style** defined in [spec/branding/](../README.md#character-illustrations-sempé-inspired-pencil-style), featuring the **orchestra cast** defined in [spec/branding/](../README.md#the-cast).

The hero scene shows all 6 musicians arranged across three spatial zones, with enough visual distinction that a returning visitor could start to recognize individuals.

## Asset Strategy

### AI-Generated Illustration
1. Generate the base scene using an image AI model, prompt-engineered for the Sempé-inspired pencil style
2. Iterate on prompts to match the warmth, line quality, and composition requirements
3. Post-process if needed to adjust stroke weights, colors, and opacity levels
4. If AI generation can't achieve the exact aesthetic, use a hybrid approach: generate composition/poses, then trace in a vector tool (Figma/Illustrator)

### Deliverables
- `hero-scene.webp` — full stage scene, landscape ~16:5 ratio, optimized for web (~200–400kb)
- `hero-scene-2x.webp` — retina version
- `hero-scene-mobile.webp` — tighter composition variant for mobile (portrait ~3:4)

### What's NOT in the Illustration
- **No curtains** — these are CSS elements
- **No text** — headline and CTAs are HTML
- **No explicit Synchestra branding** — the scene tells the story, the text names it
- **No stage floor color** — the beige comes from CSS, illustration has transparent/white background

## Outstanding Questions

None at this time.
