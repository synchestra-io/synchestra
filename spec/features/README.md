# Features: Synchestra

Feature specifications for the Synchestra project, managed by Synchestra.

## Index

| Feature | Status | Description |
|---|---|---|
| [micro-tasks](micro-tasks/README.md) | Conceptual | Pre/post prompt micro-task chains and background automation |
| [cross-repo-sync](cross-repo-sync/README.md) | Conceptual | Cross-repository branching, task coordination, and merge strategy |
| [model-selection](model-selection/README.md) | Conceptual | Smart model routing based on task complexity and configuration |
| [conflict-resolution](conflict-resolution/README.md) | Conceptual | AI-powered merge conflict detection and resolution |
| [outstanding-questions](outstanding-questions/README.md) | Conceptual | Question lifecycle management linked to tasks and features |
| [claim-and-push](claim-and-push/README.md) | Conceptual | Distributed task claiming via git push-based optimistic locking |
| [task-status-board](task-status-board/README.md) | Conceptual | Markdown task board in task directory READMEs for at-a-glance status visibility |

## Feature dependency graph

```
claim-and-push ← conflict-resolution
       ↑                ↑
cross-repo-sync ────────┘
       ↑
micro-tasks (independent)
model-selection (independent)
outstanding-questions (independent)
```

`claim-and-push` is foundational — most concurrent features depend on it.

## Outstanding Questions

- Are there features missing from this list that are already described in `docs/features/` but not yet tracked here?
- **Suggested build order:** claim-and-push first (foundational), then outstanding-questions and model-selection (independent, high value), then conflict-resolution (depends on claim-and-push), then micro-tasks and cross-repo-sync. Does this align with project priorities?

### Features with outstanding questions:

- [micro-tasks](micro-tasks/README.md): 4 outstanding questions
- [cross-repo-sync](cross-repo-sync/README.md): 4 outstanding questions
- [model-selection](model-selection/README.md): 4 outstanding questions
- [conflict-resolution](conflict-resolution/README.md): 3 outstanding questions
- [outstanding-questions](outstanding-questions/README.md): 3 outstanding questions
- [claim-and-push](claim-and-push/README.md): 3 outstanding questions
- [task-status-board](task-status-board/README.md): 4 outstanding questions
