# Synchestra Vision

---

## The Problem

AI pipelines are opaque by default.

You spin up a multi-agent system — orchestrator, coders, reviewers, deployers — point them at a task, and watch the logs scroll. Something is happening. You don't know exactly what, or why, or whether it's about to do something you'd want to stop. By the time a human looks, the mistake has already propagated.

The state of any running agent pipeline is scattered: partially in logs, partially in the model's context window, partially in whatever memory system the framework provides, and partially in your head. There is no canonical truth about what an agent did, is doing, or plans to do next. If two agents are coordinating, each one has its own partial view. If a human wants to intervene, they have nowhere to grab the system.

Existing tools have gotten very good at the *agent runtime* problem — how to call tools, chain reasoning, route between models. They haven't solved the *coordination* problem: how do agents share state, how do humans stay in the loop, and how does an organization maintain visibility and control over systems that operate autonomously at scale?

That's the gap Synchestra fills.

---

## Origin

Synchestra started as a personal tool.

The specific itch: watching a multi-agent pipeline do something — making file changes, running commands, making decisions — with no way to understand what it was doing or intervene without killing the whole process. The frustration wasn't with the agents. It was with the absence of a coordination layer. There was no shared state, no audit trail, no place for a human to grab the wheel.

Every existing option was either a full framework (take it or leave it), a cloud service (give us your data), or a log aggregator (read-only, after the fact). None of them treated coordination and human oversight as first-class problems.

So: build the missing layer. Keep it small, self-hostable, and actually usable from an agent with a single CLI call.

---

## What Synchestra Is

Synchestra is the coordination layer for multi-agent AI systems. It is the single source of truth for task and agent state — the glue between agents, humans, and systems that need to know what's happening and influence what happens next.

It is not a framework. It is not a runtime. It does not care what agent library you use, what model you call, or how you've structured your pipeline. It sits beneath all of that and answers one question clearly: *what is the current state of this system, and who can change it?*

---

## Who It's For

**Developers building multi-agent systems** — You need coordination without building it from scratch. Synchestra gives you task management, agent registration, state synchronization, and human oversight primitives out of the box. One binary, one SQLite database, zero infrastructure dependencies to get started.

**Engineering teams and organizations** — You need visibility into what your agents are doing, audit trails for compliance and debugging, and human oversight at scale. Synchestra stores every state transition, every agent action, every human intervention in a queryable log. When something goes wrong, you know exactly what happened and when.

**Solo builders** — You're running agents locally on your own machine and you don't want to stand up Kubernetes to get visibility into them. Synchestra runs with `synchestra server` and stores everything in SQLite. No accounts, no cloud, no ceremony.

**OSS contributors** — Synchestra is opinionated, minimal, and driven by real usage rather than roadmap ambition. The codebase is small by design. If you've wanted to contribute to infrastructure that AI agents will actually depend on, this is a good place to start.

---

## Core Principles

These are not aspirational. They are constraints that guide every technical and product decision:

1. **Ship working software, not roadmap theatre.** Features ship when they're useful, not when they look good on a slide.

2. **Self-hosted is always free and fully featured.** The self-hosted version will never be crippled, paywalled, or abandoned in favor of SaaS revenue. This is a hard commitment.

3. **Stable API — don't break agent integrations.** Agents are automated. If the API changes in a breaking way, agents break silently. The CLI and REST API are treated as stable contracts.

4. **Minimal footprint.** A single binary, a single database. No Kafka, no Kubernetes required. If you can run a Go binary, you can run Synchestra.

5. **Agent-first design.** Every feature must be usable by an agent without ceremony — one CLI call or one HTTP POST. If an agent has to do three things to accomplish one, the design is wrong.

6. **Human-legible state.** All state is stored in a form that's queryable and readable by humans, not just machines. Agents serve humans. The system should reflect that.

7. **Incrementally adoptable.** Use only what you need. Start with tasks and agents. Add skills, rules, and notifications when you need them. Nothing is mandatory except the parts you actually use.

---

## What Synchestra Is Not

**Not an agent runtime or framework.** Synchestra does not execute your agents, manage their prompts, or orchestrate their reasoning. Use LangChain, Autogen, CrewAI, raw API calls — whatever you prefer. Synchestra is the coordination layer beneath them.

**Not a workflow DSL or visual builder.** There is no YAML pipeline definition, no drag-and-drop workflow editor, no proprietary execution model. Workflows are code. Tasks and agents compose into pipelines; how that composition works is your code's job, not Synchestra's.

**Not an LLM provider.** Synchestra makes no LLM calls. It has no prompts, no model dependencies, no vendor relationship with any AI provider. It is model-agnostic by design.

**Not a replacement for your existing agents.** Synchestra does not replace what you've built — it is the layer between your agents, your humans, and your systems. Plug it in next to what already works.

---

## North Star

In three to five years:

Synchestra is the de facto coordination layer for multi-agent AI systems — the thing you add to a multi-agent pipeline the way you add a database or a message queue. Not because it's the only option, but because it solves a real problem with minimal friction and no lock-in.

Self-hosted remains free forever. The SaaS tier exists for teams who want managed infrastructure and don't want to operate their own server — not as a lever to degrade the self-hosted experience.

MCP is the primary interface for AI-native agents. As the ecosystem matures, agents will increasingly communicate via MCP rather than REST. Synchestra ships a first-class MCP server and treats MCP as a peer interface to the HTTP API, not an afterthought.

Human oversight is a first-class feature, not a compliance checkbox. The tools that give humans real visibility into agent behavior — live status, approval gates, audit trails, steering controls — are core to what Synchestra is, not add-ons bolted on after the fact. Autonomous systems are only trustworthy if the humans responsible for them can see what they're doing.
