# Architecture

Architectural documents describing Synchestra's foundational design decisions and structural concepts.

These are cross-cutting concerns that features and specifications build upon.

## Contents

| Document | Description |
|---|---|
| [repository-types.md](repository-types.md) | The three repository types (spec, state, code) — what they hold, why they're separate, how they connect |
| [spec-to-execution.md](spec-to-execution.md) | How features, development plans, and tasks connect across repository boundaries |

### repository-types.md

Defines the three kinds of repositories Synchestra operates with: the spec repository (requirements and documentation), the state repository (tasks and coordination), and the code repository (implementation). Explains why the state repo must be separate, naming conventions, how repos connect through `synchestra-project.yaml`, and when spec+code repos can be combined.

### spec-to-execution.md

The end-to-end pipeline from product intent to running work. Shows how features (what), development plans (how), and tasks (who/when) relate across the spec and state repositories. Covers the three-layer architecture, the complete lifecycle sequence, artifact relationships, mutability profiles, derived status without duplication, and repository boundaries.

## Outstanding Questions

None at this time.
