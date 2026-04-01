# SpecScore Unified Plan/Task Model Alignment

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align Synchestra's specs, architecture docs, AGENTS.md, and skills with the SpecScore unified plan/task model — rename "development plan" to "plan," update URLs from `development-plan/` to `plan/`, remove references to immutability/superseded, add references to SpecScore's new `task` feature, and update the spec-to-execution architecture doc to reflect the new model.

**Spec:** [SpecScore Unified Plan/Task Model Design](https://github.com/synchestra-io/specscore/blob/main/docs/superpowers/specs/2026-04-01-unified-plan-task-model-design.md)

**Scope:** Synchestra repository only. SpecScore changes are already merged.

---

## What Changed in SpecScore

The SpecScore repository has merged these changes:

1. **`development-plan/` renamed to `plan/`** — all URLs with `/spec/features/development-plan/` are now `/spec/features/plan/`
2. **New `task` feature** at `spec/features/task/` — defines task as the atomic unit of work (statuses, dependencies, board format)
3. **"Development plan" → "Plan"** throughout
4. **"Steps" → "Tasks"** — plan steps are now called tasks
5. **Immutability removed** — plans are mutable; snapshots (git hash + action + comment) replace freezing
6. **`superseded` status removed** — plans have only `draft`, `in_review`, `approved`
7. **Nesting limits removed** — tasks nest recursively, no 2-level cap
8. **Roadmap concept merged** — a plan with sub-plans is just a composite task

## File Map

**Modified files (URL + terminology updates):**
- `spec/features/README.md` — update SpecScore feature listing and dependency graph
- `spec/features/task-status-board/README.md` — update "Interaction with Development Plans" section
- `spec/features/state-store/task-store/README.md` — update artifact reference URL
- `spec/features/proposals/README.md` — update "Interaction with Development Plans" section
- `spec/features/chat/README.md` — update plan references
- `spec/features/chat/workflow/create-proposal/README.md` — update plan references
- `spec/features/chat/workflow/README.md` — update produces list
- `spec/features/chat/workflow/tweak-document/README.md` — update plan reference
- `spec/features/stakeholder/README.md` — update plan references and diagram
- `spec/features/stakeholder/gate/README.md` — update plan reference
- `spec/features/stakeholder/decision/README.md` — update plan references
- `spec/features/stakeholder/role/README.md` — update plan references
- `spec/features/testing-framework/README.md` — update plan reference
- `spec/features/onboarding/README.md` — update plan reference
- `spec/features/github-app/README.md` — update plan reference
- `spec/features/lsp/README.md` — update plan reference
- `spec/features/cli/spec/search/_args/type.md` — update plan description
- `spec/architecture/spec-to-execution.md` — major rewrite for new model
- `spec/architecture/README.md` — update description
- `spec/architecture/diagrams/work-flowchart.md` — update diagram label
- `spec/plans/README.md` — update convention description
- `AGENTS.md` — update plans section
- `README.md` — update plan references
- `docs/README.md` — update plan references
- `docs/superpowers/README.md` — update plan references
- `docs/superpowers/plans/README.md` — update plan references
- `ai-plugin/skills/synchestra-whats-next/SKILL.md` — update CLI reference

---

### Task 1: Update SpecScore feature listing in `spec/features/README.md`

**Files:**
- Modify: `spec/features/README.md`

This is the central file that lists SpecScore features and their dependency graph. It has the most references to `development-plan`.

- [ ] **Step 1: Update the SpecScore feature list**

  In the "Specification Format" section, change:
  ```markdown
  - [Development Plan](https://github.com/synchestra-io/specscore/blob/main/spec/features/development-plan/README.md) — Planning document format
  ```
  To:
  ```markdown
  - [Plan](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md) — Planning document format
  - [Task](https://github.com/synchestra-io/specscore/blob/main/spec/features/task/README.md) — Discrete units of work within a plan
  ```

- [ ] **Step 2: Update the dependency graph comment block**

  Replace all `development-plan` references in the comment block (lines ~151-182):
  - `# SpecScore features (external): feature, acceptance-criteria, source-references, development-plan, project-definition` → `..., plan, task, project-definition`
  - `proposals → [specscore:development-plan]` → `proposals → [specscore:plan]`
  - `[specscore:development-plan] → task-status-board, cli` → `[specscore:plan] → task-status-board, cli`
  - All other `[specscore:development-plan]` → `[specscore:plan]`

- [ ] **Step 3: Update the prose below the dependency graph**

  Change:
  ```markdown
  [SpecScore `development-plan`](https://github.com/synchestra-io/specscore/blob/main/spec/features/development-plan/README.md) bridges the spec-to-execution gap — proposals and feature specs flow through it to become tasks.
  ```
  To:
  ```markdown
  [SpecScore `plan`](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md) bridges the spec-to-execution gap — proposals and feature specs flow through plans to become tasks. The [task](https://github.com/synchestra-io/specscore/blob/main/spec/features/task/README.md) feature defines the methodology-level task concept that Synchestra implements.
  ```

- [ ] **Step 4: Commit**

  ```bash
  git add spec/features/README.md
  git commit -m "refactor: update SpecScore feature references from development-plan to plan"
  ```

---

### Task 2: Update task-status-board and state-store specs

**Files:**
- Modify: `spec/features/task-status-board/README.md`
- Modify: `spec/features/state-store/task-store/README.md`

These are the core task execution specs that reference SpecScore's development-plan feature.

- [ ] **Step 1: Update task-status-board "Interaction with Development Plans" section**

  Rename the section header from `## Interaction with Development Plans` to `## Interaction with Plans`.

  Update the content:
  ```markdown
  ## Interaction with Plans

  See [Spec-to-Execution Pipeline](../../architecture/spec-to-execution.md) for the full architectural view of how features, plans, and tasks connect across repository boundaries.

  Tasks generated from a [plan](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md) appear on the board like any other task. Each task's README carries a back-reference to its plan and plan task, but the board itself is unaware of plans — it tracks task status regardless of how tasks were created.

  The plan feature provides a derived status view (`synchestra plan status`) that reads plan task references from tasks and aggregates board status into a plan-oriented progress report.
  ```

  Note: Remove the broken anchor link to `#derived-status-view` — that section may not exist in the new plan spec at that exact anchor.

- [ ] **Step 2: Add reference to SpecScore task feature**

  In the task-status-board README, the statuses, lifecycle, dependency references, and board format now have a canonical definition in SpecScore's task feature. Add a note near the top of the Behavior section or after the status lifecycle:

  ```markdown
  The task methodology — statuses, lifecycle, dependency references, and board format — is defined in the [SpecScore task feature](https://github.com/synchestra-io/specscore/blob/main/spec/features/task/README.md). Synchestra implements and extends that methodology with operational tooling: claiming, optimistic locking, CLI commands, and board rendering.
  ```

- [ ] **Step 3: Update state-store/task-store artifact reference**

  Change the artifact reference URL:
  ```markdown
  See [Plan](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md) for how artifacts are declared in plans and consumed by downstream tasks.
  ```

- [ ] **Step 4: Commit**

  ```bash
  git add spec/features/task-status-board/README.md spec/features/state-store/task-store/README.md
  git commit -m "refactor: align task-status-board and task-store with SpecScore plan/task model"
  ```

---

### Task 3: Update proposal, chat, and workflow specs

**Files:**
- Modify: `spec/features/proposals/README.md`
- Modify: `spec/features/chat/README.md`
- Modify: `spec/features/chat/workflow/create-proposal/README.md`
- Modify: `spec/features/chat/workflow/README.md`
- Modify: `spec/features/chat/workflow/tweak-document/README.md`

All of these reference "development plan" in prose and URLs.

- [ ] **Step 1: Update proposals/README.md**

  - Rename section `## Interaction with Development Plans` → `## Interaction with Plans`
  - Change `[development plan](https://github.com/synchestra-io/specscore/blob/main/spec/features/development-plan/README.md)` → `[plan](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md)`
  - Change "development plan creation" → "plan creation"

- [ ] **Step 2: Update chat/README.md**

  - Replace all `development plan` prose with `plan`
  - Update URL: `development-plan/README.md` → `plan/README.md`
  - In the Interaction table, change `[Development Plan](...)` → `[Plan](...)`
  - Change "produces a development plan as a report" → "produces a plan as a report"

- [ ] **Step 3: Update chat/workflow/create-proposal/README.md**

  - Replace all `development plan` prose with `plan`
  - Update URL: `development-plan/README.md` → `plan/README.md`
  - Change `**Produces (fast path):** \`proposal\`, \`development-plan\`, \`pull-request\`` → `**Produces (fast path):** \`proposal\`, \`plan\`, \`pull-request\``
  - In the mermaid diagram, change `Development Plan` → `Plan`

- [ ] **Step 4: Update chat/workflow/README.md**

  - Change `development plan` prose → `plan`
  - Change `produces: [proposal, development-plan, pull-request]` → `produces: [proposal, plan, pull-request]`

- [ ] **Step 5: Update chat/workflow/tweak-document/README.md**

  - Change `development plan` → `plan` in the multi-repo limitation note

- [ ] **Step 6: Commit**

  ```bash
  git add spec/features/proposals/README.md spec/features/chat/README.md spec/features/chat/workflow/create-proposal/README.md spec/features/chat/workflow/README.md spec/features/chat/workflow/tweak-document/README.md
  git commit -m "refactor: update proposal and chat specs from development-plan to plan"
  ```

---

### Task 4: Update stakeholder, testing, onboarding, github-app, and LSP specs

**Files:**
- Modify: `spec/features/stakeholder/README.md`
- Modify: `spec/features/stakeholder/gate/README.md`
- Modify: `spec/features/stakeholder/decision/README.md`
- Modify: `spec/features/stakeholder/role/README.md`
- Modify: `spec/features/testing-framework/README.md`
- Modify: `spec/features/onboarding/README.md`
- Modify: `spec/features/github-app/README.md`
- Modify: `spec/features/lsp/README.md`
- Modify: `spec/features/cli/spec/search/_args/type.md`

These files have 1-3 references each — straightforward find-and-replace.

- [ ] **Step 1: Update stakeholder specs**

  In `stakeholder/README.md`:
  - Change "development plans go through `in_review → approved`" → "plans go through `in_review → approved`"
  - Update mermaid diagram label `Development plan<br/>(how)` → `Plan<br/>(how)`
  - Update Interaction table: `[development-plan](...)` → `[plan](...)`
  - Update URL: `development-plan/README.md` → `plan/README.md`

  In `stakeholder/gate/README.md`:
  - Change "development plans transition through" → "plans transition through"

  In `stakeholder/decision/README.md`:
  - Change "a development plan needs review" → "a plan needs review"
  - Change "the development plan for adding batch mode" → "the plan for adding batch mode"

  In `stakeholder/role/README.md`:
  - Change "a development plan needs review" → "a plan needs review"
  - Change "feature specs and development plans" → "feature specs and plans"

- [ ] **Step 2: Update testing-framework, onboarding, github-app, lsp specs**

  In `testing-framework/README.md`:
  - Update Interaction table: `[Development Plan](...)` → `[Plan](...)`
  - Update URL: `development-plan/README.md` → `plan/README.md`
  - Change "Plan step ACs" → "Plan task ACs"

  In `onboarding/README.md`:
  - Change "A sample development plan" → "A sample plan"

  In `github-app/README.md`:
  - Change "create PRs from development plans" → "create PRs from plans"

  In `lsp/README.md`:
  - Change "development plans, proposal documents" → "plans, proposal documents"

- [ ] **Step 3: Update CLI search type description**

  In `spec/features/cli/spec/search/_args/type.md`:
  - Change `| \`plan\` | \`spec/plans/\` | Development plans |` → `| \`plan\` | \`spec/plans/\` | Plans |`

- [ ] **Step 4: Commit**

  ```bash
  git add spec/features/stakeholder/ spec/features/testing-framework/README.md spec/features/onboarding/README.md spec/features/github-app/README.md spec/features/lsp/README.md spec/features/cli/spec/search/_args/type.md
  git commit -m "refactor: update remaining feature specs from development-plan to plan"
  ```

---

### Task 5: Rewrite spec-to-execution architecture document

**Files:**
- Modify: `spec/architecture/spec-to-execution.md`
- Modify: `spec/architecture/README.md`
- Modify: `spec/architecture/diagrams/work-flowchart.md`

The spec-to-execution document is the most impactful change — it describes plans as "immutable once approved" and "frozen," which is no longer true. This needs a conceptual update, not just find-and-replace.

- [ ] **Step 1: Update "The three layers" mermaid diagram**

  Change `DP["Development Plan<br/>how to build it"]` → `DP["Plan<br/>how to build it"]`

- [ ] **Step 2: Update the mutability table**

  Change:
  ```markdown
  | Approach (how)   | Development plan  | Immutable once approved | Spec   |
  ```
  To:
  ```markdown
  | Approach (how)   | Plan              | Mutable; snapshots track history | Spec   |
  ```

- [ ] **Step 3: Update the lifecycle sequence diagram**

  Change:
  ```
  participant Plan as Dev Plan
  ...
  Human->>Plan: Author development plan
  Note over Plan: draft → in_review → approved (frozen)
  ...
  Plan->>Tasks: Generate tasks from steps
  ```
  To:
  ```
  participant Plan as Plan
  ...
  Human->>Plan: Author plan
  Note over Plan: draft → in_review → approved
  ...
  Plan->>Tasks: Generate tasks from plan tasks
  ```

- [ ] **Step 4: Update the "Status and mutability" section**

  Replace the "Frozen" subgraph:
  ```markdown
  subgraph "Snapshot-tracked"
      DP["Plan<br/>──────────<br/>Mutable document.<br/>Snapshots capture<br/>reference points.<br/>Never tracks status."]
  end
  ```

  Update the explanation:
  - Change "Plans are **frozen** because reviewers need a stable document..." → "Plans are **mutable** — they evolve as execution reveals complexity. Snapshots (git hash + action + comment) capture reference points for review and retrospective."
  - Change "The development plan bridges these two worlds. It is the last frozen artifact before execution begins — the point where intent crystallizes into a reviewable, approvable approach." → "The plan bridges these two worlds. Snapshots mark the moments where intent crystallizes — approval, checkpoints, completion — while the plan itself remains a living document."

- [ ] **Step 5: Update the derived status section**

  Change "Plan (frozen, in spec repo)" subgraph label → "Plan (in spec repo)"
  Change references to "plan steps" → "plan tasks"

- [ ] **Step 6: Update architecture/README.md description**

  Change "development plans" → "plans" in the spec-to-execution description.

- [ ] **Step 7: Update work-flowchart.md**

  Change `PLAN[Development Plan]` → `PLAN[Plan]`

- [ ] **Step 8: Commit**

  ```bash
  git add spec/architecture/
  git commit -m "refactor: update spec-to-execution architecture for unified plan/task model"
  ```

---

### Task 6: Update AGENTS.md, README.md, and docs

**Files:**
- Modify: `AGENTS.md`
- Modify: `README.md`
- Modify: `docs/README.md`
- Modify: `docs/superpowers/README.md`
- Modify: `docs/superpowers/plans/README.md`
- Modify: `spec/plans/README.md`

- [ ] **Step 1: Rewrite AGENTS.md "Development plans location and format" section**

  Replace the section (lines ~87-96):
  ```markdown
  ## Plans location and format

  All plans must be created in `spec/plans/` and follow the structure defined in [Plan specification](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md).

  - Plans start in `draft` status and follow the approval workflow: `draft` → `in_review` → `approved`
  - Plans are mutable; snapshots (git hash + action + comment) capture reference points for review and retrospective
  - Plans live nowhere else — not in `docs/superpowers/`, not in project directories, not in temporary locations
  - Use `synchestra plan create` to scaffold a new plan; use `synchestra plan submit` and `synchestra plan approve` for workflow progression

  See the [Plan specification](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md#behavior) for complete structure, field requirements, and task generation rules.
  ```

- [ ] **Step 2: Update README.md**

  - Change `plans/                           # Development plans (bridge specs → tasks)` → `plans/                           # Plans (bridge specs → tasks)`
  - Change "Active [development plans](spec/plans/README.md)" → "Active [plans](spec/plans/README.md)"
  - Change "Plans support hierarchical nesting (roadmaps containing child plans)" → "Plans nest recursively — a plan is a composite task whose children may themselves be plans"
  - Update the SpecScore reference URL: `development-plan` → `plan`
  - Remove mention of "immutable approval workflow" if present

- [ ] **Step 3: Update docs/README.md**

  Change "immutable approval workflow" to "approval workflow with snapshots" or similar.

- [ ] **Step 4: Update docs/superpowers/README.md and docs/superpowers/plans/README.md**

  - Replace "development plans" with "plans"
  - Update link paths from `development-plan/` to `plan/`
  - Note: The `docs/superpowers/plans/README.md` link to `../../features/development-plan/README.md` is a local path that no longer exists. Since the feature spec now lives in SpecScore, update to the GitHub URL: `https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md`

- [ ] **Step 5: Update spec/plans/README.md**

  Change "development plans should use mermaid syntax" → "plans should use mermaid syntax"

- [ ] **Step 6: Update ai-plugin/skills/synchestra-whats-next/SKILL.md**

  Change:
  ```markdown
  **CLI reference:** [development-plan feature spec](../../spec/features/development-plan/README.md)
  ```
  To:
  ```markdown
  **CLI reference:** [plan feature spec](https://github.com/synchestra-io/specscore/blob/main/spec/features/plan/README.md)
  ```

- [ ] **Step 7: Commit**

  ```bash
  git add AGENTS.md README.md docs/ ai-plugin/skills/synchestra-whats-next/SKILL.md spec/plans/README.md
  git commit -m "refactor: update AGENTS.md, README, docs, and skills for unified plan/task model"
  ```

---

## Dependency Graph

```
Task 1 (spec/features/README.md)     — independent
Task 2 (task-status-board, task-store) — independent
Task 3 (proposals, chat, workflows)   — independent
Task 4 (stakeholder, misc specs)      — independent
Task 5 (architecture docs)            — independent
Task 6 (AGENTS.md, README, docs)      — independent
```

All tasks are parallel-eligible — they touch different files.

## What This Plan Does NOT Cover

- **Go code changes** — The `pkg/state/types.go` status constants use `completed` not `complete` and include `claimed` which is a Synchestra-specific status. These are Synchestra implementation details, not SpecScore methodology. No changes needed.
- **Historical docs/superpowers/plans/** — Previous implementation plans that reference the old terminology are historical records. They should not be updated.
- **spec/plans/ directory** — Existing concrete plans (chat-feature, etc.) use "development plan" in prose. These are frozen historical documents and should not be bulk-updated. New plans should use the new terminology.
- **Synchestra task-status-board conceptual overlap** — The board format, statuses, and dependency model are now defined in SpecScore's task feature. The task-status-board spec should reference SpecScore as the methodology authority (handled in Task 2) but does not need to remove its own definitions — Synchestra extends the SpecScore model with operational details (claiming, CLI, board rendering).

## Risks

- **Broken internal links** — Some files use relative paths like `../../features/development-plan/README.md` which pointed to a local directory that was already removed during SpecScore decoupling. These should be updated to GitHub URLs pointing to SpecScore.
- **Partial updates** — With 27+ files to update, it's easy to miss one. After all tasks complete, run a grep for `development-plan` and `development plan` to verify completeness.

## Outstanding Questions

None at this time.
