# Feature: Model Selection

**Status:** Conceptual

## Summary

Not every task needs the most powerful (and expensive) model. Synchestra routes tasks to the minimal viable model — either through explicit configuration, rule-based routing, or dynamic complexity assessment using a smaller model as a classifier.

## Problem

Running every task on the largest available model is wasteful. Formatting a markdown file doesn't need Opus. Updating a cross-reference link doesn't need any LLM at all. But without a routing layer, every agent session defaults to whatever model the platform is configured with — usually the most capable (and expensive) one.

Over hundreds of micro-tasks per day, this adds up to significant unnecessary cost and latency.

## Proposed Behavior

The stable user-facing profiles are `fast`, `balanced`, and `large`. Existing internal `small`, `medium`, and `large` values map respectively to those profiles during migration. Model selection operates at three levels, in order of precedence:

### 1. User override (highest priority)

Users can force a specific model via:
- **CLI argument:** `synchestra task run --model claude-opus-4`
- **API parameter:** `{"model": "claude-opus-4"}`
- **Web UI:** Model selector in the task queue interface

When set, this is passed as an argument or hint to the underlying agent platform (Claude Code, Copilot, OpenCode, etc.).

### 2. Configuration rules

Micro-task configs and project settings can specify `model_class` or `suggested_model`:

```yaml
model_class: small          # Routes to the cheapest model in the "small" tier
suggested_model: claude-haiku-4.5  # Specific suggestion, overridable
```

Model classes map to platform-specific models:

| Class | Claude | OpenAI | Description |
|---|---|---|---|
| `none` | — | — | No LLM needed; pure CLI/script execution |
| `fast` (`small`) | Configured fast Claude model | Configured fast OpenAI model | Fast, cheap. Formatting, classification, simple edits. |
| `balanced` (`medium`) | Configured balanced Claude model | Configured balanced OpenAI model | Balanced. Most implementation tasks. |
| `large` | Configured large Claude model | Configured large OpenAI model | Maximum capability. Architecture, complex reasoning. |

### 3. Dynamic assessment (lowest priority)

When no explicit model is configured, Synchestra can use a small model to assess the task's complexity before routing:

1. Feed the task description and minimal context to a fast model
2. Ask it to classify the task complexity (`fast` / `balanced` / `large`)
3. Route to the appropriate model

This adds one cheap LLM call per task but can save significantly on tasks that would otherwise default to the largest model.

## Platform interaction

Synchestra is not always the direct caller of the LLM. When an agent runs inside Claude Code or Cursor, the platform controls the model. Synchestra's model selection manifests as:

- **Hint:** Passed as metadata that the agent can use or ignore
- **Argument:** Passed as a CLI flag or API parameter that the platform respects (e.g., `claude --model haiku`)
- **Enforcement:** For Synchestra-spawned headless agents, Synchestra controls the model directly

## Outstanding Questions

- How should older `small` / `medium` values be removed after all runtime and UI contracts use `fast` / `balanced`?
- How does dynamic assessment handle tasks where the description is vague? (Default to `medium`? Ask for clarification?)
- Should there be a cost budget feature that constrains model selection? (e.g., "this project has a $50/day budget, optimize accordingly")
- How does the versioned model-profile mapping stay current as new models are released?

---
*This document follows the https://specscore.md/feature-specification*
