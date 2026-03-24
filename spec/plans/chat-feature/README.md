# Plan: Chat feature — high-level roadmap

**Status:** draft
**Features:**
  - [chat](../../features/chat/README.md)
  - [chat/workflow](../../features/chat/workflow/README.md)
**Source type:** feature
**Source:** [Chat feature spec](../../features/chat/)
**Author:** @alex
**Created:** 2026-03-14

## Context

The Chat feature introduces a guided conversational interface that produces Synchestra artifacts (proposals, features, issues, PRs) through AI-assisted workflows. This is a large feature spanning multiple subsystems: server-side chat infrastructure, a workflow engine, individual workflow implementations, web UI, API endpoints, and CLI commands.

This high-level plan decomposes the Chat feature into implementation phases. Each phase has its own detailed development plan. The phases are designed so that each produces working, testable software independently and builds on the previous phase.

## Acceptance criteria

- All phases are covered by detailed development plans
- Each phase can be implemented and tested independently
- Dependencies between phases are explicit and unidirectional (later phases depend on earlier ones, never the reverse)

## Child Plans

| Order | Plan | Status | Effort | Impact |
|-------|------|--------|--------|--------|
| 1 | [chat-infrastructure](chat-infrastructure/) | draft | L | high |
| 2 | [chat-workflow-engine](chat-workflow-engine/) | draft | M | high |

### Phases without plans yet

Phases 3-5 from the original roadmap do not yet have their own detailed development plans:

- **Phase 3 — Built-in workflows:** Implement the four shipped workflows (Create Proposal, Create Feature, Raise Issue, Tweak Document) with prompt/skill chains, artifact production, and role-based behavior.
- **Phase 4 — Web UI chat interface:** Build the chat UI components (message input, streaming responses, workflow action buttons, artifact preview/editing).
- **Phase 5 — Integration and end-to-end testing:** Full pipeline testing from user action through chat, workflow, and artifact production into Synchestra pipeline entry.

## Risks and open decisions

- **AI prompt quality.** The effectiveness of chat workflows depends heavily on prompt engineering. The prompts/skills for each workflow step need careful design and iteration. Consider allocating time for prompt tuning after initial implementation.
- **Context window limits.** Long conversations with large anchor documents may exceed context limits. The context assembly strategy (first N + last M + summary) needs testing with real-world document sizes.
- **External service integration.** The Raise Issue workflow depends on GitHub Issues API integration. This adds an external dependency and potential failure modes (rate limiting, authentication, API changes).
- **Fast-path complexity.** The fast path (implement during conversation) involves coordinating real-time task creation, agent dispatch, and code generation while maintaining a conversational UX. This is architecturally complex and may need simplification in v1.

## Outstanding Questions

- Should phases 3-5 have their own detailed development plans, or are the step descriptions in this high-level plan sufficient for task generation?
- What is the target technology stack for the server-side components — Go (matching the existing CLI), or a different language better suited for real-time chat (e.g., TypeScript/Node.js)?
- Should the web UI be part of the existing [UI](../../features/ui/README.md) feature's implementation plan, or does it warrant its own plan?
