# Features: Synchestra

Feature specifications for the Synchestra project, managed by Synchestra.

## Index

| Feature | Status | Description |
|---|---|---|
| [feature](feature/README.md) | Conceptual | Feature structure, metadata, lifecycle, and conventions — the atomic unit of product specification |
| [project-definition](project-definition/README.md) | Conceptual | `synchestra-spec.yaml` format and supported repository layouts |
| [micro-tasks](micro-tasks/README.md) | Conceptual | Pre/post prompt micro-task chains and background automation |
| [cross-repo-sync](cross-repo-sync/README.md) | Conceptual | Cross-repository branching, task coordination, and merge strategy |
| [model-selection](model-selection/README.md) | Conceptual | Smart model routing based on task complexity and configuration |
| [conflict-resolution](conflict-resolution/README.md) | Conceptual | AI-powered merge conflict detection and resolution |
| [outstanding-questions](outstanding-questions/README.md) | Conceptual | Question lifecycle management linked to tasks and features |
| [proposals](proposals/README.md) | Conceptual | Non-normative change requests attached to features with review status and optional tracker linkage |
| [ui](ui/README.md) | Conceptual | Human-facing interfaces for project navigation, proposals, tasks, and workers across web and terminal surfaces |
| [task-status-board](task-status-board/README.md) | Conceptual | Markdown task board in task directory READMEs for at-a-glance status visibility and claiming via optimistic locking |
| [development-plan](development-plan/README.md) | Conceptual | Immutable planning documents that bridge feature specs and change requests to executable tasks |
| [agent-skills](agent-skills/README.md) | In Progress | Dedicated, focused skills that AI agents use to interact with Synchestra |
| [cli](cli/README.md) | In Progress | The `synchestra` CLI — primary interface for agents and humans |
| [chat](chat/README.md) | Conceptual | Guided conversational interface that produces Synchestra artifacts (proposals, features, issues, PRs) through AI-assisted workflows |
| [global-config](global-config/README.md) | Conceptual | User-level `~/.synchestra.yaml` — repos directory and machine-local settings |
| [api](api/README.md) | In Progress | REST API exposing Synchestra operations over HTTP |
| [github-app](github-app/README.md) | Conceptual | GitHub App for webhook notifications, authenticated repo access, and organization-level installation |
| [onboarding](onboarding/README.md) | Conceptual | Guided wizard for first-time project setup — repo connection, GitHub App installation, AI-powered scaffolding, or demo launch |
| [sandbox](sandbox/README.md) | Conceptual | Isolated Docker container environments per project for executing user-initiated commands from the chat interface |
| [state-store](state-store/README.md) | Conceptual | Pluggable state storage abstraction — composable Go interface (`state.Store`) with git-backed default implementation |
| [acceptance-criteria](acceptance-criteria/README.md) | Conceptual | First-class, individually addressable verification artifacts with status lifecycle, typed inputs, and executable bash scripts |
| [testing-framework](testing-framework/README.md) | Conceptual | Markdown-native testing framework: scenario format, Go-based runner, and CLI integration for composing ACs into E2E and integration test flows |

## Feature Summaries

### [Feature](feature/README.md)

Defines the atomic unit of product specification in Synchestra — a feature is a directory under `spec/features/` with a mandatory README. The spec formalizes the structure every feature must follow (title, status, summary, problem, behavior, outstanding questions), the feature lifecycle (Conceptual → In Progress → Stable → Deprecated), nesting rules for sub-features, and how features relate to proposals, development plans, and tasks.

### [Project Definition](project-definition/README.md)

Defines the `synchestra-spec.yaml` format, mandatory and optional fields, the three repository types (state, spec, code), and the two supported layouts for spec repositories: multi-project (under `synchestra/projects/`) and dedicated (project files at root). Synchestra auto-detects the layout by checking for a project file at the repository root.

### [Micro-Tasks](micro-tasks/README.md)

Small, automated steps that run before, after, or in the background of a user's prompt — formatting, validation, cross-reference updates, link checks. They keep the project consistent without burning tokens from the main task's context window. Configured per-project or per-module as pre/post/background chains, modeled after GitHub Actions workflow jobs.

### [Cross-Repo Sync](cross-repo-sync/README.md)

Coordinates changes that span multiple repositories. When a task requires edits across repos (e.g., API spec + backend + frontend), Synchestra decomposes the work into sub-tasks, reserves matching branch names across all affected repos, manages dependency order, and handles the integration merge lifecycle.

### [Model Selection](model-selection/README.md)

Routes tasks to the minimal viable model to avoid wasting expensive tokens on mechanical work. Three levels of precedence: user override (CLI/API/UI), configuration rules (`model_class` mapping to platform-specific models), and dynamic assessment where a small model classifies task complexity before routing.

### [Conflict Resolution](conflict-resolution/README.md)

When git merge conflicts occur between concurrent agent operations, Synchestra launches a specialized sub-agent to analyze and resolve the conflict. Three tiers: auto-merge via git rebase, AI-assisted merge that understands change intent from task descriptions, and human escalation with a confidence threshold for ambiguous cases.

### [Outstanding Questions](outstanding-questions/README.md)

Every document maintains a structural "Outstanding Questions" section with a full lifecycle: open → linked (to a task) → resolved → recently resolved → archived. When a linked task completes, a sub-agent evaluates whether the output actually answers the question and resolves it automatically.

### [Proposals](proposals/README.md)

Proposals attach non-normative change requests directly to a feature without changing the feature's current specification. Each proposal has its own status lifecycle, can link to a GitHub issue for MVP, and is excluded from default current-state understanding unless explicitly requested.

### [UI](ui/README.md)

The human-facing product surfaces for Synchestra. Defines a shared information architecture (home → project menu → Features / Tasks / Workers) with MVP flows for proposal creation and task creation/enqueueing. Two delivery surfaces: a progressive [web app](ui/web-app/README.md) communicating via the HTTP API, and a [TUI](ui/tui/README.md) delivered through the CLI operating on local git state. Introduces the Workers concept at the UI level; a dedicated workers feature spec is needed before going beyond visibility.


### [Task Status Board](task-status-board/README.md)

A markdown table in task directory READMEs that provides at-a-glance visibility and serves as the source of truth for task state. Agents claim tasks by updating a board row and pushing through optimistic locking (git push-based). Conflicts on the same row indicate a claim collision; the CLI parses diffs by task ID to distinguish collisions from unrelated changes. See the [Claiming a Task](task-status-board/README.md#claiming-a-task-optimistic-locking) section for the full protocol.

### [Development Plan](development-plan/README.md)

An immutable planning document that bridges feature specifications and change requests to executable tasks. The plan captures the approach, rationale, acceptance criteria, and step-by-step decomposition in a flat, reviewable format (max two levels of nesting). Once approved, the plan is frozen — tasks generated from it evolve freely during execution while the plan remains a fixed reference for review gates and retrospective comparison.

### [Chat](chat/README.md)

A server-managed, goal-oriented conversational interface between humans and AI agents. Chats are the implementation layer behind user-facing actions like "Create a Proposal," "Raise an Issue," "New Feature," and "Tweak Document." Users never interact with chats directly — they interact with workflows that use chats under the hood. Each workflow is a declarative YAML recipe that defines what context to load, what AI steps to follow, and what artifacts to produce. Chats support two execution paths: a standard path where conversations produce documents that enter the normal Synchestra pipeline (proposal, plan, tasks), and a fast path for maintainers where the system implements changes during the conversation.

### [Global Config](global-config/README.md)

The user-level configuration file at `~/.synchestra.yaml`. Stores machine-local settings that apply across all projects, starting with `repos_dir` — the root directory where cloned repositories are stored on disk (default: `~/synchestra/repos`). Repo references resolve to `{repos_dir}/{hosting}/{org}/{repo}`. The file is optional; all settings have defaults.

### [Agent Skills](agent-skills/README.md)

A set of dedicated, focused skills that AI agents use to interact with Synchestra — claiming tasks, reporting status, updating progress. Each skill wraps a single CLI command with clear trigger conditions, parameters, and exit code handling. Skills are distributed via CLI, MCP server, or direct file access.

### [CLI](cli/README.md)

The `synchestra` command-line interface. Follows a `synchestra <resource> <action>` pattern with consistent exit codes, atomic git commit-and-push for mutations, and both query and update modes. Defines the task status model, valid transitions, and the `abort_requested` flag. Commands are organized as `cli/task/claim/`, `cli/task/status/`, etc.

### [API](api/README.md)

The REST API layer that exposes Synchestra's coordination capabilities over HTTP. Every mutation endpoint maps 1:1 to a CLI command, using the same atomic git semantics. Task and project identifiers are query parameters matching CLI flag conventions. The normative OpenAPI specs live in [`spec/api/`](../api/README.md).

### [GitHub App](github-app/README.md)

The Synchestra GitHub App registered under the `synchestra-io` organization. Provides real-time webhook delivery (issues, pull requests, pushes), fine-grained repository permissions, and short-lived installation tokens for authenticated API access. Users install the app at the organization or personal-account level during onboarding; Synchestra uses the installation to discover accessible repos, push state changes, and sync issue/PR activity. The app is the prerequisite for any real-time integration between Synchestra and GitHub-hosted repositories.

### [Onboarding](onboarding/README.md)

A guided wizard delivered through both the web app and the CLI that walks new users through first-time project setup. Offers two paths: "Connect your repositories" (GitHub App installation → spec repo selection → optional code repos → state repo provisioning → bring-your-own AI key → AI-powered repo analysis and scaffolding → project creation) and "Try the demo" (pre-built sample project with example features, tasks, and proposals). The wizard handles infrastructure bootstrapping — creating state repos, generating `synchestra-spec.yaml`, and scaffolding initial feature structures — so users reach a working project in minutes.

### [Sandbox](sandbox/README.md)

Isolated Docker container environments per project for executing user-initiated commands from the chat interface. Each project gets its own persistent container with encrypted credential storage (AES256), user-isolated sessions, and a gRPC agent for host↔container communication. The host is stateless and routes requests; all state, secrets, and execution data remain inside containers.

### [State Store](state-store/README.md)

The pluggable abstraction layer for all Synchestra project coordination state. Defines a composable, hierarchical Go interface (`state.Store`) with sub-interfaces for tasks (`TaskStore`), chat (`ChatStore`), and project configuration (`ProjectStore`). Navigated like CLI subcommands — `store.Task().Claim(ctx, ...)` — keeping each interface focused and discoverable. The default git-backed implementation (`gitstore`) maps to the existing state repository design; future backends (SQLite, PostgreSQL, cloud databases) satisfy the same interface.

### [Acceptance Criteria](acceptance-criteria/README.md)

Defines acceptance criteria as first-class, individually addressable artifacts within feature specs. Each AC is a separate `.md` file in a feature's `_acs/` directory with a status lifecycle (planned → wip → implemented → deprecated), typed inputs, and an executable bash verification script. Every feature README must include a mandatory Acceptance Criteria section — either a table summarizing ACs or "Not defined yet." with a corresponding outstanding question. Feature ACs are long-lived and evolve with the feature; plan ACs follow the same format but are frozen with the plan.

### [Testing Framework](testing-framework/README.md)

A markdown-native testing framework for composing acceptance criteria into multi-step integration and E2E test flows. Contains two sub-features: [test-scenario](testing-framework/test-scenario/README.md) defines the human-readable scenario format (named steps, dependency declarations, output passing, AC references, sub-flow includes, parallel groups), and [test-runner](testing-framework/test-runner/README.md) defines the Go execution engine (`pkg/testscenario/`) that parses scenarios, resolves AC verification scripts from `_acs/` directories, executes bash steps, and reports results. Scenarios live in `spec/tests/` (cross-feature E2E) or `spec/features/{feature}/_tests/` (feature-scoped).

```
feature → proposals, development-plan, outstanding-questions (features are the spec unit)
task-status-board ← conflict-resolution
       ↑                ↑
cross-repo-sync ────────┘
       ↑
micro-tasks (independent)
model-selection (independent)
outstanding-questions (independent)
proposals → development-plan (proposals trigger plans)
development-plan → task-status-board, cli (plans generate tasks)
chat → feature, proposals, development-plan, task-status-board, agent-skills, ui, api
ui → proposals, cli, task-status-board, agent-skills, development-plan, chat
api → cli (api mirrors cli contract)
global-config ← cli (cli reads ~/.synchestra.yaml for repo resolution)
github-app → api (callback endpoint)
onboarding → github-app, project-definition, ui, cli, api (orchestrates first-time setup)
sandbox → cli, api (containers execute commands, host routes via API)
state-store → task-status-board (board interface and claim atomicity), chat (chat persistence)
state-store ← cli, api, agent-skills (all consumers of state go through state store)
acceptance-criteria → feature (introduces mandatory AC section), development-plan (plan ACs can reference feature ACs)
testing-framework → acceptance-criteria (composes ACs into test flows), cli (new test command group), feature (_tests/ directory)
```

`feature` is the foundational spec-layer concept — proposals, plans, and outstanding questions all attach to features.

## Diagram Conventions

All diagrams in feature specifications should use **mermaid syntax** instead of ASCII art. Mermaid provides better clarity, GitHub rendering support, and maintainability.
`task-status-board` is foundational for execution — it provides the claiming mechanism (optimistic locking) and status visibility.
`development-plan` bridges the spec-to-execution gap — proposals and feature specs flow through it to become tasks.

## Outstanding Questions

- Are there features missing from this list that are already described in `docs/features/` but not yet tracked here?
- **Suggested build order:** task-status-board first (foundational), then outstanding-questions and model-selection (independent, high value), then proposals, then UI once CLI and proposal flows are ready enough to expose, then conflict-resolution, then micro-tasks and cross-repo-sync. Does this align with project priorities?

### Features with outstanding questions:

- [feature](feature/README.md): 4 outstanding questions
- [project-definition](project-definition/README.md): 2 outstanding questions
- [micro-tasks](micro-tasks/README.md): 4 outstanding questions
- [cross-repo-sync](cross-repo-sync/README.md): 4 outstanding questions
- [model-selection](model-selection/README.md): 4 outstanding questions
- [conflict-resolution](conflict-resolution/README.md): 3 outstanding questions
- [outstanding-questions](outstanding-questions/README.md): 3 outstanding questions
- [task-status-board](task-status-board/README.md): 4 outstanding questions
- [development-plan](development-plan/README.md): 4 outstanding questions
- [agent-skills](agent-skills/README.md): 3 outstanding questions
- [cli](cli/README.md): 3 outstanding questions
- [api](api/README.md): 3 outstanding questions
- [chat](chat/README.md): 4 outstanding questions
- [chat/workflow](chat/workflow/README.md): 4 outstanding questions
- [chat/workflow/create-proposal](chat/workflow/create-proposal/README.md): 3 outstanding questions
- [chat/workflow/create-feature](chat/workflow/create-feature/README.md): 3 outstanding questions
- [chat/workflow/raise-issue](chat/workflow/raise-issue/README.md): 3 outstanding questions
- [chat/workflow/tweak-document](chat/workflow/tweak-document/README.md): 3 outstanding questions
- [github-app](github-app/README.md): 4 outstanding questions
- [onboarding](onboarding/README.md): 5 outstanding questions
- [sandbox](sandbox/README.md): 5 outstanding questions
- [state-store](state-store/README.md): 4 outstanding questions
- [acceptance-criteria](acceptance-criteria/README.md): 4 outstanding questions
- [testing-framework](testing-framework/README.md): 3 outstanding questions
- [testing-framework/test-scenario](testing-framework/test-scenario/README.md): 4 outstanding questions
- [testing-framework/test-runner](testing-framework/test-runner/README.md): 5 outstanding questions
- [ui](ui/README.md): 5 outstanding questions
- [ui/web-app](ui/web-app/README.md): 5 outstanding questions
- [ui/tui](ui/tui/README.md): 5 outstanding questions
