# Feature: UI / Web App

**Status:** Conceptual

## Summary

The web app is Synchestra's browser-based progressive web application. It implements the [shared information architecture](../README.md#information-architecture) defined in the parent UI feature spec, delivered as an installable PWA that communicates with the Synchestra backend via the [HTTP API](../../../../docs/api/README.md).

This document covers what is **unique to the web surface** — technology, API integration, authentication, and PWA behavior. For the screens and navigation tree, see the [parent UI spec](../README.md#proposed-behavior).

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

### Implementation repository

The web app is implemented in the [synchestra-app](https://github.com/synchestra-io/synchestra-app) repository. This spec defines what the web app must do; `synchestra-app` is where the code lives.

### Backend communication

All data flows through the [HTTP API](../../../../docs/api/README.md) (`/api/v1/`). The web app does not read the git repository directly.

Key API surfaces the web app consumes:

| Flow | API resource group |
|---|---|
| Project list | `/projects` |
| Feature browsing | `/projects/:id/features` |
| Proposal creation | `/projects/:id/proposals` |
| Task list / create / enqueue | `/tasks` |
| Worker visibility | `/workers` (future) |
| Authentication | `/auth` — see [Auth API](../../../../docs/api/auth.md) |

### Authentication

The web app authenticates users before any project access. The [API](../../../../docs/api/README.md) requires a Bearer token in the `Authorization` header.

For MVP, the web app supports:

- **GitHub OAuth** — primary flow; the user's GitHub identity is used to sign prompt commits and co-author artifact commits (per root README).
- **Firebase Auth** — alternative provider for environments where GitHub OAuth is not available.

The authenticated user identity determines which projects appear on the home screen and which actions are permitted (e.g., submitting a proposal, creating a task).

### PWA capabilities

As a progressive web app, the web surface should support:

- **Installability** — users can add it to their home screen / app launcher
- **Offline access** — cached project and feature data remains browsable when offline; mutations queue and sync when connectivity returns
- **Responsive layout** — usable on desktop, tablet, and phone form factors

### Content rendering

Feature specifications and proposals are stored as Markdown (`README.md` files). The web app must render this content faithfully.

Options:

1. The API returns pre-rendered HTML
2. The web app receives raw Markdown and renders client-side

The choice affects offline behavior and rendering fidelity. This is an open question.

### Surface-specific interaction notes

- The **New proposal** button appears on the feature detail screen alongside the proposals list (per [Proposals UI behavior](../../proposals/README.md#synchestra-ui-behavior)).
- Task creation uses a form that maps to the fields accepted by [`synchestra task create`](../../cli/task/create/README.md).
- The proposals list on the feature screen must visually distinguish proposals from the feature's normative specification content.

## Outstanding Questions

- Should the web app render Markdown client-side or receive pre-rendered HTML from the API?
- What is the offline mutation strategy — optimistic local writes with background sync, or explicit "you are offline" gating?
- Should the web app support real-time updates (WebSocket / SSE) for task board changes, or is polling sufficient for MVP?
- What Ionic components map to the project menu and feature detail screens? Should these be defined in a design system before implementation begins?
- How does GitHub OAuth token refresh work in the context of long-lived PWA sessions?
