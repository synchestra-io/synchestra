# Features: Synchestra

Feature specifications for the Synchestra project, managed by Synchestra.

## Specification Format

Synchestra is built on **[SpecScore](https://github.com/specscore/specscore)** — an open-source specification framework. The following specification-format features are defined in SpecScore:

- [Feature](https://github.com/specscore/specscore/blob/main/spec/features/feature/README.md) — Feature structure, metadata, lifecycle, conventions
- [Acceptance Criteria](https://github.com/specscore/specscore/blob/main/spec/features/acceptance-criteria/README.md) — AC format and conventions
- [Source References](https://github.com/specscore/specscore/blob/main/spec/features/source-references/README.md) — Code-to-spec traceability
- [Plan](https://github.com/specscore/specscore/blob/main/spec/features/plan/README.md) — Planning document format
- [Task](https://github.com/specscore/specscore/blob/main/spec/features/task/README.md) — Discrete units of work within a plan
- [Repo Config](https://github.com/specscore/specscore/blob/main/spec/features/repo-config/README.md) — Project configuration

The features below are Synchestra-specific — they define orchestration, coordination, and platform capabilities built on top of SpecScore.

## Index

| Feature | Status | Description |
|---|---|---|
| [Micro-Tasks](micro-tasks/README.md) | Conceptual | Micro-tasks are small, automated steps that Synchestra runs before, after, or in the background relative to a user's prompt. They handle the routine work that keeps a project consistent — formatting, validation, cross-reference updates, link checks — without burning tokens from the main task's context window. |
| [Cross-Repo Sync](cross-repo-sync/README.md) | Conceptual | When a task requires changes across multiple repositories (e.g., updating an API endpoint in the backend and consuming it in the frontend), Synchestra coordinates the work by decomposing the task, reserving branch names, and managing the integration lifecycle across repos. |
| [Model Selection](model-selection/README.md) | Conceptual | Not every task needs the most powerful (and expensive) model. Synchestra routes tasks to the minimal viable model — either through explicit configuration, rule-based routing, or dynamic complexity assessment using a smaller model as a classifier. |
| [Conflict Resolution](conflict-resolution/README.md) | Conceptual | When git merge conflicts occur between concurrent agent operations, Synchestra launches a specialized AI sub-agent to analyze, resolve, or escalate the conflict — reducing human intervention for the mechanical cases while preserving human judgment for the ambiguous ones. |
| [Outstanding Questions](outstanding-questions/README.md) | Conceptual | Every document in Synchestra maintains an "Outstanding questions" section. Questions can be linked to tasks; when the task completes, questions can be automatically resolved. Recently resolved questions remain visible briefly for context before being archived. |
| [UI](ui/README.md) | Conceptual | The UI feature defines Synchestra's shared information architecture for human-facing interfaces — the screens, navigation, and workflows that both the Hub and the terminal UI implement. This document is the single source of truth for what the UI shows and what actions it supports; the subfeature specs ([hub](https://github.com/synchestra-io/synchestra/blob/main/spec/features/ui/hub/README.md), [tui](https://github.com/synchestra-io/synchestra/blob/main/spec/features/ui/tui/README.md)) define how each surface delivers it. |
| [Task Status Board](task-status-board/README.md) | Conceptual | A markdown table embedded in task directory READMEs that provides at-a-glance visibility into task status, ownership, and progress. The board is the source of truth for task state within a Synchestra project. |
| [Agent Skills](agent-skills/README.md) | In Progress | A set of resource-level skills that AI agents use to interact with Synchestra — one skill per CLI resource group (`task`, `feature`, `runner`, `session`, …) with per-verb instructions loaded on demand via Claude Code's progressive-disclosure mechanism. Skills expose *when* to call the CLI, *what* to run, and *how* to interpret results, while keeping the slash-menu surface scannable. |
| [CLI](cli/README.md) | In Progress | The Synchestra CLI (`synchestra`) is the primary interface for agents and humans to interact with Synchestra-managed projects. It validates inputs, enforces state transitions, and handles the git commit-and-push mechanics so callers don't have to. |
| [Chat](chat/README.md) | Conceptual | A chat is a server-managed, goal-oriented conversation between a human user and an AI agent. It is the implementation layer behind user-facing actions such as "Create a Proposal," "Raise an Issue," or "Tweak Document." Users never interact with chats directly — they interact with [workflows](https://github.com/synchestra-io/synchestra/blob/main/spec/features/chat/workflow/README.md) that use chats under the hood. |
| [Global Configuration](global-config/README.md) | Conceptual | The global Synchestra configuration file (`~/.synchestra.yaml`) stores user-level settings that apply across all projects and CLI invocations. It is the single source of truth for machine-local preferences such as where repositories are stored on disk. |
| [REST API](api/README.md) | Unknown | REST API exposing Synchestra operations over HTTP |
| [GitHub App](github-app/README.md) | Conceptual | The Synchestra GitHub App is a GitHub-registered application that Synchestra installs on users' organizations and repositories to receive webhook notifications and perform authenticated operations. It is the bridge between GitHub-hosted repositories and Synchestra's coordination layer. |
| [Onboarding](onboarding/README.md) | Conceptual | Onboarding is a guided wizard — delivered through both the Hub and the CLI — that walks new users through their first Synchestra project setup. It offers two paths: connecting real repositories with a GitHub App installation and AI-powered analysis, or launching a pre-built demo project to explore Synchestra without committing any infrastructure. |
| [Sandbox Feature](sandbox/README.md) | Unknown | Isolated Docker container environments per project for executing user-initiated commands from the chat interface |
| [Embedded State](embedded-state/README.md) | Conceptual | Embedded state allows Synchestra to manage coordination state inside an existing repository using a git worktree on an orphan branch — no separate state repo required. This provides a zero-friction onboarding path: any git repository can add Synchestra task management with a single `synchestra project init` command. |
| [State Store](state-store/README.md) | Conceptual | The state store is the abstraction layer for all Synchestra project coordination state. It defines a composable, hierarchical Go interface (`state.Store`) that formally specifies every operation the system can perform on project state — tasks, artifacts, chat, and project configuration. |
| [Testing Framework](testing-framework/README.md) | Conceptual | Synchestra's testing framework turns specifications into executable verification — without leaving markdown. Acceptance criteria define what "correct" means for each feature. Test scenarios compose those criteria into multi-step workflows. The runner executes everything and reports results. |
| [LSP for Specifications](lsp/README.md) | Conceptual | A [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) (LSP) server that exposes Synchestra's specification navigation capabilities to IDEs and editors. The `synchestra feature` CLI commands already form an LSP-like semantic layer — this feature would wrap them in the standard LSP protocol, giving humans live IDE integration for specification files. |
| [Bots](bots/README.md) | Conceptual | Messenger bots that serve as conversational interfaces to Synchestra. Three distinct kinds of bots operate at different layers of the platform, each with its own lifecycle, hosting model, and audience. |
| [Stakeholder](stakeholder/README.md) | Conceptual | A stakeholder is any entity — human or AI agent — that participates in workflow decisions within a Synchestra project. Stakeholders review specifications, approve plans, review code, and provide input when agents need guidance. They are the "who decides" layer that connects Synchestra's workflow artifacts (features, plans, tasks) to the people and agents responsible for governing them. |
| [Runner](runner/README.md) | Conceptual | Remote hosts, VMs, and cloud environments where AI agents execute sessions and claim tasks. A runner is a registered compute endpoint that Synchestra connects to for remote agent interaction. Users interact with agents on runners through sessions — ephemeral, chat-like conversations surfaced in the web UI. |
| [Host-Hub Authentication](host-auth/README.md) | Conceptual | Mutual authentication between runner hosts and the Synchestra Hub. Hosts authenticate to the Hub using short-lived access tokens derived from a permanent registration token. The Hub authenticates to hosts by signing requests with a private key whose public counterpart is published at a well-known URL. |
| [User Authentication](user-authentication/README.md) | Draft | The host authenticates inbound HTTP requests by validating OIDC ID tokens locally against one or more configured issuers. `hub.synchestra.io` is the default issuer; operators may configure additional OIDC-compliant providers (Auth0, Keycloak, Okta, Google, Microsoft, a self-hosted IdP, etc.) to run alongside hub or replace it. Authorization is project-scoped: tokens issued by hub carry a `synchestra.projects` claim that is intersected with the host's `served_projects`; tokens issued by any non-hub issuer grant access to all of the host's `served_projects` once authenticated. |
| [Project](project/README.md) | Draft | A Synchestra project is a named unit of work identified canonically by its primary repository reference (e.g., `github.com/synchestra-io/synchestra`). Projects have a two-phase lifecycle — pre-repo (Firestore-authoritative membership) and post-repo (yaml-authoritative membership) — that enables both cloud-mediated and bypass-auth session authorization. Membership semantics, authorization primitives, yaml schema, and caching are specified in the [members](https://github.com/synchestra-io/synchestra/blob/main/spec/features/project/members/README.md) sub-feature. |
| [Channels](channels/README.md) | Conceptual | Channels provide bidirectional, real-time messaging between users and Claude Code instances running inside sandbox containers on remote runners. Users interact from the Hub (browser) or Telegram; messages flow through the Synchestra cloud layer to the runner host, into the container, and reach Claude Code via a local MCP channel server. Replies flow the reverse path. |
| [Routines](routines/README.md) | draft | Cross-platform scheduled and triggered agent workflows. A routine is a spec-native, runtime-portable unit of recurring or event-driven work that runs on a chosen runner and produces reviewable git artifacts. |
| [Plugins](plugins/README.md) | Deferred | The eventual Synchestra plugin SPI is expected to follow a single, simple shape: **plugins contribute namespaced commands.** The project's configuration then composes those commands into either workflow-event hooks (Spec Kit-style: *after the spec is written, run X*) or [micro-task](../micro-tasks/README.md) chain steps (Synchestra-style: *before every prompt, run X*). One plugin shape serves both surfaces. |
| [Agent Coordination](agent-coordination/README.md) | Approved | Tracks efforts, agent runs, worktree claims, audited messages, active/recent views, overlap detection, recovery, and cleanup state across repositories. |
| [Remote Task Dispatch](dispatch/README.md) | Approved | Remote Task Dispatch accepts an ad-hoc repository prompt or a SpecScore Plan/Task target, records it as durable scheduled work, leases it to an eligible registered worker, and returns a reviewable Git branch plus execution evidence. The initial proof uses the existing personal VM and Claude Code, while the contract supports multiple workers, agents, model profiles, and future runtimes. |
| [Synchestra Repo Config](repo-config/README.md) | Approved | Defines `synchestra.yaml`, the single repo-level config file that holds Synchestra-only orchestration metadata. The file lives at the repository root next to `specscore.yaml`. Project identity (title, host, org, repo, repositories) is **read** from `specscore.yaml` — never duplicated. `synchestra.yaml` holds only what SpecScore does not: state-repo location, sync policy, hub registration, runner pinning, and similar orchestration concerns. The schema mirrors SpecScore's `repo-config` pattern: a fixed line-1 schema-pointer comment, a single authoritative root file, no front-matter, an empty file is valid. |
| [State Repo Config](state-repo-config/README.md) | Approved | Defines `synchestra-state.yaml`, the self-identifier file that lives at the root of every Synchestra **state location** — whether that is a dedicated state repository, an orphan branch in the project's main repo (embedded mode), or a Hub-managed state worktree. The file declares which spec repos this state belongs to, distinguishing legitimate state locations from arbitrary directories that happen to look like one. |
| [wb-session-transport](wb-session-transport/README.md) | Approved | Accept typed WB session handoff requests through Synchestra durable runner infrastructure without exposing arbitrary remote shell execution. |

## Feature Summaries

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

The human-facing product surfaces for Synchestra. Defines a shared information architecture (home → project menu → Features / Tasks / Workers) with MVP flows for proposal creation and task creation/enqueueing. Two delivery surfaces: [Synchestra Hub](ui/hub/README.md) — a browser-based management interface at hub.synchestra.io for projects, runners, and tasks — and a [TUI](ui/tui/README.md) delivered through the CLI operating on local git state. Introduces the Workers concept at the UI level; a dedicated workers feature spec is needed before going beyond visibility.


### [Task Status Board](task-status-board/README.md)

A markdown table in task directory READMEs that provides at-a-glance visibility and serves as the source of truth for task state. Agents claim tasks by updating a board row and pushing through optimistic locking (git push-based). Conflicts on the same row indicate a claim collision; the CLI parses diffs by task ID to distinguish collisions from unrelated changes. See the [Claiming a Task](task-status-board/README.md#claiming-a-task-optimistic-locking) section for the full protocol.

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

A guided wizard delivered through both the Hub and the CLI that walks new users through first-time project setup. Offers two paths: "Connect your repositories" (GitHub App installation → spec repo selection → optional code repos → state repo provisioning → bring-your-own AI key → AI-powered repo analysis and scaffolding → project creation) and "Try the demo" (pre-built sample project with example features, tasks, and proposals). The wizard handles infrastructure bootstrapping — creating state repos, generating `synchestra-spec-repo.yaml`, and scaffolding initial feature structures — so users reach a working project in minutes.

### [Sandbox](sandbox/README.md)

Isolated Docker container environments per project for executing user-initiated commands from the chat interface. Each project gets its own persistent container with encrypted credential storage (AES256), user-isolated sessions, and a gRPC agent for host↔container communication. The host is stateless and routes requests; all state, secrets, and execution data remain inside containers.

### [Embedded State](embedded-state/README.md)

Zero-friction onboarding path for Synchestra. Uses a git orphan branch checked out as a worktree to store coordination state inside an existing repository — no separate state repo required. One command (`synchestra project init`) sets up task management in any git repo. Provides the same history isolation as a dedicated state repo (orphan branches share no commits with `main`) and the same optimistic locking protocol. Designed as an on-ramp: projects that outgrow embedded mode can extract state to a dedicated repo later.

### [State Store](state-store/README.md)

The pluggable abstraction layer for all Synchestra project coordination state. Defines a composable, hierarchical Go interface (`state.Store`) with sub-interfaces for tasks (`TaskStore`), chat (`ChatStore`), and project configuration (`ProjectStore`). Navigated like CLI subcommands — `store.Task().Claim(ctx, ...)` — keeping each interface focused and discoverable. The default git-backed implementation (`gitstore`) maps to the existing state repository design; future backends (SQLite, PostgreSQL, cloud databases) satisfy the same interface.

### [Testing Framework](testing-framework/README.md)

Turns specifications into executable verification — without leaving markdown. Composes acceptance criteria into multi-step test workflows that read as documentation and execute as test suites. The full specification — including the [test-scenario](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/test-scenario/) format and [test-runner](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/test-runner/) engine — lives in the [synchestra-io/rehearse](https://github.com/synchestra-io/rehearse/blob/main/spec/features/testing-framework/) repository. Synchestra integrates Rehearse as its testing framework.

### [LSP for Specifications](lsp/README.md)

A Language Server Protocol server that wraps the same Go packages powering the `synchestra feature` CLI commands. Gives humans live IDE integration: document symbols from `feature info`, go-to-definition from `feature deps`, find-references from `feature refs`, diagnostics from `spec validate`, and autocomplete for feature IDs. The CLI serves agents; the LSP serves humans editing specs in VS Code, Neovim, JetBrains, or Emacs. A later-phase feature — depends on the CLI packages being implemented first.

### [Bots](bots/README.md)

Messenger bots that serve as conversational interfaces to Synchestra. Three kinds are recognized: [SynchestraBot](bots/synchestra-bot/README.md) (platform-operated, embedded in the server, Telegram-first), in-container bots (user-defined, running inside sandbox containers), and host-level bots (user-defined, running on the host machine). Only SynchestraBot is specified at this time. It provides project management commands, sandbox container control, prompt relay to in-container agents, and bidirectional notifications — built on [bots-go-framework/bots-fw](https://github.com/bots-go-framework/bots-fw) for platform-agnostic messenger support.

### [Stakeholder](stakeholder/README.md)

Humans and AI agents that participate in workflow decisions. Stakeholders are identified by inline string references (`alex@github`, `agent-x:model=opus`), assigned to roles (`code-reviewer`, `spec-approver`) that resolve hierarchically through the feature tree using `add`/`remove` overrides at each level. When a workflow hits a decision point — either a built-in gate (plan review, code review) or an agent-initiated blocker — a structured decision task is created with typed options (`pick-one`, `approve-reject`, etc.) and assigned to the resolved stakeholders. Responses are recorded in a per-task audit log. Sub-features cover [roles](stakeholder/role/README.md), [decisions](stakeholder/decision/README.md) (with [options](stakeholder/decision/options/README.md) and [audit](stakeholder/decision/audit/README.md)), [gates](stakeholder/gate/README.md), and [notifications](stakeholder/notification/README.md).

### [Runner](runner/README.md)

Remote hosts, VMs, and cloud environments where AI agents execute sessions and claim tasks. A runner is a registered compute endpoint — users interact with agents on runners through sessions, ephemeral chat-like conversations surfaced in the web UI. Runners provide persistent availability, multi-environment support, and centralized visibility across all registered compute endpoints.

### [Host-Hub Authentication](host-auth/README.md)

Mutual authentication between runner hosts and the Synchestra Hub. Hosts prove identity using a two-tier token model: a permanent registration token (stored on disk) is exchanged for short-lived access tokens (held in memory) with server-dictated TTL. The Hub authenticates to hosts by signing requests with an Ed25519 private key; hosts verify using a public key published at a well-known URL. Registration happens through `synchestra-host hub connect` (interactive device flow) or `synchestra-host hub connect --token {token}` (pre-provisioned). Hosts are managed by one or more users (managers) and are independent of projects.

### [Channels](channels/README.md)

Bidirectional, real-time messaging between users and Claude Code instances running inside sandbox containers on remote runners. Messages flow from the Hub (browser) or Telegram through the Synchestra cloud layer (Cloud Run + Firestore) to runner hosts, into containers via the sandbox agent's gRPC interface, and reach Claude Code through a local MCP channel server implementing the Claude Code channels protocol. Firestore is the source of truth for all messages; Hub subscribes via onSnapshot for real-time delivery. Extends the sandbox agent with session management and messaging RPCs, and ships a Go-based channel MCP server in the container image.

### [Routines](routines/README.md)

Cross-platform scheduled and triggered agent workflows. A routine binds four declarative components — a trigger (cron, git event, task state, or manual), a runtime adapter (Claude Code headless, Copilot CLI, raw LLM API, etc.), a runner target (local, VM, cloud), and a body (prompt, skill, or task reference) — into a versioned spec under `spec/routines/`. Routines are runtime- and compute-portable: the same spec runs on any supported runtime and any registered runner without rewrite. Every run produces a reviewable git artifact (branch, PR, or task update) by default, keeping humans in the loop and making runs first-class citizens of the work graph.

### [Plugins](plugins/README.md)

Plugin SPI for Synchestra and SpecScore — **intentionally deferred** for 2026. Until plugin authors are knocking, extensibility ships into GitHub Spec Kit's existing extension system (via three first-party extensions: `speckit-specscore`, `speckit-synchestra`, `speckit-rehearse`) rather than a parallel one. The intended shape when revisited is V7: plugins contribute namespaced commands; the project composes them into either workflow-event hooks (Spec Kit-style) or [micro-task](micro-tasks/README.md) chain steps (Synchestra-style). One plugin shape, two event surfaces. Trigger conditions and the full rationale live in [`synchestra-marketing/decisions/2026-05-01-plugin-system-strategy.md`](https://github.com/synchestra-io/synchestra-marketing/blob/main/decisions/2026-05-01-plugin-system-strategy.md).

```
# SpecScore features (external): feature, acceptance-criteria, source-references, plan, task, repo-config
# See https://github.com/specscore/specscore

task-status-board ← conflict-resolution
       ↑                ↑
cross-repo-sync ────────┘
       ↑
micro-tasks (independent)
model-selection (independent)
outstanding-questions (independent)
proposals → [specscore:plan] (proposals trigger plans)
[specscore:plan] → task-status-board, cli (plans generate tasks)
chat → [specscore:feature], proposals, [specscore:plan], task-status-board, agent-skills, ui, api
ui → proposals, cli, task-status-board, agent-skills, [specscore:plan], chat
api → cli (api mirrors cli contract)
global-config ← cli (cli reads ~/.synchestra.yaml for repo resolution)
github-app → api (callback endpoint)
onboarding → github-app, [specscore:repo-config], ui, cli, api (orchestrates first-time setup)
sandbox → cli, api (containers execute commands, host routes via API)
bots → sandbox, chat, api, state-store (SynchestraBot relays prompts to containers, routes complex workflows through chat, uses API for operations)
lsp → cli/feature, [specscore:feature] (LSP server reuses CLI feature packages for IDE integration)
state-store → task-status-board (board interface and claim atomicity), chat (chat persistence)
state-store ← cli, api, agent-skills (all consumers of state go through state store)
[specscore:acceptance-criteria] → [specscore:feature] (mandatory AC section), [specscore:plan] (plan ACs can reference feature ACs)
testing-framework → [specscore:acceptance-criteria] (composes ACs into test flows), cli (new test command group), [specscore:feature] (_tests/ directory)
[specscore:source-references] → [specscore:feature], cli, [specscore:repo-config] (annotations link code to spec resources, validated by linter)
stakeholder → task-status-board (decisions are tasks), [specscore:plan] (gates trigger on plan transitions), [specscore:feature] (_config.yaml for role overrides), cli (decision/stakeholder commands), agent-skills (decision-request skill), state-store (DecisionStore)
stakeholder ← chat (workflows create decisions), ui (renders decision options), bots (delivers notifications, accepts responses)
host-auth → runner (prerequisite for runner registration), channels (authenticated host-hub messaging), api (token endpoints, public key endpoint)
channels → runner (host compute layer), sandbox (agent gRPC extensions, container image), api (cloud endpoints), state-store (Firestore persistence)
channels ← ui/hub (browser surface), bots (Telegram surface), chat (sessions may trigger workflows)
```

[SpecScore `feature`](https://github.com/specscore/specscore/blob/main/spec/features/feature/README.md) is the foundational spec-layer concept — proposals, plans, and outstanding questions all attach to features.

## Diagram Conventions

All diagrams in feature specifications should use **mermaid syntax** instead of ASCII art. Mermaid provides better clarity, GitHub rendering support, and maintainability.
`task-status-board` is foundational for execution — it provides the claiming mechanism (optimistic locking) and status visibility.
[SpecScore `plan`](https://github.com/specscore/specscore/blob/main/spec/features/plan/README.md) bridges the spec-to-execution gap — proposals and feature specs flow through plans to become tasks. The [task](https://github.com/specscore/specscore/blob/main/spec/features/task/README.md) feature defines the methodology-level task concept that Synchestra implements.

## Open Questions

- Are there features missing from this list that are already described in `docs/features/` but not yet tracked here?
- **Suggested build order:** task-status-board first (foundational), then outstanding-questions and model-selection (independent, high value), then proposals, then UI once CLI and proposal flows are ready enough to expose, then conflict-resolution, then micro-tasks and cross-repo-sync. Does this align with project priorities?

### Features with outstanding questions:

- [micro-tasks](micro-tasks/README.md): 4 outstanding questions
- [cross-repo-sync](cross-repo-sync/README.md): 4 outstanding questions
- [model-selection](model-selection/README.md): 4 outstanding questions
- [conflict-resolution](conflict-resolution/README.md): 3 outstanding questions
- [outstanding-questions](outstanding-questions/README.md): 3 outstanding questions
- [task-status-board](task-status-board/README.md): 4 outstanding questions
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
- [testing-framework](testing-framework/README.md): 3 outstanding questions
- [ui](ui/README.md): 5 outstanding questions
- [ui/hub](ui/hub/README.md): 7 outstanding questions
- [ui/tui](ui/tui/README.md): 5 outstanding questions
- [bots](bots/README.md): 2 outstanding questions
- [bots/synchestra-bot](bots/synchestra-bot/README.md): 5 outstanding questions
- [lsp](lsp/README.md): 5 outstanding questions
- [stakeholder](stakeholder/README.md): 5 outstanding questions
- [stakeholder/role](stakeholder/role/README.md): 4 outstanding questions
- [stakeholder/decision](stakeholder/decision/README.md): 4 outstanding questions
- [stakeholder/decision/options](stakeholder/decision/options/README.md): 4 outstanding questions
- [stakeholder/decision/audit](stakeholder/decision/audit/README.md): 4 outstanding questions
- [stakeholder/gate](stakeholder/gate/README.md): 4 outstanding questions
- [stakeholder/notification](stakeholder/notification/README.md): 4 outstanding questions
- [host-auth](host-auth/README.md): 3 outstanding questions
- [channels](channels/README.md): 8 outstanding questions

---
*This document follows the https://specscore.md/features-index-specification*
