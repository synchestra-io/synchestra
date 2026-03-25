# Feature: UI / Hub

**Status:** Conceptual

## Summary

Synchestra Hub is the browser-based management interface for Synchestra. Users browse projects, manage AI-agent runners, and handle tasks (such as review requests from agents) — all from a web UI at [hub.synchestra.io](https://hub.synchestra.io).

This document covers what is **unique to the Hub surface** — technology, API integration, authentication, runner management, and hosting model. For the screens and navigation tree, see the [parent UI spec](../README.md#proposed-behavior).

## Problem

Terminal-first users have the [TUI](../tui/README.md), but many Synchestra users — especially non-developers reviewing features or approving proposals — need a graphical interface accessible from any browser or mobile device without installing tooling or cloning a repository.

## Proposed Behavior

### Technology stack

| Layer | Choice |
|---|---|
| Language | TypeScript |
| Monorepo | Nx |
| Framework | Angular |
| UI toolkit | Ionic |
| Delivery | Progressive Web App (PWA) |

### Hosting model

The Hub is initially available only at [hub.synchestra.io](https://hub.synchestra.io) as a managed service. Self-hosting support is planned for a future release.

### Implementation repository

The Hub is implemented in the [synchestra-app](https://github.com/synchestra-io/synchestra-app) repository. This spec defines what the Hub must do; `synchestra-app` is where the code lives.

### Backend communication

All data flows through the [HTTP API](../../../../docs/api/README.md) (`/api/v1/`). The Hub does not read the git repository directly.

Key API surfaces the Hub consumes:

| Flow | API resource group |
|---|---|
| Project list | `/projects` |
| Feature browsing | `/projects/:id/features` |
| Proposal creation | `/projects/:id/proposals` |
| Task list / create / enqueue | `/tasks` |
| Runner management | `/runners` |
| Authentication | `/auth` — see [Auth API](../../../../docs/api/auth.md) |

### Project browsing

The Hub can load project data directly from GitHub. Users connect their GitHub account and browse any repository that contains a Synchestra spec tree without cloning it locally.

### Runner management

Users manage AI-agent execution environments ("runners") through the Hub:

- **VMs** — cloud or on-premise virtual machines
- **Cloud Docker images** — containerized runner environments
- **Local machines** — development environments registered as runners

The Hub provides visibility into runner status, allows provisioning and decommissioning, and routes queued tasks to available runners.

### Task management

The Hub surfaces tasks created by both humans and AI agents. Key workflows:

- Browse and filter tasks by status, assignee, or feature
- Review requests from AI agents (approve, reject, or request changes)
- Create and enqueue tasks manually
- Monitor task progress and agent activity

### Authentication

The Hub authenticates users before any project access. The [API](../../../../docs/api/README.md) requires a Bearer token in the `Authorization` header.

For MVP, the Hub supports:

- **GitHub OAuth** — primary flow; the user's GitHub identity is used to sign prompt commits and co-author artifact commits (per root README).
- **Firebase Auth** — alternative provider for environments where GitHub OAuth is not available.

The authenticated user identity determines which projects appear on the home screen and which actions are permitted (e.g., submitting a proposal, creating a task).

### PWA capabilities

As a progressive web app, the Hub supports:

- **Installability** — users can add it to their home screen / app launcher
- **Offline access** — cached project and feature data remains browsable when offline; mutations queue and sync when connectivity returns
- **Responsive layout** — usable on desktop, tablet, and phone form factors

### Content rendering

Feature specifications and proposals are stored as Markdown (`README.md` files). The Hub must render this content faithfully.

Options:

1. The API returns pre-rendered HTML
2. The Hub receives raw Markdown and renders client-side

The choice affects offline behavior and rendering fidelity. This is an open question.

### Surface-specific interaction notes

- The **New proposal** button appears on the feature detail screen alongside the proposals list (per [Proposals UI behavior](../../proposals/README.md#synchestra-ui-behavior)).
- Task creation uses a form that maps to the fields accepted by [`synchestra task create`](../../cli/task/create/README.md).
- The proposals list on the feature screen must visually distinguish proposals from the feature's normative specification content.

## Outstanding Questions

- Should the Hub render Markdown client-side or receive pre-rendered HTML from the API?
- What is the offline mutation strategy — optimistic local writes with background sync, or explicit "you are offline" gating?
- Should the Hub support real-time updates (WebSocket / SSE) for task board changes, or is polling sufficient for MVP?
- What Ionic components map to the project menu and feature detail screens? Should these be defined in a design system before implementation begins?
- How does GitHub OAuth token refresh work in the context of long-lived PWA sessions?
- What is the timeline and requirements for self-hosting support?
- How does runner provisioning integrate with cloud provider APIs (AWS, GCP, etc.)?
