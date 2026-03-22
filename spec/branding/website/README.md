# Website Brand Application

This specification defines how Synchestra's brand identity (defined in [spec/branding/](../README.md)) is applied to the website — layout principles, hero section behavior, illustration placement, and page structure.

## General Principles

- Full light background (`#ffffff`), reasonable whitespace
- Dark/light theme switch planned for future (ship light first)
- Illustrations sit naturally on the white page — no frames or containers needed
- Feature/concept sections use smaller illustration vignettes from the same visual world
- Inline wireframe characters appear alongside text to introduce concepts

## Hero Section

Red curtains animate open on page load, revealing the architect and orchestra on the stage. The curtains remain at the edges of the hero section only — they do not persist through the rest of the site.

This is the signature brand moment: the theater metaphor made literal. The opening animation embodies "the show is about to begin" and directly connects to Synchestra's purpose — revealing structured coordination beneath apparent complexity.

**Reduced-motion fallback:** When `prefers-reduced-motion` is active, curtains render in the open position (parted at edges) with no animation. The hero content is immediately visible.

## Section Structure

Illustrations introduce each major concept section. The visual world is consistent throughout — always the same style, same cast, same stage — but each vignette highlights a different aspect of Synchestra's functionality.

### Copy Pattern

Each content section follows the voice system defined in [spec/branding/](../README.md#voice--tone):

1. **Section heading** — declarative, punchy (e.g., "Coordination, not chaos.")
2. **Problem/context paragraph** — technical but warm voice, relatable
3. **Solution/feature paragraph** — confident and declarative voice, authoritative
4. **Supporting illustration** — vignette from the recurring cast illustrating the concept

### Illustration Placement

| Section Type | Illustration Approach |
|---|---|
| Hero | Full stage scene — architect, musicians, curtains. Animated curtain opening. |
| Feature highlight | Medium vignette — 1–2 characters + relevant objects |
| How-it-works steps | Small inline figures — single character or object |
| CTA / closing | Callback to hero — architect in confident pose, stage visible |

## Outstanding Questions

None at this time.
