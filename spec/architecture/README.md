# Architecture

Architectural documents describing Synchestra's foundational design decisions and structural concepts.

These are cross-cutting concerns that features and specifications build upon.

## Contents

| Document | Description |
|---|---|
| [repository-types.md](repository-types.md) | The three repository types (spec, state, code) — what they hold, why they're separate, how they connect |

### repository-types.md

Defines the three kinds of repositories Synchestra operates with: the spec repository (requirements and documentation), the state repository (tasks and coordination), and the code repository (implementation). Explains why the state repo must be separate, naming conventions, how repos connect through `synchestra-project.yaml`, and when spec+code repos can be combined.

## Outstanding Questions

None at this time.
