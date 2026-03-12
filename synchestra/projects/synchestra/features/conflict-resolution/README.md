# Feature: Conflict Resolution

**Status:** Conceptual

## Summary

When git merge conflicts occur between concurrent agent operations, Synchestra launches a specialized AI sub-agent to analyze, resolve, or escalate the conflict —reducing human intervention for the mechanical cases while preserving human judgment for the ambiguous ones.

## Problem

Concurrent agents working on the same project will occasionally produce conflicting changes. The claim-and-push protocol prevents two agents from claiming the same task, but it doesn't prevent two agents working on different tasks from editing the same file or document section.

Git detects these conflicts but can't resolve them. Agents that encounter merge conflicts typically fail and require human intervention. For a system designed around async, concurrent work, this is a bottleneck.

## Proposed Behavior

### Detection

Conflicts are detected when an agent's `git push` fails due to diverged history. The Synchestra daemon (or pre-push hook) intercepts the failure and triggers the conflict resolution flow.

### Resolution tiers

1. **Auto-merge (no LLM needed).** Git's built-in merge handles non-overlapping changes. Synchestra attempts `git pull --rebase` first. If it succeeds, push and continue.

2. **AI-assisted merge.** If git can't auto-merge, Synchestra launches a conflict resolution sub-agent that:
   - Reads both versions and the common ancestor
   - Understands the intent of each change (from task descriptions and commit messages)
   - Produces a merged result
   - Validates the merge against the project schema (via inGitDB)
   - Commits and pushes if validation passes

3. **Human escalation.** If the sub-agent cannot confidently resolve the conflict (e.g., both sides made intentional but contradictory design decisions), it:
   - Creates a new task flagged for human resolution
   - Includes both versions, the sub-agent's analysis, and a recommended resolution
   - Notifies the user via configured channels (web UI, Telegram, webhook)

### Confidence threshold

The AI merge sub-agent assigns a confidence score to its resolution. Below a configurable threshold (default: high), it escalates to human review rather than auto-committing.

## Open Design Decisions

- What model should the conflict resolution sub-agent use? (Needs to understand code and intent —probably `medium` or `large`.)
- Should the resolution sub-agent have access to the full task context of both conflicting tasks, or just the diff?
- How is the confidence threshold configured and calibrated?
- Should there be a "dry-run" mode where the sub-agent proposes a resolution but always waits for human approval?

## Outstanding Questions

- How does this interact with branch protection rules that require PR reviews?
- What happens when the conflict resolution itself conflicts with another concurrent push? (Recursive conflict.)
- Should resolved conflicts be logged in a dedicated audit section of the affected document?
