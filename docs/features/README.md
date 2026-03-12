# Features

Synchestra is built around a small set of focused primitives that compose into a surprisingly powerful coordination layer. Here's what's inside.

---

## Feature Index

| Feature | Description |
|---|---|
| [Agent Coordination](agent-coordination.md) | Register agents, declare skills, route tasks to the right agent |
| [State Synchronization](state-synchronization.md) | Keep task and agent state consistent across distributed agents |
| [Progress Reporting](progress-reporting.md) | Structured logs, status transitions, and history queries |
| [Workflow Orchestration](workflow-orchestration.md) | Multi-step pipelines, sub-tasks, and sequential/parallel execution |
| [Communication Interfaces](communication.md) | CLI, HTTP API, and MCP server —  agents pick their interface |
| [Human Steering](human-steering.md) | Visibility, approval gates, override hooks, and notifications |

---

## How the Features Fit Together

Synchestra's features layer on top of each other. At the base you have **state synchronization** —  the engine that keeps everything consistent. On top of that sit the coordination features: agent management, task routing, and progress reporting. Workflow orchestration composes those primitives into pipelines. Human steering gives you the controls to influence running workflows.

```
Human Steering
     ↑
Workflow Orchestration
     ↑
Agent Coordination + Progress Reporting
     ↑
State Synchronization
     ↑
Storage (SQLite / Postgres)
```

---

## Design Principles

- **Minimal footprint.** A single binary, a single database. No Kafka, no Kubernetes required to get started.
- **Agent-first.** Every feature is designed so agents can use it without ceremony —  one CLI call or one HTTP POST.
- **Human-legible.** All state is stored in a form that's queryable and readable by humans, not just machines.
- **Incrementally adoptable.** Use only what you need. Start with tasks and agents; add rules and skills when you need them.
