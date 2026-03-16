# Feature: Test Scenario

**Status:** Conceptual

## Summary

Defines the markdown-native scenario format — a human-readable `.md` file with named steps, dependency declarations, input/output passing, [acceptance criteria](../../acceptance-criteria/README.md) references, and sub-flow includes. Scenarios are both documentation and executable test definitions.

## Problem

Without a structured test scenario format, multi-step verification requires either:
- **Ad-hoc shell scripts** that are hard to read, don't compose ACs, and lack structured reporting.
- **Go test functions** that couple verification to the implementation language and can't be authored by non-Go developers or AI agents working at the spec level.
- **Third-party test frameworks** that introduce external dependencies and don't integrate with Synchestra's AC system.

A markdown-native format solves all three: human-readable, AC-composable, and executable by the [test runner](../test-runner/README.md) without language-specific tooling.

## Behavior

### Scenario file structure

A test scenario is a markdown file with a title, metadata, and named steps:

```markdown
# Scenario: Project lifecycle

**Description:** E2E test of project create and verify.
**Tags:** e2e, cli

## Setup

```bash
export TEST_DIR=$(mktemp -d)
```

## create-project

**Outputs:**

| Name | Store | Extract |
|---|---|---|
| project_id | context | `jq -r '.id' $STEP_STDOUT` |

**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/new](spec/features/cli/project/new/) | * |

```bash
synchestra project new --format json
```

## Teardown

```bash
rm -rf $TEST_DIR
```
```

### Scenario metadata

| Field | Required | Description |
|---|---|---|
| Title | Yes | `# Scenario: {name}` — human-readable name |
| Description | Yes | One-line summary of what the scenario tests |
| Tags | No | Comma-separated labels for filtering (`e2e`, `cli`, `smoke`, etc.) |

### Step elements

Each step is an `## {step-name}` heading with optional metadata and a bash code block:

| Element | Required | Description |
|---|---|---|
| Heading | Yes | `## {step-name}` — kebab-case, unique within the scenario |
| Depends on | No | References to other steps that must complete first |
| Parallel | No | `true` to mark this step as part of a concurrent group |
| Inputs | No | Table declaring required context/step variables |
| Outputs | No | Table with Name, Store (`context`/`step`/`both`), and Extract columns |
| ACs | No | Table with Feature and ACs columns — references to [acceptance criteria](../../acceptance-criteria/README.md) |
| Include | No | Delegates to a sub-flow `.md` file (mutually exclusive with code block) |
| Code block | Conditional | Bash script to execute (required unless Include is specified) |

A step with neither a code block nor an Include directive is a validation error.

### Step identification and references

Steps are identified by their heading name (kebab-case, e.g., `create-project`, `verify-configs`). Step names must be unique within a scenario. Names — not positions — are used for `Depends on` references, which makes scenarios resilient to reordering.

### Reserved steps: Setup and Teardown

`## Setup` and `## Teardown` are reserved step names with special behavior:

- **Setup** runs before all other steps. If it fails, no steps execute and the scenario fails.
- **Teardown** runs after all steps complete, **always** — even on failure. It is the cleanup hook.

Both are optional. Neither supports Outputs, ACs, Parallel, or Depends on.

### Execution model

Steps execute **sequentially in file order** by default. This is the simple, predictable default.

**Parallel groups:** Consecutive steps marked `**Parallel:** true` form a parallel group. The group starts after the preceding sequential step completes, and the next non-parallel step waits for the entire group to finish.

```markdown
## step-a
(sequential — runs first)

## step-b
**Parallel:** true
(parallel group starts)

## step-c
**Parallel:** true
(part of the same parallel group as step-b)

## step-d
(sequential — waits for step-b and step-c to complete)
```

**Depends on:** Explicit dependencies override file order within parallel groups. A step with `**Depends on:** step-x` will not start until `step-x` completes, even if both are in the same parallel group. Dependencies must reference steps that appear earlier in the file.

### Output model

Steps can declare outputs that are stored and accessible to later steps:

| Store | Scope | Access syntax | Description |
|---|---|---|---|
| `context` | Global | `${{ context.name }}` | Available to all subsequent steps. Context names must be unique across the scenario. |
| `step` | Local | `${{ steps.{step-name}.outputs.{name} }}` | Available only via explicit reference. Step-scoped outputs can share names across steps. |
| `both` | Both | Either syntax | Stored in both scopes. |

The **Extract** column contains a shell expression that runs against the step's stdout (`$STEP_STDOUT`) and stderr (`$STEP_STDERR`). The expression's stdout becomes the output value.

### AC reference syntax

ACs are referenced via a table with Feature and ACs columns:

```markdown
**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/remove](spec/features/cli/project/remove/) | * |
| [cli/project/list](spec/features/cli/project/list/) | [in-list](spec/features/cli/project/list/_acs/in-list.md) |
```

| Pattern | Meaning |
|---|---|
| `*` | All ACs under that feature's `_acs/` directory |
| `[ac-name](link)` | Specific AC by slug (linked to the AC file) |
| `[ac1](link), [ac2](link)` | Multiple specific ACs, comma-separated in the ACs column |

The runner only parses the first two columns (Feature, ACs). Additional columns are allowed for human readability — the runner ignores them.

When specific ACs are listed, they execute in the order specified in the table. When `*` is used, ACs execute in alphabetical order by slug.

### Include (sub-flows)

A step can delegate to a sub-flow `.md` file instead of containing its own code block:

```markdown
## verify-project-exists

**Include:** [flows/verify-project.md](spec/tests/flows/verify-project.md)
```

The included file is a full scenario with its own steps. Behavior:

- **Context is shared.** The sub-flow reads and writes the same context as the parent.
- **Step outputs are namespaced.** Sub-flow step outputs are accessible as `${{ steps.{include-step}.{sub-step}.outputs.{name} }}`.
- **Circular includes are detected and rejected.** The runner tracks the include chain and errors on cycles.

Sub-flows live in `spec/tests/flows/` for cross-feature flows, or `spec/features/{feature}/_tests/flows/` for feature-scoped flows.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Acceptance Criteria](../../acceptance-criteria/README.md) | Scenarios reference ACs via table syntax. AC verification scripts are the atomic assertions. |
| [Test Runner](../test-runner/README.md) | The runner parses scenarios, resolves AC references, executes steps, and reports results. |
| [Testing Framework](../README.md) | Parent feature — defines file locations, CLI commands, and design principles. |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria not yet defined for this feature.
- Should scenarios support conditional steps (skip-if / run-if based on output values)?
- Should there be a maximum nesting depth for includes to prevent overly complex test hierarchies?
- Should step timeouts be configurable per-step, or only at the scenario/framework level?
