# Feature: Stakeholder / Notification

**Status:** Conceptual

## Summary

The notification sub-feature defines how stakeholders learn about [decisions](../decision/README.md) and how responses flow back to requesting agents. It separates **signal** (something happened) from **payload** (the details), using payload size to determine delivery strategy.

## Problem

Creating a decision task is not enough — stakeholders need to know about it, and requesting agents need to know when it resolves. Without structured notification:

- Decisions sit unnoticed until someone happens to check the task board
- Agents remain blocked longer than necessary because there is no push mechanism for resolution
- Human stakeholders miss time-sensitive decisions because they are not actively monitoring Synchestra

## Behavior

### Signal vs Payload

Every decision event (created, response received, resolved, expired) produces a notification. The notification has two parts:

- **Signal** — always delivered. Contains: decision ID, task reference, role, one-line summary. Enough to know what happened and where to look.
- **Payload** — the full decision context or response content. Delivery depends on size.

### Size-Based Payload Delivery

| Payload size | Delivery |
|---|---|
| Small (below threshold) | Inlined in the notification. Recipient has everything without a round-trip. |
| Large (above threshold) | Notification contains a reference (task path/URL). Recipient reads full content from the task directory. |

The threshold is configurable per project. Default: 500 characters. Below that, the full text of the option selected or response provided is included inline. Above that, the notification says "decision resolved — see task" with a link.

### Delivery Channels

Notifications are delivered through available channels. The stakeholder feature does not implement channels — it produces notification events that channel integrations consume.

| Channel | Integration | Notes |
|---|---|---|
| Synchestra bot | [bots](../../bots/README.md) | Telegram, Slack, etc. via bots-fw |
| GitHub | [github-app](../../github-app/README.md) | Review requests, PR comments, issue mentions |
| Webhook | [api](../../api/README.md) | HTTP POST to configured endpoints |
| Email | Future integration | Notification-only initially |
| CLI polling | [cli](../../cli/README.md) | For agents in active sessions |

**Email** is notification-only in the initial implementation. Reply-by-email (stakeholder responds directly to the email with their choice) is a future enhancement — the email would contain instructions like "Reply with A, B, or C" and Synchestra would parse the inbound email to record the response.

### Agent Resumption

When a decision resolves and the requesting agent's task is unblocked, the resumption strategy depends on the agent's runtime state:

**Live agent** (long-running session, still connected):
- Push the signal directly to the agent's session
- Agent reads the decision outcome and continues work
- The runtime integration determines how the push is delivered (CLI event, websocket, etc.)

**Ephemeral agent** (session ended, no active connection):
- The task transitions from `blocked` back to `queued`
- The next agent that picks up the task finds the decision in the [audit log](../decision/audit/README.md)
- The task's markdown body is updated with the resolved outcome so the agent has full context

In both cases, the decision outcome is written to the task **before** unblocking, ensuring the agent always has a consistent read path regardless of how it resumes.

### Notification Events

| Event | Recipients | Payload |
|---|---|---|
| `decision.created` | Assigned stakeholders | Decision summary, options, context link |
| `decision.response` | Requester, other assigned stakeholders | Who responded, what they chose |
| `decision.resolved` | Requester, all assigned stakeholders | Outcome, audit summary |
| `decision.expired` | Requester, all assigned stakeholders | Expiry notice, escalation info (if configured) |

### Channel Configuration

Stakeholders can configure their preferred notification channels. This is out of scope for the stakeholder feature itself — it will be part of the future stakeholder registry (`stakeholders.yaml`) or user profile configuration.

Until per-stakeholder preferences exist, notifications are delivered to all available channels for the project.

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria are not yet defined for this feature.

- Should notification preferences be per-stakeholder, per-role, or per-gate (e.g., "email me for spec-review but bot me for code-review")?
- What is the retry strategy for failed notification delivery — should Synchestra retry, or is delivery best-effort with the task board as the fallback source of truth?
- Should there be a "digest" mode that batches multiple pending decisions into a single notification for stakeholders who are assigned to many roles?
