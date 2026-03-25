# Synchestra Branding

This specification defines Synchestra's brand identity, visual system, and communication guidelines. It is the authoritative reference for all brand decisions across the website, marketing materials, product illustrations, and documentation.

## Children

| Directory | Description |
|---|---|
| [website/](website/README.md) | Website-specific brand application — layout, hero animation, section structure |

## Brand Identity

### Name Etymology

"Synchestra" = synchronized + orchestra. The metaphor is structural, not decorative.

### Brand Personality (Layered)

Synchestra operates with a layered personality that adapts tone to audience while maintaining a consistent core:

- **Core: The Score** — systematic, elegant, structured. A beautifully precise notation system. This is the DNA of the brand.
- **OSS/community tone: The Ensemble** — collaborative, warm, interconnected. Used when addressing open-source projects, small teams, and community contexts.
- **Enterprise tone: The Conductor** — authoritative, precise, in command. Used when addressing corporate customers.

### Positioning

Synchestra brings engineering discipline — linting, validation, schema enforcement — to the coordination of AI agents and the specifications that drive them. It is the infrastructure layer between your agents, your humans, and your systems.

### Metaphor Usage

| Layer | Approach |
|---|---|
| **Naming** | Baked in — Synchestra, Rehearse. No explanation needed. |
| **Copy** | Selective — orchestra analogies used only where they genuinely clarify a concept, never as decoration. |
| **Illustrations** | Primary — the architect-conductor on a theater stage directing musicians/agents is the central visual world. |

## Color System

### UI Palette

| Role | Color | Hex |
|---|---|---|
| Primary | Deep Navy Blueprint | `#1a3a6a` |
| Primary Light | Medium Blueprint | `#2a5a9a` |
| Red accent | | `#c83030` |
| Gold accent | | `#c09018` |
| Green accent | | `#1a7a5a` |
| Purple accent | | `#7a4aaa` |
| Background | White | `#ffffff` |
| Body text | Dark gray | `#333333` |

The primary color is derived from traditional architect's blueprint blue. It signals trust, stability, and depth — appropriate for foundational infrastructure tooling. The multicolor accent system reflects the breadth of Synchestra's capabilities (task management, agent coordination, state sync, human oversight, skills).

### Illustration Palette

The illustration palette is extended but curated, broader than the UI palette. The wireframe world uses graphite ink on white paper. Key characters and meaningful objects receive color from a richer set.

| Element | Color | Hex | Notes |
|---|---|---|---|
| Wireframe stroke | Graphite | `#404040` | All non-colored line work |
| Curtains | Theater Red | `#c42b2b` | Matches `#c83030` UI accent family |
| Curtain folds/highlights | Light Red | `#d94040` | Lower opacity overlays |
| Baton | Gold | `#c09018` | Matches UI gold accent |
| Plans/blueprints | Blueprint Blue | `#1a5ea0` | Slightly different from UI navy |
| Plans fill | Light Blue | `#eef2f8` | Subtle fill behind plan lines |
| Architect's vest | Burgundy | `#7a2a2a` | Warm, distinguishes from curtain red |
| Cello/wooden instruments | Warm Brown | `#8a6030` | Natural, organic |
| Metronome pendulum/light | Gold | `#c09018` | Reuses baton gold |
| Metronome indicator | Green | `#2a8a50` | Active/timing signal |

This table is a starting reference, not exhaustive. New colored elements may be added as new scenes are created, provided they follow the rule below and maintain visual consistency with existing entries.

**Rule: color in illustrations = attention.** If an element has color, it matters to the scene. If it is wireframe, it is context. This hierarchy must be maintained as new illustration scenes are created.

## Typography

| Role | Font | Weight | Notes |
|---|---|---|---|
| Headings & labels | JetBrains Mono | 600–700 | Gives CLI/technical identity |
| Body text | Inter | 400–500 | Clean, neutral, high readability |
| Code | JetBrains Mono | 400 | Consistent with heading font |
| Regular buttons/CTAs | Inter | 600 | Clean and clickable |
| Special/hero buttons | JetBrains Mono | 500 | Draws attention, signals importance |
| Annotations in illustrations | JetBrains Mono | 400 | Reinforces technical drawing feel |

### Rationale

Monospace headings project a CLI and robotics aesthetic — the distinctive feature Synchestra brings to the docs/specifications world. The JetBrains Mono + Inter pairing says "this tool lives where your code lives" while keeping body text highly readable.

Monospace is used as a **highlight device**: regular UI elements use Inter, but monospace signals "pay attention, this is the important thing" — whether in headings, special buttons, or illustration annotations.

## Illustration Style

### Two Visual Registers

The brand uses two complementary illustration styles that share a common palette but differ in purpose and detail level:

| Register | Used For | Style |
|---|---|---|
| **Character illustrations** | Website scenes, marketing, storytelling | Sempé-inspired pencil drawing — warm, expressive, individual |
| **Technical diagrams** | Product UI, documentation, architecture | Clean wireframe — precise, schematic, functional |

Both registers use the same graphite base (`#404040`) and selective color philosophy (color = attention). They coexist under the brand: **human warmth in storytelling, technical precision in the product.**

### Character Illustrations: Sempé-Inspired Pencil Style

Inspired by Jean-Jacques Sempé's editorial illustrations. Warm, slightly whimsical pencil work where every character feels alive through a few perfectly chosen details — a tilt of the head, the way someone holds their instrument, a distinctive piece of clothing.

**Detail level:** Clean outlines with selective detail. Faces have expressions, clothing is distinct, but not every fold is drawn. Important details pop, the rest is suggested. Confident pencil strokes with lots of white space and breathing room.

| Property | Value |
|---|---|
| Line work | Graphite pencil `#404040` |
| Background | White `#ffffff` |
| Foreground figures | Full pencil detail — expressions, posture, clothing |
| Background figures | Lighter pencil work, 30–35% opacity, less detail |
| Colored elements | Full saturation, selective — only meaningful objects |

### Technical Diagrams: Wireframe Style

Used in product interfaces, documentation, and architecture diagrams. Clean vector strokes, schematic figures, minimal personality.

| Property | Value |
|---|---|
| Wireframe stroke color | Graphite `#404040` |
| Background | White `#ffffff` |
| Grid | Subtle blueprint grid underlay, barely visible |
| Foreground figures | Graphite at full weight (stroke-width ~1.6–1.8) |
| Background figures | Graphite at 30–35% opacity, less detail |
| Colored elements | Full saturation, extended curated palette |
| Annotations | JetBrains Mono, e.g. `FIG. 1 — ARCHITECT / CONDUCTOR` |

### The Recurring Cast

#### The Architect (Conductor)

Central character. Conducts and holds plans. Represents the human or Synchestra itself. Full pencil detail with colored vest (burgundy `#7a2a2a`) and baton (gold `#c09018`). Confident, composed posture — authority without rigidity.

#### The Musicians (The Orchestra)

A world-class ensemble that sells out Carnegie Hall — but still plays Tuesday nights at their local pub for friends and family. Virtuosos who chose music for love, not fame. Each member has a life beyond the stage (a profession, a family, a neighborhood), and those lives make the music richer. Synchestra is built with the same philosophy: serious, professional-grade tooling — shared freely, built for the community, used by people who care about craft.

The word "amateur" comes from the Latin *amator* — one who does it for love. These musicians are amateurs in that original sense: world-class in skill, motivated by passion rather than prestige.

Each musician is a distinct character with:

- **A cultural vibe** that guides their visual personality (used as a creative brief tool, not communicated to viewers)
- **A distinct personality** conveyed through posture, expression, and clothing details
- **Their own instrument** — a mix of classical and folk/popular, whatever people brought
- **A backstory** that makes them recognizable across scenes

A returning visitor should start to recognize individuals across different illustrations on the site.

##### The Cast

| # | Character | Instrument | Creative Vibe | Personality |
|---|---|---|---|---|
| — | **The Architect** | Baton | Central European precision, quiet warmth | The calm center of gravity — authority through competence, not volume |
| 1 | **The Cellist** | Cello | Russian intellectual warmth | The scholar — studies carefully before speaking |
| 2 | **The Flutist** | Flute | Parisian effortlessness | The questioner — finds the flaw in the plan |
| 3 | **The Guitarist** | Classical guitar | Latin American ease | The enthusiast — already on board, makes everyone relax |
| 4 | **The Accordionist** | Accordion | Italian expressiveness | The skeptic — strong opinions, deeply loyal once committed |
| 5 | **The Violinist** | Violin | Japanese quiet precision | The observer — says little, misses nothing |
| 6 | **The Drummer** | Cajón + percussion kit | Black cool, jazz/reggae soul | The black sheep — does his own thing, but holds it all together when it matters |

**Creative vibe note:** Cultural backgrounds are used as shorthand to guide character energy, clothing style, and body language in AI image generation and illustration briefs. They are not labeled or communicated to website visitors — the viewer sees diverse, distinct individuals, not nationalities.

**The Drummer's special role:** In scenes about flexibility, customization, or adaptive behavior, the Drummer becomes the featured character. Jazz improvisation = systems that adapt to whatever comes. Reggae groove = steady, reliable rhythm underneath apparent chaos.

**Visual treatment:** Pencil outlines, subordinate to the architect. 30–35% opacity in ensemble scenes.

#### The Setting

| Element | Role | Visual Treatment |
|---|---|---|
| **The Stage** | The coordination space — the repository. | Floor line, red curtains at edges |
| **The Plans/Score** | Blueprints the architect holds. Represent specs, tasks, state. | Drawn in blue (`#1a5ea0`) |
| **Objects** | Metronome (timing/scheduling), music stands (task queues), sheet music (specifications), instruments. | Pencil base, colored when significant |

### Color Hierarchy in Illustrations

Characters and objects exist on a spectrum from pure pencil to selectively colored:

1. **Background context** — pencil only, low opacity (musicians in background, stage floor)
2. **Supporting elements** — pencil at full opacity (foreground musicians, furniture)
3. **Key objects** — pencil with selective color (instruments of featured musicians, metronome pendulum)
4. **Primary characters and dramatic elements** — colored accents on pencil base (architect's vest, baton, curtains, plans)

**When musicians get color:** Musicians are pencil-only by default. A musician becomes "featured" — and receives color on their instrument or figure — when a scene is illustrating a concept that maps to a specific agent role. For example, in a scene about "task claiming," the musician reaching for a sheet of music would have a colored instrument to draw the eye. In ensemble/overview scenes, all musicians remain pencil-only. The rule: **one scene, one focus, one featured musician at most.**

## Voice & Tone

### Voice System

| Context | Voice | Style |
|---|---|---|
| Problem/context copy | Technical but warm | Empathetic, relatable, shows deep understanding of the pain |
| Solution/feature copy | Confident and declarative | Bold claims, no hedging, delivers with authority |
| Headings | Declarative | Short, punchy, assertive |
| Documentation/how-it-works | Technical but warm | Clear, precise, human, allows occasional dry humor |
| Orchestra metaphors | Selective | Only where they genuinely illuminate a concept |

### Narrative Pattern

**Open with warmth, close with confidence.** Problem sections meet the reader where they are — describing pain so accurately the reader knows the author has lived it. Solution sections deliver the answer with conviction and zero hedging.

This maps to the page structure:
- **Section openings / problem statements** → Technical but warm voice
- **Feature statements / solutions / CTAs** → Confident and declarative voice
- **Hero headline** → Declarative

### Example

> Running one AI agent on one task works great. Running five agents across three platforms on a tree of interdependent tasks? That's where things fall apart. Context gets expensive, state scatters across chat logs and git history, and you become the glue — copying context between sessions, checking what's done, figuring out what's blocked.
>
> **Synchestra is the coordination layer.** One repo. One source of truth. Agents claim tasks atomically. State is schema-validated at every commit. Humans see exactly what's happening. No server needed. No database. Git is the protocol.

## Outstanding Questions

None at this time.
