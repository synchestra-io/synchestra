# Hero Scene Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the landing page hero section as an immersive theater scene with animated red curtains, layered character reveals, and brand-aligned typography/colors — replacing the current dark-themed, text-only hero.

**Architecture:** The landing page remains a single static HTML file (`apps/landing/src/index.html`) with Tailwind CSS (CDN). The hero section uses CSS animations (keyframes + animation-delay) for the layered reveal sequence. Curtains are CSS `clip-path` polygon elements. The illustration is a single `<img>` tag referencing a pre-generated asset. No JavaScript is required for the animation — CSS handles all timing, easing, and reduced-motion fallback via `@media (prefers-reduced-motion)`.

**Tech Stack:** HTML, Tailwind CSS (CDN), CSS custom properties, CSS keyframes, Google Fonts (JetBrains Mono + Inter)

**Spec:** `spec/branding/website/hero-scene.md`

**Brand Spec:** `spec/branding/README.md`

---

## File Structure

```
apps/landing/src/
  index.html              — Single landing page file (modify hero section + global styles)
  assets/
    hero-scene.webp        — Full stage scene illustration (placeholder until AI-generated)
    hero-scene-2x.webp     — Retina version (placeholder)
    hero-scene-mobile.webp — Center-cropped mobile variant (placeholder)
```

The landing page is a single HTML file with inline Tailwind. All hero-specific CSS (keyframes, custom properties, clip-paths) will be added in a `<style>` block in the `<head>`. This avoids introducing a build system for what is a static page.

---

## Task 1: Set Up Brand Foundations (Fonts + Colors + Tailwind Config)

**Files:**
- Modify: `apps/landing/src/index.html` — `<head>` section and Tailwind config

This task updates the global page foundation: fonts, color palette, and background from dark to light. No hero changes yet — just the brand system.

- [ ] **Step 1: Add Google Fonts import for JetBrains Mono and Inter**

Add to `<head>`, before the Tailwind script:

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
```

- [ ] **Step 2: Update Tailwind config with brand colors and fonts**

Replace the existing `tailwind.config` script block:

```html
<script>
  tailwind.config = {
    theme: {
      extend: {
        colors: {
          navy: { DEFAULT: '#1a3a6a', light: '#2a5a9a' },
          curtain: { DEFAULT: '#c42b2b', light: '#d94040' },
          gold: '#c09018',
          burgundy: '#7a2a2a',
          blueprint: { DEFAULT: '#1a5ea0', light: '#eef2f8' },
        },
        fontFamily: {
          mono: ['"JetBrains Mono"', 'monospace'],
          sans: ['Inter', 'system-ui', 'sans-serif'],
        },
      }
    }
  }
</script>
```

- [ ] **Step 3: Switch body from dark to light theme**

Change the `<body>` tag:

```html
<body class="bg-white text-gray-800 font-sans antialiased">
```

- [ ] **Step 4: Update nav to light theme**

Replace the `<header>` with:

```html
<header class="fixed top-0 inset-x-0 z-50 border-b border-gray-200 bg-white/80 backdrop-blur">
  <div class="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
    <span class="text-lg font-mono font-semibold tracking-tight text-navy">Synchestra</span>
    <a href="/app/" class="px-4 py-2 rounded-lg bg-navy hover:bg-navy-light text-sm font-medium text-white transition">
      Sign in
    </a>
  </div>
</header>
```

- [ ] **Step 5: Verify the page renders with light theme, correct fonts, updated nav**

Open `apps/landing/src/index.html` in a browser. Check:
- Background is white
- Nav text uses JetBrains Mono for "Synchestra"
- Body text uses Inter
- Navy color (`#1a3a6a`) is used for the nav brand name
- Sign-in button is navy, not indigo

- [ ] **Step 6: Commit**

```bash
git add apps/landing/src/index.html
git commit -m "feat(landing): switch to brand color system, fonts, and light theme"
```

---

## Task 2: Create Placeholder Hero Illustration Assets

**Files:**
- Create: `apps/landing/src/assets/hero-scene.webp`
- Create: `apps/landing/src/assets/hero-scene-mobile.webp`

Placeholder images that establish the correct dimensions and composition zones. These will be replaced with the AI-generated Sempé-style illustration later. Using simple gradient/shape placeholders so the CSS layout and animation can be developed and tested independently of the final art.

- [ ] **Step 1: Create the assets directory**

```bash
mkdir -p apps/landing/src/assets
```

- [ ] **Step 2: Generate placeholder images**

Create simple placeholder images using an HTML canvas or download tool. The placeholders should be:
- `hero-scene.webp`: 1920x1080 (16:9), light gray with a subtle grid pattern and text "HERO SCENE — STAGE ILLUSTRATION" centered. The upper third should be lighter (text-safe zone). A dark circle in the center-bottom third indicates where the architect will stand.
- `hero-scene-mobile.webp`: 800x900 (~4:5 crop), center-cropped version of the same concept.

If tooling is unavailable, create minimal SVG-based placeholders inline and skip this step — Task 3 can use a CSS gradient background as fallback.

- [ ] **Step 3: Commit**

```bash
git add apps/landing/src/assets/
git commit -m "feat(landing): add placeholder hero illustration assets"
```

---

## Task 3: Build Hero Section HTML Structure

**Files:**
- Modify: `apps/landing/src/index.html` — replace the `<!-- Hero -->` section

This task builds the static HTML structure for the immersive hero: illustration background, curtain elements, text overlay, and CTAs. No animation yet — everything is in its final resting state.

- [ ] **Step 1: Add hero CSS custom properties and base styles**

Add a `<style>` block in `<head>`, after the Tailwind script:

```html
<style>
  /* Hero scene custom styles */
  .hero-scene {
    position: relative;
    height: 100vh;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  /* Illustration background */
  .hero-illustration {
    position: absolute;
    inset: 0;
    z-index: 0;
  }
  .hero-illustration img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    object-position: center 40%;
  }

  /* Curtain elements */
  .curtain {
    position: absolute;
    top: 0;
    bottom: 0;
    z-index: 10;
    background: linear-gradient(180deg, #c42b2b 0%, #d94040 30%, #c42b2b 70%, #b02525 100%);
  }
  .curtain-left {
    left: 0;
    width: 52%;
    clip-path: polygon(0 0, 100% 0, 30% 100%, 0 100%);
  }
  .curtain-right {
    right: 0;
    width: 52%;
    clip-path: polygon(0 0, 100% 0, 100% 100%, 70% 100%);
  }

  /* Text overlay */
  .hero-content {
    position: relative;
    z-index: 20;
    text-align: center;
    padding: 0 2rem;
    margin-top: -10vh;
  }

  /* Responsive */
  @media (max-width: 767px) {
    .hero-scene {
      height: 75vh;
    }
    .curtain-left,
    .curtain-right {
      clip-path: polygon(0 0, 100% 0, 20% 100%, 0 100%);
    }
    .curtain-right {
      clip-path: polygon(0 0, 100% 0, 100% 100%, 80% 100%);
    }
  }
  @media (max-width: 479px) {
    .curtain {
      display: none;
    }
  }
</style>
```

- [ ] **Step 2: Replace the hero section HTML**

Replace the `<!-- Hero -->` section with:

```html
<!-- Hero -->
<section class="hero-scene">
  <!-- Illustration background -->
  <div class="hero-illustration">
    <picture>
      <source media="(max-width: 767px)" srcset="assets/hero-scene-mobile.webp">
      <img src="assets/hero-scene.webp"
           srcset="assets/hero-scene.webp 1x, assets/hero-scene-2x.webp 2x"
           alt="An amateur orchestra on a theater stage — the conductor stands center stage with baton raised, musicians arranged in a semicircle behind, each a distinct character with their own instrument and personality"
           width="1920" height="1080"
           loading="eager"
           fetchpriority="high">
    </picture>
  </div>

  <!-- Curtains (CSS-only, animated separately) -->
  <div class="curtain curtain-left"></div>
  <div class="curtain curtain-right"></div>

  <!-- Text overlay -->
  <div class="hero-content">
    <h1 class="font-mono font-bold text-4xl md:text-6xl lg:text-7xl text-navy leading-tight tracking-tight">
      Every agent knows its part.
    </h1>
    <p class="mt-4 text-lg md:text-xl text-gray-600 max-w-2xl mx-auto">
      Spec-driven coordination for AI-assisted development.
    </p>
    <div class="mt-8 flex flex-col sm:flex-row gap-4 justify-center">
      <a href="https://github.com/synchestra-io/synchestra"
         class="inline-flex items-center gap-2 px-6 py-3 rounded-lg bg-navy hover:bg-navy-light text-white font-mono font-medium text-base transition">
        <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 .3a12 12 0 0 0-3.8 23.38c.6.12.82-.26.82-.58l-.02-2.05c-3.34.73-4.04-1.61-4.04-1.61-.55-1.39-1.34-1.76-1.34-1.76-1.08-.74.08-.73.08-.73 1.2.09 1.84 1.24 1.84 1.24 1.07 1.83 2.8 1.3 3.49 1 .1-.78.42-1.3.76-1.6-2.67-.31-5.47-1.34-5.47-5.93 0-1.31.47-2.38 1.24-3.22-.14-.3-.54-1.52.1-3.18 0 0 1-.32 3.3 1.23a11.5 11.5 0 0 1 6.02 0c2.28-1.55 3.29-1.23 3.29-1.23.64 1.66.24 2.88.12 3.18a4.65 4.65 0 0 1 1.23 3.22c0 4.61-2.8 5.62-5.48 5.92.42.36.81 1.1.81 2.22l-.01 3.29c0 .32.21.7.82.58A12 12 0 0 0 12 .3"/></svg>
        Star on GitHub
      </a>
      <a href="#why-synchestra"
         class="inline-flex items-center gap-2 px-6 py-3 rounded-lg border border-navy text-navy font-semibold text-base hover:bg-navy hover:text-white transition">
        See how it works
        <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"/></svg>
      </a>
    </div>
  </div>
</section>
```

- [ ] **Step 3: Add id="why-synchestra" to the Why section for smooth scroll**

On the `<!-- Why Synchestra -->` section tag, add the id:

```html
<section id="why-synchestra" class="py-24 px-6 border-t border-gray-200">
```

Also update all remaining section `border-gray-800` classes to `border-gray-200` for the light theme.

- [ ] **Step 4: Verify the static hero renders correctly**

Open in browser. Check:
- Hero fills viewport height
- Curtains are visible at left and right edges, triangular shape (wider at top)
- Headline "Every agent knows its part." in JetBrains Mono, navy color
- Subheadline in Inter
- Two CTAs: "Star on GitHub" (filled navy with GitHub icon) and "See how it works" (outlined)
- "See how it works" scrolls to the Why section
- On mobile viewport: hero is 75vh, curtains narrower or hidden at <480px

- [ ] **Step 5: Commit**

```bash
git add apps/landing/src/index.html
git commit -m "feat(landing): build immersive hero section with curtains and brand copy"
```

---

## Task 4: Add Animation Keyframes and Layered Reveal Sequence

**Files:**
- Modify: `apps/landing/src/index.html` — add keyframes to `<style>`, add animation classes to hero elements

This task adds the full animation choreography: curtain reveal, musician/architect overlays, headline fade-up, and CTA fade-in.

- [ ] **Step 1: Add CSS keyframes to the style block**

Add these keyframes inside the existing `<style>` block:

```css
/* Animation keyframes */
/* Curtains start at 52% width covering the viewport center, then slide
   outward so only the triangular edge remains visible (~15% effective). */
@keyframes curtain-open-left {
  from { transform: translateX(0); }
  to   { transform: translateX(-72%); }
}
@keyframes curtain-open-right {
  from { transform: translateX(0); }
  to   { transform: translateX(72%); }
}
@keyframes fade-in {
  from { opacity: 0; }
  to   { opacity: 1; }
}
@keyframes fade-up {
  from { opacity: 0; transform: translateY(20px); }
  to   { opacity: 1; transform: translateY(0); }
}
@keyframes scale-in {
  from { opacity: 0; transform: scale(0.95); }
  to   { opacity: 1; transform: scale(1); }
}
@keyframes musician-activate {
  from { opacity: 0.22; }
  to   { opacity: 0.35; }
}
@keyframes baton-shimmer {
  0%, 100% { filter: brightness(1); }
  50%      { filter: brightness(1.6) drop-shadow(0 0 8px #c09018); }
}
```

- [ ] **Step 2: Add animation styles for each beat**

Add these styles below the keyframes:

```css
/* Beat 1: Curtains part (0-1500ms) */
.curtain-left {
  animation: curtain-open-left 1.5s cubic-bezier(0.25, 0.1, 0.25, 1) forwards;
}
.curtain-right {
  animation: curtain-open-right 1.5s cubic-bezier(0.25, 0.1, 0.25, 1) forwards;
}

/* Beat 2: Musicians materialize (1200-2000ms) — overlay regions */
.hero-musicians {
  position: absolute;
  inset: 0;
  z-index: 5;
  background: white;
  opacity: 0.78;
  animation: fade-in 0.8s ease-out 1.2s forwards;
  animation-fill-mode: backwards;
}
/* Musicians start hidden behind white overlay, which fades to reveal them at low opacity */

/* Beat 3: Architect arrives (1800-2300ms) — overlay region */
.hero-architect {
  position: absolute;
  inset: 0;
  z-index: 6;
  background: white;
  opacity: 0;
  animation: fade-in 0.5s ease-out 1.8s forwards;
}

/* Beat 3b: Musicians come alive (2100-2500ms) */
/* Handled by a second opacity transition on the musician overlay */

/* Beat 4: Headline lands (2200-2500ms) */
.hero-headline {
  opacity: 0;
  animation: fade-up 0.4s ease-out 2.2s forwards;
}
.hero-subheadline {
  opacity: 0;
  animation: fade-up 0.4s ease-out 2.35s forwards;
}

/* Beat 5: CTAs appear (2400-2600ms) */
.hero-ctas {
  opacity: 0;
  animation: fade-in 0.3s ease-out 2.4s forwards;
}

/* Reduced-motion fallback */
@media (prefers-reduced-motion: reduce) {
  .curtain-left {
    animation: none;
    transform: translateX(-72%);
  }
  .curtain-right {
    animation: none;
    transform: translateX(72%);
  }
  .hero-musicians,
  .hero-architect {
    animation: none;
    opacity: 0;
  }
  .hero-headline,
  .hero-subheadline,
  .hero-ctas {
    animation: none;
    opacity: 1;
    transform: none;
  }
}
```

- [ ] **Step 3: Add animation classes to the hero HTML elements**

Update the hero content div's children with animation classes:

```html
<div class="hero-content">
  <h1 class="hero-headline font-mono font-bold text-4xl md:text-6xl lg:text-7xl text-navy leading-tight tracking-tight">
    Every agent knows its part.
  </h1>
  <p class="hero-subheadline mt-4 text-lg md:text-xl text-gray-600 max-w-2xl mx-auto">
    Spec-driven coordination for AI-assisted development.
  </p>
  <div class="hero-ctas mt-8 flex flex-col sm:flex-row gap-4 justify-center">
    <!-- CTAs unchanged -->
  </div>
</div>
```

- [ ] **Step 4: Verify the full animation sequence**

Open in browser. Check:
- Page loads with curtains closed (theater red fills hero)
- Curtains sweep outward over ~1.5s with smooth easing
- Illustration is revealed progressively
- Headline fades up into view at ~2.2s
- CTAs appear at ~2.4s
- Total sequence completes in ~2.6s
- Disable animation: in DevTools, enable "prefers-reduced-motion" — everything should appear instantly with no animation

- [ ] **Step 5: Commit**

```bash
git add apps/landing/src/index.html
git commit -m "feat(landing): add layered curtain reveal animation with reduced-motion fallback"
```

---

## Task 5: Update Remaining Sections for Light Theme Consistency

**Files:**
- Modify: `apps/landing/src/index.html` — update Why, How-it-works, CTA, and Footer sections

The hero is now on a light theme but the remaining sections still use dark theme classes. This task brings them into alignment.

- [ ] **Step 1: Update the Why Synchestra section**

Replace dark theme classes:
- `border-gray-800` → `border-gray-200`
- `text-gray-400` → `text-gray-500`
- `bg-gray-900` → `bg-white`
- `border-gray-800` (on cards) → `border-gray-200`
- Card backgrounds: add `shadow-sm` for subtle elevation on light background

- [ ] **Step 2: Update the How It Works section**

Replace dark theme classes:
- `border-gray-800` → `border-gray-200`
- `text-gray-400` → `text-gray-500`
- `bg-indigo-600` (step circles) → `bg-navy`

- [ ] **Step 3: Update the CTA section**

Replace:
- `border-gray-800` → `border-gray-200`
- `text-gray-400` → `text-gray-500`
- `bg-indigo-600` → `bg-navy`
- `hover:bg-indigo-500` → `hover:bg-navy-light`

- [ ] **Step 4: Update the Footer**

Replace:
- `border-gray-800` → `border-gray-200`
- `text-gray-500` stays (works on light)
- `hover:text-gray-300` → `hover:text-navy`

- [ ] **Step 5: Replace all remaining `indigo-600` references**

Search the file for any remaining `indigo` classes and replace with the navy brand color equivalents.

- [ ] **Step 6: Verify full page consistency**

Scroll through the entire page. Check:
- All sections have light backgrounds
- No dark theme remnants (gray-800 borders, gray-900 backgrounds)
- All accent colors are navy, not indigo
- Cards are readable on white background
- Footer links work

- [ ] **Step 7: Commit**

```bash
git add apps/landing/src/index.html
git commit -m "feat(landing): update all sections to light theme with brand colors"
```

---

## Task 6: Generate Hero Illustration with AI

**Files:**
- Create/Replace: `apps/landing/src/assets/hero-scene.webp`
- Create/Replace: `apps/landing/src/assets/hero-scene-mobile.webp`

This task generates the actual Sempé-inspired pencil illustration using AI image generation. This is a creative/iterative task, not a code task.

- [ ] **Step 1: Craft the image generation prompt**

Use the Image Prompt Engineer agent (per `spec/branding/how-to.md`) to craft a prompt based on these requirements from the spec:

- **Style:** Sempé-inspired pencil illustration — warm, whimsical, clean outlines with selective detail
- **Scene:** Amateur orchestra on a theater stage, viewed from slightly elevated audience POV (~5th row center)
- **Center:** Conductor/Architect figure, 3/4 angle, one arm raised with baton, confident composed posture
- **Behind conductor:** 5-7 musicians in loose semicircle, each distinct (different clothing, posture, instruments)
- **Stage:** Warm wood floor, music stands with sheet music
- **Upper third:** Intentionally sparse/lighter — text-safe zone for headline overlay
- **Color:** Primarily graphite pencil (`#404040`). Selective color only on: conductor's vest (burgundy), baton (gold), sheet music (blueprint blue). Everything else is pencil/graphite.
- **No curtains** in the illustration (these are CSS)
- **Wide aspect ratio:** ~16:9 or 2:1
- **White background**

- [ ] **Step 2: Generate and iterate**

Generate the image. Evaluate against the spec:
- Does it feel like Sempé? (Warm, individual characters, expressive line work)
- Is the upper third sparse enough for text overlay?
- Are the musicians distinct individuals?
- Is the color selective (only vest, baton, sheet music)?

Iterate on the prompt until the result matches. Consider the hybrid approach (generate, then trace in vector tool) if the AI can't achieve the exact pencil aesthetic.

- [ ] **Step 3: Create responsive variants**

- `hero-scene.webp`: Full scene, 1920x1080 or similar wide format, optimized for web (target <400kb)
- `hero-scene-mobile.webp`: Center-cropped to focus on the architect, ~800x900 or similar portrait crop

- [ ] **Step 4: Replace placeholder assets**

Replace the placeholder files in `apps/landing/src/assets/` with the final illustrations.

- [ ] **Step 5: Verify in browser**

Open the landing page. Check:
- Illustration fills the hero background naturally
- Text is readable over the lighter upper portion
- Curtain animation reveals the scene correctly
- Mobile viewport shows the center-cropped version focused on the architect
- Image loads quickly (check file size)

- [ ] **Step 6: Commit**

```bash
git add apps/landing/src/assets/
git commit -m "feat(landing): add Sempé-inspired hero illustration"
```

---

## Outstanding Questions

- The animation overlay technique (Beats 2-3b: musician/architect opacity transitions) depends on the final illustration composition. The overlay approach in Task 4 uses full-element white overlays that fade out — this works as a simple reveal but doesn't achieve per-character opacity control. If per-character animation is needed, the illustration would need to be sliced into separate layers (architect layer, musicians layer, background layer) and composited with CSS. This is a scope decision to make after the illustration is generated.
- Retina asset (`hero-scene-2x.webp`) is mentioned in the spec but may not be needed if the base image is high enough resolution. Decide after testing.
