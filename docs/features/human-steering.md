# Human Steering

**Summary:** Synchestra gives humans real-time visibility into what AI agents are doing and the controls to steer, approve, override, or halt execution at any point.

---

## Overview

The problem with AI pipelines isn't that they're autonomous — it's that they're *opaque*. Things happen, decisions get made, mistakes propagate, and by the time a human looks, the damage is done.

Synchestra inverts this. Agents report continuously; humans can look in at any time. And when you want to step in — approve a risky step, redirect an agent, update a task's acceptance criteria mid-flight — you can.

---

## Visibility

### System status

Get an instant overview of everything running:

```bash
synchestra status
```

Output:

```
Synchestra Status
=================
Server: running (uptime 14h 23m)
Agents: 4 active, 1 idle, 0 offline

Active tasks: 7
  task_abc1  [in_progress]  coder-agent-1   "Implement JWT auth"         (35m)
  task_abc2  [in_progress]  tester-agent    "Write auth tests"           (12m)
  task_abc3  [blocked]      coder-agent-2   "Refactor DB layer"          (2h)
  task_abc4  [in_progress]  reviewer-agent  "Review PR #42"              (8m)
  task_abc5  [pending]      —               "Deploy to staging"          
  task_abc6  [in_progress]  deployer-agent  "Update CI config"           (5m)
  task_abc7  [pending]      —               "Run smoke tests"            

Recent failures: 0
```

### Task detail

```bash
synchestra status task task_abc3
```

Shows the full timeline: status transitions, progress log entries, which agent did what, when.

---

## Notifications

Synchestra can push notifications to humans via:

### Telegram

Configure in your server config:

```yaml
notifications:
  telegram:
    bot_token: "bot123:abc..."
    chat_id: "123456789"
    events:
      - task_failed
      - task_blocked
      - agent_offline
      - task_complete  # optional — can be noisy
```

### Webhooks

```yaml
notifications:
  webhooks:
    - url: "https://your-system.example.com/hooks/synchestra"
      events:
        - task_failed
        - task_blocked
      headers:
        Authorization: "Bearer your-webhook-secret"
```

Webhook payload:

```json
{
  "event": "task_failed",
  "task_id": "task_abc123",
  "task_title": "Deploy to production",
  "agent_id": "deployer-agent",
  "reason": "Health check failed after deploy",
  "at": "2024-01-15T14:32:00Z"
}
```

---

## Steering Interventions

### Redirect a task

Update a task's assignment or description while it's in progress:

```bash
synchestra task update task_abc3 \
  --description "Focus on the UserRepository first, skip OrderRepository for now" \
  --agent coder-agent-3
```

### Unblock a task

If an agent marked a task as `blocked` waiting for something:

```bash
synchestra task update task_abc3 --status in_progress
```

### Fail and reassign

If you want to cancel a task and retry it with different parameters:

```bash
synchestra task fail task_abc3 --reason "Changing approach"

synchestra task new \
  --title "Refactor DB layer (revised approach)" \
  --parent task_parent_abc \
  --description "Use repository pattern instead of active record" \
  --agent coder-agent-1
```

---

## Rules for Guardrails

Rules let you bake human policy into the coordination layer, so agents don't need to ask permission for things that are already decided:

```bash
# No production deploys without human approval
synchestra rule new \
  --name "prod-deploy-approval" \
  --content "Before deploying to production, post a summary of changes and wait for explicit human approval via task comment or status update." \
  --scope project \
  --scope-id proj_abc123

# Never delete data
synchestra rule new \
  --name "no-data-deletion" \
  --content "Never DELETE records from the database. Use soft deletes (is_deleted flag) only." \
  --scope org \
  --scope-id org_xyz789
```

Agents that query their task context receive applicable rules. This is most useful for agents that pull their full context from Synchestra before starting work.

---

## Approval Gates

For high-stakes steps, pattern your workflow to pause and wait for human action:

```bash
# Orchestrator creates a "waiting for approval" task
synchestra task new \
  --title "APPROVAL NEEDED: Deploy v2.1.0 to production" \
  --description "Staging tests passed. Awaiting human approval to proceed." \
  --status pending \
  --agent human-review  # signals this needs a human

# Human reviews staging, approves by updating status
synchestra task update task_approval_abc --status complete

# Orchestrator detects completion and proceeds
```

---

## Audit Trail

Every change — who made it, when, what changed — is stored in the event log. This includes human interventions, not just agent actions.

```bash
synchestra status task task_abc3
# Shows: "11:30 - Status changed to in_progress by human_xyz (was: blocked)"
```

---

## Related

- [CLI: `synchestra status`](../cli/status.md)
- [CLI: `synchestra task update`](../cli/task.md#update)
- [CLI: `synchestra rule`](../cli/rule.md)
- [API: Tasks](../api/tasks.md)
- [Feature: Progress Reporting](progress-reporting.md)
- [Feature: State Synchronization](state-synchronization.md)
- [Self-Hosting: Notifications config](../self-hosting.md)
