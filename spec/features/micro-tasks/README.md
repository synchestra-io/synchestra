# Feature: Micro-Tasks

**Status:** Conceptual

## Summary

Micro-tasks are small, automated steps that Synchestra runs before, after, or in the background relative to a user's prompt. They handle the routine work that keeps a project consistent — formatting, validation, cross-reference updates, link checks — without burning tokens from the main task's context window.

## Problem

Every prompt an agent processes exists in a larger context. Before the agent starts, there may be setup work (pull latest state, validate inputs, gather context). After it finishes, there may be cleanup work (format output, update cross-references, validate schema). Running this in parallel, there may be background work (update links, check consistency of sibling documents).

Today, this work either doesn't happen (drift accumulates) or gets included in the main prompt (wasting expensive tokens on mechanical tasks).

## Proposed Behavior

Micro-tasks are configured per-project or per-module and execute as a chain around the user's prompt:

```
[pre-tasks] → [user prompt / main task] → [post-tasks]
                     ↕
              [background tasks]
```

- **Pre-tasks** run sequentially before the main task. If a pre-task fails, the main task does not start.
- **Post-tasks** run sequentially after the main task completes successfully.
- **Background tasks** run in parallel with the main task. They do not block the main task and their failure is logged but does not fail the main task.

## Configuration (Draft)

Modeled after GitHub Actions workflow jobs:

```yaml
micro-tasks:
  pre:
    - name: validate_schema
      cmd: synchestra validate
      model_class: none  # No LLM needed, pure CLI
    - name: gather_context
      cmd: synchestra context generate
      model_class: small

  post:
    - name: format_markdown
      cmd: format_markdown
      model_class: none
    - name: update_cross_references
      cmd: synchestra refs update
      model_class: small
      suggested_model: claude-haiku-4.5

  background:
    - name: check_links
      cmd: synchestra links check
      model_class: none
    - name: update_sibling_summaries
      cmd: synchestra summarize --scope siblings
      model_class: small
```

### Configuration fields

| Field | Required | Description |
|---|---|---|
| `name` | Yes | Human-readable identifier for the micro-task |
| `cmd` | Yes | Command to execute (CLI command, script path, or built-in operation) |
| `model_class` | No | `none`, `small`, `medium`, `large`. Defaults to `none`. |
| `suggested_model` | No | Specific model hint (e.g., `claude-haiku-4.5`). Overridable by user. |
| `skills` | No | List of Synchestra skills to make available to the micro-task agent |
| `agent` | No | Specific agent name to use for execution |
| `on_failure` | No | `stop` (default for pre), `log` (default for post/background), `retry` |
| `max_retries` | No | Number of retry attempts if `on_failure: retry`. Default: 1. |

### Configuration inheritance

Micro-task configs can be defined at multiple levels:
1. **Global** — applies to all projects in this Synchestra instance
2. **Project** — applies to all tasks in a specific project
3. **Module** — applies to tasks within a specific module (e.g., `spec/`, `tasks/`)
4. **Task** — applies to a specific task only

Lower levels override higher levels. A task-level config replaces (not merges with) the module-level config for the same phase.

## Examples

### Minimal: just format output

```yaml
micro-tasks:
  post:
    - name: format_markdown
      cmd: format_markdown
```

### Full pipeline: validate, gather, format, update

```yaml
micro-tasks:
  pre:
    - name: pull_latest
      cmd: git pull --rebase
    - name: validate_inputs
      cmd: synchestra validate --scope task
  post:
    - name: format_output
      cmd: format_markdown
    - name: update_refs
      cmd: synchestra refs update
      model_class: small
  background:
    - name: update_summaries
      cmd: synchestra summarize --scope siblings
      model_class: small
```

## Open Design Decisions

- Should micro-tasks have access to the main task's context, or are they fully isolated?
- Should background tasks be cancellable if the main task fails?
- How does the micro-task chain interact with the task claiming protocol? (Does the pre-task chain run before or after claiming?)
- Should there be a `finally` phase that runs regardless of main task success/failure?
- Can micro-tasks spawn sub-tasks, or are they strictly leaf operations?

## Outstanding Questions

- What is the config file name and location? (e.g., `.synchestra/micro-tasks.yaml`, or inline in the project config?)
- How are custom `cmd` values resolved — PATH lookup, relative to project root, or registered scripts?
- What is the logging/observability story for micro-task execution? Are results stored as documents?
- How does `model_class` interact with the model-selection feature — does micro-task config feed into the same routing logic?
