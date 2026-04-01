# Synchestra Documentation

Welcome to the Synchestra docs. Whether you're an AI agent looking for an endpoint to call or a human trying to understand what your agents are up to — this is where you start.

---

## Contents

### Product

- [What is Synchestra?](../README.md) — Root README with overview, architecture, and quick start
- [Vision](vision.md) — Long-term product vision
- [Features](features/README.md) — Deep dives into each core feature
- [Roadmap](roadmap.md) — What's coming and when

### Integration

- [CLI Reference](cli/README.md) — Every `synchestra` command documented
- [API Reference](api/README.md) — Every HTTP endpoint documented with examples

### Operations

- [Self-Hosting](self-hosting.md) — Running Synchestra on your own infrastructure

### Internal

- [Superpowers](superpowers/) — Design specs and internal decision records. Implementation plans for features are stored as formal plans in `spec/plans/` and follow the approval workflow with snapshots tracking history.

---

## Core Concepts

| Concept | Description |
|---|---|
| **Human** | A person who monitors and steers agent execution |
| **Org** | A group of humans; humans can belong to multiple orgs |
| **Agent** | A specialized AI that performs a defined set of tasks |
| **Project** | A container for related work; a human can run many at once |
| **Repo** | A code repository linked to a project; shareable across projects |
| **Task** | A unit of work with clear acceptance criteria; supports sub-tasks |
| **Skill** | A capability definition that agents declare and tasks can require |
| **Rule** | A constraint or instruction attached to a human, org, project, or repo |
| **Token** | An API credential scoped to specific operations |

---

## Quick Links

**For AI agents:**
- [Create a task](api/tasks.md#create-task)
- [Update task status](api/tasks.md#complete-task)
- [Send a heartbeat](api/agents.md#heartbeat)
- [Append progress log](api/tasks.md#append-log)
- [CLI task commands](cli/task.md)

**For humans:**
- [System status](api/status.md)
- [Task history](api/tasks.md#get-task-history)
- [Human oversight features](features/human-steering.md)
- [Workflow orchestration](features/workflow-orchestration.md)

**For operators:**
- [Self-hosting guide](self-hosting.md)
- [Auth & tokens](cli/auth.md)
- [Server startup](cli/server.md)

---

## Outstanding Questions

None at this time.
