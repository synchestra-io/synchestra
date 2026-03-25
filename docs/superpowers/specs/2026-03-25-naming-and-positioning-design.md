# Synchestra Naming and Positioning Design

## Status

Approved (brainstorming session 2026-03-25)

## Problem

Synchestra has two major components:

1. **Spec format, mental model, and CLI** — open-source, git-native, local-first. In active development.
2. **Platform for remote agent execution with WebUI** — conceptual stage with early groundwork.

The current website positions Synchestra as "Not another agent framework" and emphasizes "No server." With the platform component, both claims become dishonest. The duality is a strength but creates a positioning challenge.

## Decisions

### Product Naming

**Synchestra** is the open-source foundation:

- Spec format (features, plans, tasks, stakeholders)
- CLI (linting, discovery, task coordination)
- Git-native state protocol (atomic operations, agent coordination)
- Free and open source

**Synchestra Hub** is the platform layer:

- Remote agent execution (VM, cloud, self-hosted)
- Dispatcher and API
- WebUI (dashboards, visibility, control)
- Commercial product (the monetization/funnel goal)

### Relationship Model

Layered, not peer. Hub builds on top of Synchestra. Hub users always use Synchestra underneath. Synchestra users do not need Hub.

Analogy: Docker (CLI/runtime) and Docker Desktop/Hub. The foundation is just "Synchestra" — no qualifier needed. Only the platform gets a name: "Synchestra Hub."

### Why "Hub"

- Describes the role (central coordination point), not the deployment model
- Works for both server-side (dispatcher/API) and frontend (WebUI/dashboard)
- Fits self-hosted and cloud-managed deployments equally
- Short, easy to say, pairs well: "Synchestra" and "Synchestra Hub"

### Positioning

**Headline** (unchanged): "Every agent knows its part."

**Subtitle** (unchanged): "Spec-driven coordination for AI-assisted development."

**Clarifying line** (new, replaces "Not another agent framework"):
"Define work as specs. Run it anywhere — locally, or on Synchestra Hub."

**Positioning statement**: "Specs first. Then agents."

- Synchestra is the layer above agent runtimes (Claude Code, Copilot, Cursor, custom scripts)
- It defines *what* to do; agents decide *how*
- Not competing with agent frameworks — sitting above them

### Competitive Framing

Drop the negation ("not another X"). Position by architecture layer:

- **Agent runtimes** (Claude Code, Copilot, Cursor, custom scripts) = how work gets done
- **Synchestra** = what work needs doing, and coordination
- **Synchestra Hub** = where and when work runs

### Website Structure

**synchestra.io** — main landing page:

- Leads with specs/CLI value prop (the open-source story)
- Hub introduced in a dedicated section (not buried, not dominant)
- "Specs first. Then agents." as positioning thread

**synchestra.io/hub/** — Hub promo/landing page:

- Focused on remote execution audience
- Assumes reader may not know about specs yet
- Explains the full stack: specs define work, Hub runs it
- Self-hosted and managed options presented
- Marketing and docs only — no interactive app features

**hub.synchestra.io** — Hub WebUI/console (the actual app):

- In-cloud frontend for remote agent execution, dashboards, and project management
- Separate subdomain from marketing pages intentionally: corporate firewalls may block app domains that enable data upload/leakage, but read-only marketing/docs pages on the main domain remain accessible
- Landing pages should reach as many users as possible; the app domain can be allowlisted separately

### Brand Family

| Product | Relationship | Audience entry |
|---------|-------------|----------------|
| **Synchestra** | Core / foundation | Developers, tech leads who want structured agent coordination |
| **Synchestra Hub** | Platform layer on core | Teams wanting remote execution, visibility, scale |
| **Rehearse** | Independent sibling (sub-brand) | Anyone who wants to test with markdown specs |

### Sub-brand Criteria (reaffirmed)

A component earns its own brand when it has: (1) standalone CLI, (2) independent users, (3) own ecosystem, (4) top-of-funnel potential. Hub does not meet these — it depends on the spec layer — so it stays within the Synchestra brand.

## Open Questions

1. **LangChain/CrewAI integration** — Can Synchestra sit on top of LangChain/CrewAI as agent runtimes? Needs research. Do not claim or deny for now. If confirmed, strengthens the "layer above" positioning significantly.
2. **Hub pricing model** — Not relevant to naming/positioning; to be decided separately.
