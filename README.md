# Synchestra

**Synchestra** — as in *synch*ronized *orchestra* — is a modular, opinionated platform for orchestrating multi-platform AI agents asynchronously. It manages the **I/O** of AI-driven development: the **inputs** (prompts, specifications, task queues) and the **outputs** (code, documents, artifacts) — keeping token usage minimal, output velocity high, and humans in the loop.

## Why Synchestra Exists

AI agents are getting powerful. Running a single agent on a single task works. But real work isn't a single task — it's a tree of tasks, features, sub-features, dependencies, and decisions that unfold over time.

The moment you try to coordinate multiple agents — or even one agent across multiple sessions — you hit the same problems:

- **Context is expensive.** Every time an agent starts, it needs to understand what's going on. Loading the full project context burns tokens and loses focus. Most of that context is irrelevant to the current task.
- **State is scattered.** What's been done? What's left? What's blocked? The answers live in chat logs, git history, your head, and nowhere canonical.
- **Agents can't find each other's work.** Agent A finishes a sub-task. Agent B needs the output. Without a shared protocol, you're the glue — copying context between sessions, platforms, and tools.
- **Humans can't see what's happening.** When agents work asynchronously across platforms (Claude Code, Cursor, GPT, custom scripts), there's no dashboard, no progress bar, no single place to look and understand: *where are we?*

Existing tools solve the *agent runtime* problem well — how to call tools, chain reasoning, manage prompts. They haven't solved the *coordination* problem: how do agents share state, minimize redundant work, and keep humans in the loop?

Synchestra is that coordination layer.

## What Synchestra Is

Synchestra is a **mental framework, a set of tools, and a runtime** that turns a git repository into a coordination protocol for AI agents.

At its core, Synchestra is a chain of small, automatable steps and background checks that anyone *could* do manually — but nobody has time to do consistently. Think of it as a disciplined workflow engine tuned to one mission: envisioning, planning, and managing the development of software projects with AI agents.

### The tools

- **CLI and MCP server.** Agents call Synchestra to create, search, validate, and update project state. Humans use the same CLI for manual operations. See [CLI spec](spec/features/cli/README.md).
- **[Skills.](#skills)** Pre-built, agent-ready instructions that wrap CLI commands with trigger conditions, parameters, and error handling. Any agent platform can use them — with or without the full Synchestra runtime.
- **Daemon.** Runs as a background process that spawns agents when new tasks are queued, manages headless agent sessions in highly focused mode, and runs pre/post micro-task chains around each prompt.
- **Web UI and HTTP API.** Remote access to everything — edit projects, queue tasks from your phone, monitor progress. Users authenticate via GitHub OAuth or Firebase; their identity is used to sign prompt commits and co-author output artifact commits.
- **Git hooks and CI guards.** Pre-commit, pre-push, and GitHub Actions workflows validate the structure and consistency of Synchestra project files on every change.

### The key properties

- **Hierarchical and nested.** Features have sub-features. Tasks have sub-tasks. The directory tree mirrors the work breakdown structure. This isn't a flat backlog — it's a tree that reflects how work actually decomposes.
- **Modular but opinionated.** Modules are flexible and composable, but they communicate through enforced naming conventions at the top level (e.g., tasks live in `tasks/`) with configurable details (e.g., max nesting depth, source code location). A `README.md` in every directory. A predictable structure that agents can navigate without custom tooling.
- **Multi-platform.** Agents running in Claude Code, Cursor, Windsurf, GPT, or a custom script can all participate. The shared state is the repo, not a proprietary runtime.
- **Token-efficient by design.** Not just structurally — Synchestra actively minimizes token usage through automatic context generation, model selection, and micro-task decomposition (see [How Token Efficiency Works](#how-token-efficiency-works)).
- **Async-first.** Agents work independently and asynchronously. Coordination happens through the repo state, not real-time messaging.

## Repository Structure

A Synchestra-managed repository follows this structure:

```
repo/
  README.md                          # Repository overview

  spec/                              # Product specifications (configurable per project)
    features/
      feature-1/
        README.md                    # Feature description, acceptance criteria
        sub-feature-1/
          README.md
        sub-feature-2/
          README.md

  docs/                              # Product documentation (configurable per project)
    ...

  synchestra/                        # Orchestration layer (hardcoded name)
    projects/
      my-project/
        synchestra-project.yaml      # Project configuration
        tasks/                       # Task queue
          task-1/
            README.md                # Task description, status, assignment
            subtask-1/
              README.md
            subtask-2/
              README.md
          task-2/
            README.md
```

`spec/` and `docs/` live at the repository root — they are the product's specification and documentation. `synchestra/` is the orchestration layer that manages projects, each with its own task tree. The locations of `spec/` and `docs/` are configurable per project via [`synchestra-project.yaml`](spec/project-definition/README.md).

**Every directory has a `README.md`.** This is the atomic unit of Synchestra. Each README contains the context an agent needs to understand that node: what it is, what's expected, what's done, what's blocked — and what questions remain open.

**The directory tree is the work breakdown structure.** Nesting means decomposition. A feature directory contains its sub-features. A task directory contains its sub-tasks. The hierarchy is both organizational and navigational.

**Naming conventions are the API.** Agents looking for work check `synchestra/projects/{project}/tasks/`. Agents needing requirements check `spec/features/`. No registration, no discovery protocol — just filesystem semantics enforced by schema validation.

**Everything is human-readable text.** State is stored as YAML, JSON, or Markdown. Task status lives in the task's parent document alongside a list of sub-tasks and their statuses. Agents read and update state through the [Synchestra CLI](spec/features/cli/README.md), which validates changes against the project schema.

## Storage: Git as a Database

Synchestra uses [inGitDB](https://ingitdb.com) as its storage engine. inGitDB treats the git repository as a structured database with schema validation, enforcing the hierarchical data model that Synchestra depends on.

This means:
- **No external database.** The repo is the database. Git history is the audit trail.
- **Schema-enforced structure.** The inGitDB schema validates directory layout, required fields, naming conventions, and relationships between documents.
- **Consistency guardrails at every layer.** Both the inGitDB CLI and Synchestra CLI run on git pre-commit and pre-push hooks, and inside GitHub Actions workflows. Invalid changes are rejected before they reach the repository.

## How Token Efficiency Works

Token efficiency isn't just about loading fewer files. Synchestra optimizes at multiple levels:

**Minimal context generation.** For each micro-task, Synchestra automatically generates the minimum context an agent needs — the task description, the parent chain for broader context, sibling tasks for awareness of parallel work, and outstanding questions from prior attempts. Everything else stays unloaded.

**Smart model selection.** Not every task needs the most powerful (and expensive) model. Synchestra can select the minimal viable model for each task — either by configuration rules or by using a smaller model to assess task complexity before routing to the right tier.

**Micro-task chains.** Because Synchestra knows the workflow structure, it can run configured chains of specialized micro-tasks before and after processing a user's prompt. Some run sequentially (e.g., validation before submission), others in the background (e.g., updating cross-reference links, running consistency checks).

**Persistent outstanding questions.** Every document maintains an "Outstanding questions" section. When a task is restarted or a new agent picks up related work, it inherits awareness of known pitfalls and pending decisions — avoiding wasted tokens rediscovering known issues. Questions can be linked to sub-tasks; when the sub-task completes, a specialized sub-agent can automatically resolve and remove the question.

## Concurrent Work and Conflict Resolution

Multiple agents working on the same project will inevitably compete for tasks and touch shared files. Synchestra handles this with a layered approach:

### Prevention: claim-and-push protocol

Synchestra's philosophy is **commit often**. When an agent starts work, it must:

1. Claim an unclaimed task by updating its status to "claimed/wip"
2. Commit and push immediately

If the push fails due to a merge conflict, another agent already claimed the task. The agent moves on to the next available task or exits. This is standard distributed locking — implemented through git, requiring zero additional infrastructure. The [synchestra-claim-task](skills/synchestra-claim-task/README.md) skill handles this entire flow for agents automatically.

### Resolution: AI-powered merge handling

When conflicts do occur (e.g., two agents working on different tasks update the same document section), Synchestra launches a specialized AI sub-agent that:

1. Analyzes the merge conflict
2. Merges changes automatically if possible
3. Flags the issue and queues it for rework or human resolution if not

## Skills

Synchestra ships with a library of [skills](skills/README.md) — focused, self-contained instructions that teach AI agents how to perform specific Synchestra operations. Each skill wraps a single [CLI command](spec/features/cli/README.md) and tells the agent exactly when to use it, what parameters to pass, and how to handle every exit code.

**Skills work with any orchestrator.** While skills are designed for the integrated Synchestra workflow, they don't require it. Any agent platform that supports custom instructions — Claude Code, Cursor, Windsurf, GPT, custom scripts — can load Synchestra skills and use them independently. An agent doesn't need the Synchestra daemon, web UI, or even a Synchestra-managed project to benefit from skills. If the agent has access to the CLI, it can use the skills.

This makes skills the lowest-friction entry point to Synchestra: add a few skills to your agent's configuration and it gains structured task management, even if the rest of your workflow is entirely custom.

### Available skills

| Skill | What it does |
|---|---|
| [synchestra-task-create](skills/synchestra-task-create/README.md) | Create a new task |
| [synchestra-task-enqueue](skills/synchestra-task-enqueue/README.md) | Move a task from planning to queued |
| [synchestra-claim-task](skills/synchestra-claim-task/README.md) | Claim a task before starting work |
| [synchestra-task-start](skills/synchestra-task-start/README.md) | Begin work on a claimed task |
| [synchestra-task-status](skills/synchestra-task-status/README.md) | Query or update task status |
| [synchestra-task-complete](skills/synchestra-task-complete/README.md) | Mark a task as completed |
| [synchestra-task-fail](skills/synchestra-task-fail/README.md) | Mark a task as failed |
| [synchestra-task-block](skills/synchestra-task-block/README.md) | Mark a task as blocked |
| [synchestra-task-unblock](skills/synchestra-task-unblock/README.md) | Resume a blocked task |
| [synchestra-task-release](skills/synchestra-task-release/README.md) | Release a claimed task back to the queue |
| [synchestra-task-abort](skills/synchestra-task-abort/README.md) | Request abortion of a task |
| [synchestra-task-aborted](skills/synchestra-task-aborted/README.md) | Report a task has been aborted |
| [synchestra-task-list](skills/synchestra-task-list/README.md) | List tasks with filtering |
| [synchestra-task-info](skills/synchestra-task-info/README.md) | Show full task details and context |

See the [agent-skills feature spec](spec/features/agent-skills/README.md) for design principles — one skill per action, consistent exit codes, and why skills wrap the CLI rather than replacing it.

## Multi-Repository Projects

Synchestra adapts to how your project is organized:

- **Single repo.** Synchestra lives alongside your code in the same repository. Simple and self-contained.
- **Multi-repo projects.** If your project spans multiple repositories (frontend, backend, infrastructure), create a dedicated Synchestra repository for the orchestration layer. This repo holds the specs, tasks, and project state, and can create and update files in your project repositories. The branching strategy and cross-repo synchronization are defined in a dedicated specification file.
- **Multiple projects.** For developers or teams working across multiple projects in parallel, a dedicated Synchestra org provides a single control plane across all of them.

## How It's Different

| | Traditional orchestrators | Synchestra |
|---|---|---|
| State storage | Database / API server | Git repository via inGitDB |
| Agent integration | SDK / API client | CLI, MCP, or direct file access |
| Infrastructure | Server, database, networking | Git + single binary |
| Context loading | Full project dump or custom retrieval | Auto-generated minimal context per task |
| Multi-platform | Locked to one runtime | Any tool that can read/write files |
| Coordination | Real-time messaging | Async via repo state + claim-and-push |
| Audit trail | Event log in database | Git history |
| Validation | Application-level checks | Schema-enforced at commit time |

### Compared to agent frameworks (LangChain, CrewAI, AutoGen)

These are runtimes — they execute agents, manage prompts, and chain tool calls. Synchestra doesn't replace them. It sits beneath them as the shared state and coordination layer. Your CrewAI agents and your Claude Code session can coordinate through the same repo. Load [Synchestra skills](#skills) into any of these frameworks and agents gain structured task claiming, status reporting, and conflict-safe coordination without changing their runtime.

### Compared to project management tools (Linear, Jira)

These track work for humans. Synchestra tracks work for agents *and* humans. The directory structure is both machine-navigable and human-readable. An agent doesn't need an API client to check task status — but it gets validation and consistency guarantees when it uses one.

### Compared to CI/CD systems

CI/CD pipelines are linear and event-driven. Synchestra workflows are hierarchical and async. A task can spawn sub-tasks dynamically. Progress is visible at every level of the tree.

## Fair Questions

**"Isn't this just a well-organized repo?"**

You can absolutely do what Synchestra does manually. You could also deploy software without CI/CD, manage infrastructure without Terraform, and track bugs without an issue tracker. Synchestra is the automation and discipline layer that makes the "well-organized repo" approach sustainable. It's the chain of small steps and background checks that anyone could do but nobody has time to do consistently — schema validation, context generation, model selection, conflict resolution, cross-reference updates, progress tracking. The value isn't in any single convention. It's in the system that enforces and automates all of them together.

**"Naming conventions are fragile. What happens when they break?"**

They don't break silently. Synchestra enforces conventions at multiple checkpoints: the CLI validates on every operation, pre-commit hooks catch problems before they enter the repo, pre-push hooks catch anything that slipped through, and GitHub Actions provide a final safety net. The high-level structure is enforced (tasks go in `tasks/`), while the details are configurable (max nesting depth, source code paths, custom modules). inGitDB's schema validation ensures that the structural invariants hold at every commit.

**"How does async coordination work without conflicts?"**

Through the same mechanism distributed systems have used for decades: optimistic locking. Agents claim tasks by committing a status change and pushing. If the push fails, someone else got there first. For the remaining edge cases — two agents editing different parts of the same file — Synchestra provides AI-powered merge resolution that either handles it automatically or escalates to a human. The "commit often" philosophy minimizes the window for conflicts in the first place.

**"Does this actually scale beyond a solo developer?"**

Synchestra started small — one developer, a few projects, a $10/month VM. But the architecture scales naturally: git already handles distributed collaboration, inGitDB's schema validation works regardless of team size, and the claim-and-push protocol handles concurrency without a central coordinator. The same conventions that keep a solo developer organized keep a team aligned.

## Features

Core features driving Synchestra's development:

| Feature | Status | Description |
|---|---|---|
| [Micro-tasks](spec/features/micro-tasks/README.md) | Conceptual | Pre/post prompt micro-task chains and background automation |
| [Cross-repo sync](spec/features/cross-repo-sync/README.md) | Conceptual | Cross-repository branching, task coordination, and merge strategy |
| [Model selection](spec/features/model-selection/README.md) | Conceptual | Smart model routing based on task complexity and configuration |
| [Conflict resolution](spec/features/conflict-resolution/README.md) | Conceptual | AI-powered merge conflict detection and resolution |
| [Outstanding questions](spec/features/outstanding-questions/README.md) | Conceptual | Question lifecycle management linked to tasks and features |
| [Proposals](spec/features/proposals/README.md) | Conceptual | Non-normative change requests attached to features with review status and optional issue linkage |
| [UI](spec/features/ui/README.md) | Conceptual | Human-facing web and terminal interfaces for projects, features, tasks, proposals, and workers |
| [Claim-and-push](spec/features/claim-and-push/README.md) | Conceptual | Distributed task claiming via git push-based optimistic locking |
| [Agent skills](spec/features/agent-skills/README.md) | In Progress | Focused skills that teach AI agents to use Synchestra |
| [CLI](spec/features/cli/README.md) | In Progress | The `synchestra` command-line interface |

See [feature specifications](spec/features/README.md) for detailed specs and dependency graph.

## Getting Started

**Try it instantly:**
- **Fork a demo project** or use a Synchestra template repository to explore the structure and conventions locally.
- **Sign in at [synchestra.io](https://synchestra.io)** with GitHub OAuth, choose the repo(s) you want to orchestrate, answer a few setup questions, and you're running.

**For full control:**
Install Synchestra on your own VM for persistent daemon mode, headless agent management, and full CLI access. For evaluation, Synchestra can provide a demo VM, or you can run it entirely within a GitHub Actions workflow.

## Dogfooding

Synchestra's own development is managed by Synchestra. We build the tool with the tool — which means every rough edge gets felt immediately and fixed quickly.

**Synchestra-managed projects:**

- [synchestra-project](https://github.com/synchestra-io/synchestra-project) — Synchestra's own development project (specs, tasks, project state)
- [synchestra-go](https://github.com/synchestra-io/synchestra-go) — CLI, daemon, server, HTTP API, and task runner (Go)
- [synchestra-app](https://github.com/synchestra-io/synchestra-app) — Web UI frontend (TypeScript, Angular, Ionic)

## Current Status

Synchestra is in active development. The conventions, module structure, and CLI are being built and refined through daily use on real projects.

## Outstanding Questions

- What is the full lifecycle of a cross-repo task — from branch reservation through integration testing to merge? (Early vision exists; dedicated specification pending. See [Cross-Repo Sync](spec/features/cross-repo-sync/README.md).)
- What is the configuration format for micro-task chains? (Conceptual stage; GitHub Actions-inspired YAML being explored. See [Micro-Tasks](spec/features/micro-tasks/README.md).)
- How does Synchestra interact with agent platform settings when it's not the direct model caller? (Configurable: Synchestra can decide, or the user can override via UI/CLI/API. Hints or arguments are passed to the underlying platform.)

### Children with outstanding questions:

- [spec/](spec/README.md)
  - [features/](spec/features/README.md): 3 outstanding questions
    - [micro-tasks](spec/features/micro-tasks/README.md): 4 outstanding questions
    - [cross-repo-sync](spec/features/cross-repo-sync/README.md): 4 outstanding questions
    - [model-selection](spec/features/model-selection/README.md): 4 outstanding questions
    - [conflict-resolution](spec/features/conflict-resolution/README.md): 3 outstanding questions
    - [outstanding-questions](spec/features/outstanding-questions/README.md): 3 outstanding questions
    - [claim-and-push](spec/features/claim-and-push/README.md): 3 outstanding questions
    - [agent-skills](spec/features/agent-skills/README.md): 3 outstanding questions
    - [cli](spec/features/cli/README.md): 3 outstanding questions
  - [project-definition/](spec/project-definition/README.md): 2 outstanding questions

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## Links

- [Vision](docs/vision.md)
- [Roadmap](docs/roadmap.md)
- [Features](spec/features/README.md)
- [Skills](skills/README.md)
- [CLI Spec](spec/features/cli/README.md)
- [Self-Hosting](docs/self-hosting.md)
- [inGitDB](https://ingitdb.com)
