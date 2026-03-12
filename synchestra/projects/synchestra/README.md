# Project: Synchestra

Synchestra's own development, managed by Synchestra.

## Features

| Feature | Status | Description |
|---|---|---|
| [micro-tasks](features/micro-tasks/README.md) | Conceptual | Pre/post prompt micro-task chains and background automation |
| [cross-repo-sync](features/cross-repo-sync/README.md) | Conceptual | Cross-repository branching, task coordination, and merge strategy |
| [model-selection](features/model-selection/README.md) | Conceptual | Smart model routing based on task complexity and configuration |
| [conflict-resolution](features/conflict-resolution/README.md) | Conceptual | AI-powered merge conflict detection and resolution |
| [outstanding-questions](features/outstanding-questions/README.md) | Conceptual | Question lifecycle management linked to tasks and features |
| [claim-and-push](features/claim-and-push/README.md) | Conceptual | Distributed task claiming via git push-based optimistic locking |

## Outstanding Questions

- What is the priority order for building these features?
- Which features are prerequisites for others? (e.g., claim-and-push likely needs to exist before cross-repo-sync)
- **Task status document format is a cross-cutting decision.** Is task status stored as YAML frontmatter in the task's README? In the parent's README as a list? In a separate `status.yaml`? This affects claim-and-push, micro-tasks, and cross-repo-sync -- should be decided early.
- **Relationship between `docs/features/` and `synchestra/projects/synchestra/features/`.** The first is user-facing product documentation, the second is internal development specs. They should cross-reference each other but stay separate. How do we keep them in sync?

### Children with outstanding questions:

- [features/](features/README.md): 3 outstanding questions
  - [micro-tasks](features/micro-tasks/README.md): 4 outstanding questions
  - [cross-repo-sync](features/cross-repo-sync/README.md): 4 outstanding questions
  - [model-selection](features/model-selection/README.md): 4 outstanding questions
  - [conflict-resolution](features/conflict-resolution/README.md): 3 outstanding questions
  - [outstanding-questions](features/outstanding-questions/README.md): 3 outstanding questions
  - [claim-and-push](features/claim-and-push/README.md): 3 outstanding questions
