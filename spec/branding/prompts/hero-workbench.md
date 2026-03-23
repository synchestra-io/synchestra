# Hero Illustration — Workbench Scene Prompts

Reference prompts for AI-generating the Synchestra hero scene illustration.
Scene spec: `spec/branding/website/hero-scene.md` | Brand spec: `spec/branding/README.md`

---

## 1. Base Scene Description

A group of eight diverse musicians standing around a solid woodworking workbench on a theater stage. Their leader stands IN FRONT of the workbench with his BACK TO THE VIEWER, looking to the RIGHT toward the group. His left hand rests on the workbench edge behind him. His right hand holds a gold baton, pointing it DOWN at a white paper with blue ink lines (blueprint) on the workbench, rolled a bit on the right side. The baton tip is touching the blueprint. Only ONE baton in the entire scene — the one in his right hand. The others stand on the far side and ends of the workbench, facing him.

The viewpoint is slightly elevated, as if from the 5th row of a theater.

The stage has subtle construction hints — a sawhorse, planks leaning in the background — but the focus is on the characters.

---

## 2. Character Descriptions

### The Architect (in front of workbench, back to viewer, looking right)
Clean-shaven man in his early 40s. Lean, energetic, confident — a modern tech leader with 15 years of experience, not an academic or professor. Wearing a burgundy vest over a modern henley shirt with rolled-up sleeves. Stands IN FRONT of the workbench with his BACK TO THE VIEWER, looking to the RIGHT toward the group. Left hand rests on the workbench edge behind him. Right hand holds a gold baton, pointing it DOWN at the blueprint on the workbench, baton tip touching the blueprint. Only ONE baton in the scene. Stands a comfortable distance from the front edge of the stage — not near the edge, with plenty of floor space in front of him. Authority through competence, not volume.

### 1. The Cellist (far left corner of workbench)
Man, 60s. Large, bearish build. Thick-rimmed glasses, wool cardigan. Standing at the FAR LEFT CORNER of the workbench, leaning over to study the blueprint with intense focus. Cello resting nearby. The scholar.

### 2. The Flutist (behind table, center — ONLY on far side, NOT in front)
Woman, early 40s. Slender, poised. Light scarf, modern fitted jacket. Standing BEHIND the table (far side, center) with one hand on the blueprint and silver flute held in the other hand. Engaged with the plans, looks like she's about to ask a pointed question. The questioner.

### 3. The Guitarist (far right corner of workbench)
Man, early 30s. Medium build, relaxed. Open-collar shirt, sleeves rolled up. Classical guitar slung across his back. Standing at the FAR RIGHT CORNER of the workbench, leaning on it with both hands, grinning. The enthusiast.

### 4. The Accordionist (front right corner of workbench)
Man, 50s. Broad, heavyset build with a full beard. Wearing a flat cap, heavy utility jacket with chest pockets over a turtleneck sweater, baggy jeans, and sturdy shoes. Standing at the FRONT RIGHT CORNER of the workbench (audience side), touching it with his right hand. Skeptical "convince me" expression. Accordion next to him under the table. The skeptic.

### 5–6. The Violinist and The Librarian (walking together from the left)
Two people walking together from the LEFT toward the workbench — mid-stride, clearly in motion.

**The Violinist:** Woman, mid-20s. Slight, composed. Clean minimalist modern clothing. Walking mid-step, one foot in front of the other, violin tucked under her arm (NOT playing it), looking curiously toward the blueprint. The observer — last to join.

**The Librarian:** Young man carrying a tall stack of sheet music and folders, walking beside the violinist. The organized one, the librarian of the group.

### 7. The Drummer (off to one side, setting up)
Man, late 30s. Athletic, easy presence. Simple t-shirt, comfortable in his skin. Off to the right side but not too far from the group, setting up a cajón and percussion kit, facing toward the center of the stage, not paying attention to the blueprint huddle. The black sheep.

---

## 3. Environment Description

- **Workbench:** Solid woodworking workbench made of heavy timber. Sturdy, practical — clearly a workbench, not a desk or table. Center stage.
- **Blueprint:** Large white paper with blue ink lines (#1a5ea0) unrolled on the workbench, rolled a bit on the right side.
- **Baton:** Only ONE gold baton in the scene — held in the architect's right hand, pointing at the blueprint. NOT resting on the table.
- **Construction hints (subtle):** A sawhorse behind the workbench, planks leaning against something off to the left.
- **Stage floor:** Straight wooden floorboards running left-to-right. Boards are rectangular planks (roughly 180mm wide by 30mm thick, about 6:1 ratio) — NOT square. The front edge of the stage is a perfectly straight horizontal line showing the cut ends of the boards. A darker/heavier pencil line along the very bottom edge of the stage for weight and definition. The floor extends beyond both left and right edges of the image — no visible side edges. Floorboards must continue at full detail and opacity all the way to the very left and right edges — NO fading, NO vignette, NO softening at the edges. Boards are simply cut off by the image boundary.
- **Bottom crop:** Crop tightly at the bottom — right below the bottom border of the stage edge. Show the stage front edge (side/thickness of the boards) but absolutely nothing below it. The very bottom pixel row should be the bottom of the stage.
- **Background above stage:** Solid pure white (#FFFFFF), no curtains (CSS handles those), no text, no audience. IMPORTANT: Do NOT use a transparent or checkerboard background — AI generators render "transparent" as a visible checkerboard pattern baked into the pixels. Always request a solid white background.

---

## 4. Style Description

Modern editorial pencil illustration — warm, clean graphite line drawing. Contemporary people in modern casual clothing (NOT old-fashioned, NOT Victorian, NOT 19th century). Every character feels alive through expressive details.

- **Line work:** Warm graphite pencil, clean outlines, confident strokes
- **Detail level:** Expressive faces, distinct clothing, lots of white space
- **Color:** HIGHLY SELECTIVE — only three colored elements:
  1. The architect's vest — burgundy
  2. The baton in his hand — gold
  3. The blueprint — blue
- **All musicians are pure graphite monochrome** — no color on any person, clothing, or instrument
- **NOT:** photorealistic, 3D, digital painting, anime, old-fashioned, Victorian, academic

---

## 5. Composition Constraints

- **Format:** Wide panoramic landscape, approximately 16:5 aspect ratio (3200x1000px)
- **Character placement:** All eight characters must be within the CENTER 90% of the image width. Leave empty stage floor space on both left and right edges (at least 5% on each side) — these margins will be covered by curtain overlays. No character, instrument, or prop should be in the outer 5% on either side.
- **Character arrangement:** Leader stands IN FRONT of workbench facing RIGHT toward the group. Others on far side and ends. NOT a Last Supper composition (not everyone behind the table).
- **No duplicates:** Each of the eight characters is a DIFFERENT person with a DIFFERENT instrument or role. No character or instrument should appear twice. Each instrument appears EXACTLY ONCE in the entire illustration — do not draw partial, extra, or background copies of any instrument. The flutist appears ONLY on the far side of the workbench, NOT in front of it. The conductor/architect stands ONLY in front of the workbench (audience side), NOT behind it. Every musician must have their instrument visibly with them — held, worn, or placed right next to them.
- **All characters standing** — no one seated
- **Stage floor:** Straight floorboards, straight front edge, extends off-screen left and right. Full detail to edges — no fading.
- **Bottom crop:** Tight crop at bottom of stage edge.

---

## 6. Platform-Specific Prompts

### Gemini (recommended — best results)

```
Draw a modern editorial pencil illustration of a whimsical orchestra rehearsal scene. Wide panoramic format, 16:5 aspect ratio (3200x1000 pixels).

A diverse group of eight people stand around a heavy wooden workbench on a theater stage. Their leader — a clean-shaven, energetic man in his early 40s wearing a burgundy vest over a modern henley shirt with rolled-up sleeves — stands IN FRONT of the workbench with his BACK TO THE VIEWER, looking to the RIGHT toward the group. His left hand rests on the workbench edge behind him. His right hand holds a gold baton, pointing it DOWN at a large white paper with blue ink lines (a blueprint) spread on the workbench, rolled a bit on the right side. The baton tip is touching the blueprint. Only ONE baton in the entire scene — the one in his right hand. He looks like an experienced, confident tech leader — clearly early 40s with subtle maturity in his face, not a college student. He stands a comfortable distance from the front edge of the stage — not near the edge, with plenty of floor space in front of him.

The positions around the workbench, from LEFT to RIGHT:
1. FAR LEFT CORNER of table: A large, bearish older man with thick-rimmed glasses and a wool cardigan, leaning over the blueprint studying it intently. His cello rests nearby.
2. IN FRONT of the table (audience side), BACK TO THE VIEWER: The leader/conductor described above.
3. BEHIND the table (far side, center): A slender woman with a light scarf, one hand on the blueprint and a silver flute in the other hand, looking like she's about to ask a question.
4. FAR RIGHT CORNER of table: A relaxed young man with a guitar slung on his back, leaning on the table with both hands, grinning.
5. FRONT RIGHT CORNER of table: A broad, heavyset man with a full beard, wearing a flat cap and heavy utility jacket with chest pockets over a turtleneck sweater, touching the table with his right hand, skeptical expression. His accordion is next to him under the table.

Not at the table:

Two people walking together from the LEFT toward the workbench — they are mid-stride, clearly in motion, approaching the group:
- A slight young woman with a violin tucked under her arm (not playing it), looking curiously toward the blueprint. She is walking, mid-step, one foot in front of the other.
- A young man carrying a tall stack of sheet music and folders, walking beside her — the organized one, the librarian of the group.

Off to the right:
- An athletic man in a t-shirt off to the right side but not too far from the group, setting up a cajón and percussion kit, facing toward the center of the stage, not paying attention to the group

IMPORTANT: All eight characters must be within the CENTER 86% of the image width. Leave empty stage floor space on both the left and right edges (at least 7% on each side) — these margins will be covered by curtain overlays. No character, instrument, or prop should be in the outer 7% on either side.

A sawhorse and some planks lean in the background as subtle construction hints.

The stage floor is made of straight wooden floorboards running left to right. The front edge of the stage is a perfectly straight horizontal line — NOT curved — showing the cut ends of the boards. The boards are rectangular planks (roughly 180mm wide by 30mm thick, about 6:1 ratio) — NOT square. The visible front edge should show these thin rectangular board ends side by side, with a darker/heavier pencil line along the very bottom edge of the stage to give it weight and definition. The stage is LARGER than the image — it extends BEYOND the viewport on both left and right sides. The image crops into the middle of the stage. There must be NO visible side edges or corners of the stage — the floorboards are simply cut off by the image boundary at full detail and opacity. NO fading, NO vignette, NO softening, NO empty space at the edges. The stage floor fills the entire bottom of the image from the very left pixel to the very right pixel.

IMPORTANT: Crop the image tightly at the bottom — crop right below the bottom border of the stage edge. Show the stage front edge (the side/thickness of the boards) but absolutely nothing below it. The very bottom pixel row of the image should be the bottom of the stage.

IMPORTANT: Each of the eight characters is a DIFFERENT person with a DIFFERENT instrument or role. No character or instrument should appear twice. Each instrument appears EXACTLY ONCE in the entire illustration — do not draw partial, extra, or background copies of any instrument. The flutist appears ONLY on the far side of the workbench, NOT in front of it. The conductor/architect stands ONLY in front of the workbench (audience side), NOT behind it. Every musician must have their instrument visibly with them — held, worn, or placed right next to them.

Style: warm graphite pencil line drawing, clean outlines, expressive faces, lots of white space. Contemporary modern casual clothing — not old-fashioned, not Victorian. The entire drawing is monochrome graphite EXCEPT for three elements which have color: the leader's burgundy vest, the gold baton, and the blue blueprint. Everything else is black and white pencil sketch. No curtains, no text, no audience. Solid pure white background above the stage — NOT transparent, NOT a checkerboard pattern. Solid white (#FFFFFF) background everywhere that is not the stage or characters.
```

---

## Quick Evaluation Checklist

| # | Criterion | Notes |
|---|---|---|
| 1 | Modern pencil illustration style? | Warm, clean, NOT photorealistic or old-fashioned |
| 2 | Architect in front of table, back to viewer, looking right? | Back to viewer, facing right toward group, NOT behind the table |
| 3 | Architect looks early 40s, modern, clean-shaven? | NOT old, bearded, or academic |
| 4 | Baton in architect's RIGHT hand only? | Only ONE baton, held not resting on table |
| 5 | All contemporary clothing? | NOT Victorian or 19th century |
| 6 | All 8 characters present and distinct? | Each has different instrument/role, no duplicates |
| 7 | Everyone standing? | No one seated |
| 8 | Workbench is solid wood? | NOT a desk, table, or podium |
| 9 | Blueprint is white paper with blue ink, rolled on right? | NOT solid blue sheet |
| 10 | Musicians are monochrome graphite? | No color on people/instruments |
| 11 | Characters within center 70%? | 15% empty margins on each side for curtains |
| 12 | Characters around table, not all behind it? | NOT a Last Supper composition |
| 13 | Stage LARGER than image, extends beyond viewport? | NO visible corners/edges of stage, floor fills entire bottom edge-to-edge |
| 14 | Board ends rectangular (6:1), darker bottom line? | NOT square ends |
| 15 | Tight bottom crop at stage edge? | Nothing below the stage |
| 16 | Violinist and librarian walking together from left? | Mid-stride, in motion, not standing still |
| 17 | Skeptic at front right corner of workbench? | Touching workbench with right hand, accordion next to him |
| 18 | Librarian is a young man with sheet music stack? | Walking beside violinist from left |
| 19 | Drummer off to side, facing center? | Not too far from group |
| 20 | No curtains, no text, no audience? | CSS handles curtains |
| 21 | Solid white background above stage? | NO checkerboard/transparency pattern |
| 22 | Each instrument appears exactly once? | No partial, extra, or background duplicates |

---

## Iteration Log

### Final working version (Gemini)
- **Platform:** Gemini (model name TBD)
- **Iterations:** ~6 rounds of refinement
- **Key learnings:**
  - "Sempé" and "New Yorker" references pull Victorian/19th century aesthetics — use "modern editorial pencil illustration" instead
  - Must explicitly say "NOT old-fashioned, NOT Victorian" and "contemporary modern casual clothing"
  - Must explicitly say architect is "early 40s, clean-shaven, mature" — AI defaults to old bearded professor
  - Must say "IN FRONT of the workbench facing the group" — otherwise all characters end up behind the table (Last Supper)
  - Must say "standing" for all characters — AI defaults to seating some
  - Must say stage floor is "straight, NOT curved" and "extends off-screen left and right"
  - Must say violinist is "walking, violin under arm, NOT playing"
  - Midjourney moderation blocks prompts with specific demographic descriptions of realistic people — use "illustrated characters" and avoid ethnicity/age combos
  - AI generators render "transparent background" as a visible checkerboard pattern baked into pixels — always explicitly request "solid pure white background", never "transparent"
  - Must specify baton is in architect's hand, not on the table — otherwise AI generates multiple batons
  - Must specify architect faces RIGHT — generic "facing the group" can produce ambiguous poses
  - Added 8th character (librarian with sheet music) for richer scene
  - Must specify center 70% placement with 15% margins — otherwise characters spill into curtain overlay zones
  - Must specify board dimensions (6:1 ratio) and no edge fading — AI defaults to vignetting at edges
  - Must specify tight bottom crop — AI adds empty space or ground below stage
  - Must explicitly say each instrument appears EXACTLY ONCE — AI duplicates instruments (e.g. rendering a cello 1.5 times)
  - Must say stage is LARGER than the image and extends BEYOND the viewport — "extends off-screen" is too weak, AI still draws stage corners/edges visible in frame

---

## Asset Export

### Desktop (full panoramic)
- **Source:** `spec/branding/illustrations/hero-workbench3.png` (3712x1152)
- **2x:** `hero-scene-2x.webp` — full resolution, cwebp -q 85
- **1x:** `hero-scene.webp` — cwebp -q 85 -resize 1856 576

### Mobile (center crop, shifted right)
- **Breakpoint:** `max-width: 800px`
- **Goal:** Show all workbench characters fully (conductor, cellist, flutist, guitarist, accordionist) — outside characters (walking pair on left, drummer on right) and their instruments out of view
- **Crop:** 1870x1152 from the 3712px-wide source
- **Offset:** `--cropOffset 0 1050` (shifted right of center: keeps accordionist's hands in frame on the right, cuts walking pair and sawhorse on the left)
- **Command:** `sips --cropOffset 0 1050 --cropToHeightWidth 1152 1870 <source.png> --out <crop.png>`
- **Export:** `hero-scene-mobile.webp` — cwebp -q 85 -resize 1000 576
- **Tuning notes:** 1800px crops accordionist's hand; 1950px brings drums into view; 1870px is the sweet spot
