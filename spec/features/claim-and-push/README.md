# Feature: Claim-and-Push

**Status:** Conceptual

## Summary

Distributed task claiming through git's built-in push semantics. Agents claim tasks by committing a status change and pushing. If the push fails, another agent got there first. No central lock server needed.

## Problem

When multiple agents run concurrently against the same task queue, two agents may try to start the same task simultaneously. Traditional solutions require a central lock server or database with atomic transactions. Synchestra's storage layer is git, which doesn't have row-level locking.

## Proposed Behavior

### Claim protocol

When an agent is ready for work:

1. **Pull latest state.** `git pull --rebase` to ensure the agent sees current task statuses.
2. **Find an unclaimed task.** Query `synchestra task list --status pending` or navigate the `tasks/` directory.
3. **Claim the task.** Update the task's status to `claimed` with the agent's identity and a timestamp.
4. **Commit.** `git commit` the status change.
5. **Push.** `git push` immediately.

### Conflict = someone else claimed first

If step 5 fails because the remote has diverged:
- Another agent pushed first (likely claiming the same or a different task that touched the same status document)
- The agent pulls, checks if the task is still unclaimed
- If yes: re-commit and push
- If no: move to the next available task
- If no tasks available: exit or wait

### Status transitions via claim

```
pending → claimed → in_progress → complete
                                → failed
                                → blocked
```

The `claimed` status is the lock. It means "an agent has committed to working on this and will transition to `in_progress` shortly."

### Commit-often philosophy

Once claimed, the agent should commit frequently:
- On starting work: transition to `in_progress`
- On meaningful progress: commit partial results
- On completion: transition to `complete`
- On failure: transition to `failed` with reason

Frequent commits minimize the conflict window and provide granular audit trail.

## Open Design Decisions

- Should `claimed` have a TTL? (If an agent claims a task and crashes, the task is stuck in `claimed` forever.)
- How is agent identity represented in the claim? (Git author? A Synchestra agent ID?)
- Should claiming be done via CLI only (`synchestra task claim`) or is manual git commit also valid?
- What if the status document contains multiple tasks and two agents claim different tasks? (The push still conflicts on the same file.)

## Outstanding Questions

- What is the exact format of the status document? (YAML frontmatter in README? Separate status file?)
- How does the daemon detect a crashed agent and reclaim stuck tasks?
- Should there be a `synchestra task release` command for agents that claim a task but decide not to do it?
