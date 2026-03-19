# Feature: Bots

**Status:** Conceptual

## Summary

Messenger bots that serve as conversational interfaces to Synchestra. Three distinct kinds of bots operate at different layers of the platform, each with its own lifecycle, hosting model, and audience.

## Contents

| Directory | Description |
|---|---|
| [synchestra-bot](synchestra-bot/README.md) | The platform-operated bot (initially Telegram) for project management, container control, and prompt relay |

### synchestra-bot

The first-party bot operated by the Synchestra platform. Embedded in the Synchestra server process, it provides a messenger-based interface for managing projects, controlling sandbox containers, and relaying prompts to in-container agents. Telegram is the initial platform; the architecture supports additional platforms (WhatsApp, etc.) via the bots-go-framework abstraction layer.

## Bot Taxonomy

Synchestra recognizes three kinds of bots:

| Kind | Hosting | Operator | Purpose |
|---|---|---|---|
| **SynchestraBot** | Synchestra server | Platform | Project management, container control, prompt relay, notifications |
| **In-container bots** | Inside sandbox container | User | User-defined bots running within a project's sandbox environment |
| **Host-level bots** | Host machine | User | User-defined bots running on the host outside of containers |

Only SynchestraBot is specified at this time. In-container and host-level bots will be defined as sub-features when their requirements are understood.

## Problem

Synchestra's primary interfaces (CLI, web UI, TUI) require users to be at a terminal or browser. Messenger bots provide a lightweight, mobile-friendly channel for common operations -- checking task status, sending prompts to agents, receiving notifications about container events -- without context-switching away from the tools people already use throughout the day.

## Behavior

Each bot kind has a distinct trust boundary and lifecycle:

- **SynchestraBot** is managed by the platform, runs inside the server, and has access to all Synchestra APIs. It authenticates messenger users via a linking flow and enforces per-user authorization.
- **In-container bots** (future) run inside a project's sandbox container. They have the same permissions as the container's agent and cannot escape the sandbox boundary.
- **Host-level bots** (future) run on the host machine outside containers. Their permissions and lifecycle are managed by the user, not by Synchestra.

### Dependencies

- [sandbox](../sandbox/README.md) -- container lifecycle that bots interact with
- [chat](../chat/README.md) -- complex prompts route through the chat feature
- [api](../api/README.md) -- bots use the API for Synchestra operations
- [cli](../cli/README.md) -- bot commands mirror CLI semantics

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Should in-container bots and host-level bots share a common registration/discovery mechanism, or are they fundamentally different enough to warrant separate specs?
- Do we need a bot permission model beyond the existing user/project authorization?
