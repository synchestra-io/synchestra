# Hero Illustration Generation Plan

> **For agentic workers:** This plan involves human-in-the-loop AI image generation. Steps use checkbox (`- [ ]`) syntax for tracking. The human operator runs the AI image generation tool and evaluates results; the agentic worker assists with prompt crafting, post-processing, and integration.

**Goal:** Generate the Sempé-inspired pencil illustration for the hero scene — the orchestra gathered around a workbench studying blueprints — and integrate it into the landing page.

**Architecture:** Iterative prompt engineering → AI image generation → human evaluation → post-processing → integration into the static landing page. Multiple generation platforms may be tried (Midjourney, DALL-E 3, Stable Diffusion XL). The final asset replaces the current placeholder SVGs.

**Tech Stack:** AI image generation (platform TBD), image editing (background removal, color adjustment), WebP conversion, HTML/CSS integration

**Spec:** `spec/branding/website/hero-scene.md`

**Brand Spec:** `spec/branding/README.md` (see "The Cast" and "Character Illustrations: Sempé-Inspired Pencil Style")

---

## File Structure

```
apps/landing/src/
  index.html                    — Update <picture> element to reference final assets
  assets/
    hero-scene.svg              — Current placeholder (will be replaced)
    hero-scene-mobile.svg       — Current placeholder (will be replaced)
    hero-scene.webp             — Final desktop illustration (~16:5 ratio)
    hero-scene-2x.webp          — Final desktop illustration (retina)
    hero-scene-mobile.webp      — Final mobile illustration (~3:4 ratio)
```

---

### Task 1: Craft Desktop Prompt

Write the primary AI image generation prompt based on the scene spec and character roster.

**Reference files:**
- `spec/branding/website/hero-scene.md` — scene composition, zones, construction hints
- `spec/branding/README.md` — character roster (The Cast table), illustration style, color rules

- [ ] **Step 1: Write the base scene description**

Describe the scene in natural language covering:
- Workbench at center stage with blueprint unrolled on it
- Architect at near side, 3/4 angle, pointing at blueprint
- Baton nearby but not in use (tucked under arm, on bench, or in vest pocket)
- Slightly elevated audience POV, 5th row center

- [ ] **Step 2: Write each character description**

For each of the 7 characters, describe:
- Physical appearance (age, build, clothing driven by cultural vibe)
- Instrument and how they're holding/positioning it
- Body language and what it communicates
- Position in the scene (zone 1/2/3)

Characters at workbench (Zone 1): Architect, Cellist, Flutist, Guitarist
Away from table (Zone 2): Accordionist, Violinist
Own world (Zone 3): Drummer

- [ ] **Step 3: Write the environment description**

- Solid wood workbench (workbench, not a desk)
- Blueprint in blueprint blue (#1a5ea0) with light blue fill (#eef2f8)
- Subtle construction hints: sawhorse, planks leaning, toolbox
- White/transparent background (no stage floor color — CSS handles that)
- No curtains, no text, no audience

- [ ] **Step 4: Write the style description**

- Jean-Jacques Sempé editorial illustration style
- Warm graphite pencil (#404040) line work
- Selective color ONLY on: burgundy vest (#7a2a2a), gold baton (#c09018), blueprint blue (#1a5ea0)
- All musicians are pure graphite monochrome — no color
- Clean outlines, selective detail, expressive faces, lots of white space
- NOT photorealistic, NOT schematic wireframe

- [ ] **Step 5: Write the composition constraints**

- Wide landscape panoramic format (~16:5 or similar)
- Characters occupy center 60% of image width (edges will be covered by CSS curtains)
- Generous white space on left and right margins

- [ ] **Step 6: Assemble platform-specific prompts**

Create prompt variants for:
- **Midjourney v6**: Concise, comma-separated, with `--ar 16:5 --style raw --s 200` and `--no` parameters
- **DALL-E 3**: Natural language paragraphs with explicit negative instructions woven in
- **SDXL**: Token-weighted `(term:1.4)` syntax with separate negative prompt

Save all prompts to `apps/landing/src/assets/prompts/desktop-prompts.md` for reference.

- [ ] **Step 7: Commit**

```bash
git add apps/landing/src/assets/prompts/desktop-prompts.md
git commit -m "docs: add AI generation prompts for hero desktop illustration"
```

---

### Task 2: Generate Desktop Illustration

Human-driven task: run prompts through chosen AI platform, evaluate results, iterate.

- [ ] **Step 1: Choose platform and run first generation**

Start with whichever platform is most accessible. Run the desktop prompt.
Save raw outputs to `apps/landing/src/assets/prompts/generations/` for reference.

- [ ] **Step 2: Evaluate against checklist**

For each generated image, check:

| Criterion | Pass? |
|---|---|
| Sempé-like pencil style (warm, whimsical, not photorealistic) | |
| Architect is clearly the protagonist at center | |
| Workbench is a solid wood workbench (not a desk) | |
| Blueprint visible on workbench in blue tones | |
| At least 3-4 distinct characters visible around workbench | |
| Characters have varied ages, builds, postures | |
| Selective color: vest is burgundy-ish, baton is gold-ish | |
| Musicians are mostly monochrome graphite | |
| Construction hints present but subtle | |
| No curtains, no text, no audience in the image | |
| Characters in center 60% with space at edges | |
| Overall composition reads well at web scale | |

- [ ] **Step 3: Iterate on prompt**

Based on evaluation, adjust the prompt:
- If too much color: emphasize monochrome musicians, add to negative prompt
- If wrong style: try "New Yorker editorial cartoon" or "Sempé for Le Petit Nicolas"
- If wrong composition: add explicit spatial language
- If characters lack individuality: describe each more specifically
- If workbench missing/wrong: emphasize "solid woodworking workbench, not a desk"

Repeat Steps 1-3 until satisfied. Save the winning prompt and generation parameters.

- [ ] **Step 4: Select best result and note the winning prompt**

Document which platform, prompt version, and settings produced the best result.

---

### Task 3: Post-Process Desktop Image

Prepare the selected image for web integration.

- [ ] **Step 1: Remove or make background transparent**

The illustration needs a transparent or white background so the CSS beige stage floor shows through. Use an image editor or background removal tool.

- [ ] **Step 2: Adjust colors if needed**

Verify selective color elements match brand palette:
- Vest should be close to burgundy #7a2a2a
- Baton should be close to gold #c09018
- Blueprint should be close to blue #1a5ea0

Adjust in image editor if AI generation drifted from target colors.

- [ ] **Step 3: Crop to target aspect ratio**

Crop to approximately 16:5 landscape ratio. Ensure:
- Architect + workbench centered
- Edge margins have enough space for curtain overlap
- No important characters cut off at edges

- [ ] **Step 4: Export web-optimized versions**

```bash
# Standard (1920px wide)
cwebp -q 85 hero-scene-cropped.png -o apps/landing/src/assets/hero-scene.webp

# Retina (3840px wide or original resolution)
cwebp -q 85 hero-scene-cropped-2x.png -o apps/landing/src/assets/hero-scene-2x.webp
```

Target file sizes: ~200-400kb for standard, ~400-800kb for retina.

- [ ] **Step 5: Verify dimensions and file sizes**

```bash
file apps/landing/src/assets/hero-scene.webp
file apps/landing/src/assets/hero-scene-2x.webp
ls -la apps/landing/src/assets/hero-scene*.webp
```

- [ ] **Step 6: Commit**

```bash
git add apps/landing/src/assets/hero-scene.webp apps/landing/src/assets/hero-scene-2x.webp
git commit -m "feat(landing): add AI-generated hero illustration (desktop)"
```

---

### Task 4: Generate and Process Mobile Illustration

Create the tighter mobile variant.

- [ ] **Step 1: Adapt the desktop prompt for mobile**

Tighter composition focusing on:
- Architect + workbench + 2-3 closest musicians (cellist, flutist, guitarist)
- Drop the drummer, accordionist, and violinist
- Portrait-ish ratio (~3:4)
- Same style and color rules

- [ ] **Step 2: Generate mobile illustration**

Run the mobile prompt. Evaluate against the same checklist but with mobile-specific criteria:
- Architect is prominent and centered
- Workbench and blueprint visible
- 2-3 musicians clearly distinct
- Reads well at mobile viewport widths (~375-430px)

- [ ] **Step 3: Post-process mobile image**

Same steps as Task 3: background removal, color adjustment, crop to ~3:4 ratio.

- [ ] **Step 4: Export web-optimized mobile version**

```bash
# Standard
cwebp -q 85 hero-scene-mobile-cropped.png -o apps/landing/src/assets/hero-scene-mobile.webp

# Retina (if source resolution allows)
cwebp -q 85 hero-scene-mobile-cropped-2x.png -o apps/landing/src/assets/hero-scene-mobile-2x.webp
```

- [ ] **Step 5: Commit**

```bash
git add apps/landing/src/assets/hero-scene-mobile.webp
git commit -m "feat(landing): add AI-generated hero illustration (mobile)"
```

---

### Task 5: Integrate Illustrations into Landing Page

Update the HTML to use the final WebP assets instead of placeholder SVGs.

**Files:**
- Modify: `apps/landing/src/index.html`

- [ ] **Step 1: Update the `<picture>` element**

Replace the current SVG references with WebP assets and add retina support:

```html
<picture>
  <source media="(max-width: 767px)"
          srcset="assets/hero-scene-mobile.webp"
          type="image/webp">
  <source srcset="assets/hero-scene-2x.webp 2x,
                  assets/hero-scene.webp 1x"
          type="image/webp">
  <img src="assets/hero-scene.webp"
       alt="An amateur orchestra gathered around a workbench on a theater stage — the architect in a burgundy vest points at blueprints while musicians with diverse instruments look on, some eager, some skeptical, one setting up drums off to the side"
       width="1920" height="600"
       loading="eager"
       fetchpriority="high">
</picture>
```

- [ ] **Step 2: Verify the illustration displays correctly**

```bash
cd apps/landing/src && python3 -m http.server 8080
```

Check at http://localhost:8080:
- Desktop: illustration fills the beige stage floor area, characters visible through curtain opening
- Mobile: tighter composition shows architect + nearby musicians
- Characters sit on beige stage floor (no white background visible between illustration and CSS floor)
- Dark edge line aligns with top of illustration area
- Curtain animation still works correctly over the illustration

- [ ] **Step 3: Remove placeholder SVG files**

```bash
rm apps/landing/src/assets/hero-scene.svg
rm apps/landing/src/assets/hero-scene-mobile.svg
```

- [ ] **Step 4: Commit**

```bash
git add apps/landing/src/index.html
git rm apps/landing/src/assets/hero-scene.svg apps/landing/src/assets/hero-scene-mobile.svg
git commit -m "feat(landing): integrate final hero illustrations, remove placeholders"
```

---

### Task 6: Visual QA and Final Adjustments

- [ ] **Step 1: Cross-browser check**

Test in Chrome, Safari, and Firefox:
- Illustration renders correctly
- Curtain animation works over illustration
- Text is readable against white backdrop
- Stage floor/illustration boundary looks clean
- No layout shift on load

- [ ] **Step 2: Responsive check**

Test at these viewpoints:
- Desktop wide (1920px)
- Desktop standard (1440px)
- Tablet (768px)
- Mobile (375px)

Verify illustration scales cleanly and mobile variant switches at 767px breakpoint.

- [ ] **Step 3: Performance check**

```bash
ls -la apps/landing/src/assets/hero-scene*.webp
```

Verify:
- Total hero assets < 1MB combined
- No layout shift (width/height attributes set correctly on `<img>`)
- `loading="eager"` and `fetchpriority="high"` on hero image

- [ ] **Step 4: Final adjustments and commit**

Fix any issues found. Commit with:

```bash
git add apps/landing/src/
git commit -m "fix(landing): hero illustration visual QA adjustments"
```
