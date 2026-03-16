# Feature: Test Scenario

**Status:** Conceptual

## Summary

A test scenario is a markdown file that reads like a step-by-step guide and executes like a test suite. Each scenario composes [acceptance criteria](../../acceptance-criteria/README.md) into multi-step workflows with named steps, data flowing between them, and structured pass/fail reporting. The format requires no special tooling to read — it renders natively on GitHub, in any markdown editor, and in Synchestra's own UI. The [test runner](../test-runner/README.md) executes it.

## Problem

Verifying multi-step workflows today means choosing between bad options:

- **Ad-hoc shell scripts** get the job done but are hard to read, impossible to compose with ACs, and produce unstructured output that nobody reviews after CI turns green.
- **Go test functions** couple verification to the implementation language. A product owner cannot review whether the test matches the requirement without reading Go. An AI agent working at the spec level cannot author or modify them without Go expertise.
- **Gherkin/Cucumber** separates intent from implementation — but at a cost. Step definitions live in code files, far from the feature they describe. Extending requires writing glue code. The `.feature` file looks clean; the machinery behind it does not.

Synchestra's scenario format avoids all three traps: human-readable markdown that renders anywhere, bash blocks that execute directly (no glue code), and native AC composition that references verification logic instead of reimplementing it.

## Behavior

### Scenario file structure

A test scenario is a markdown file with a title, metadata, and named steps:

```markdown
# Scenario: Project lifecycle

**Description:** End-to-end test of creating a project and verifying its configuration files.
**Tags:** e2e, cli, project

## Setup

```bash
export TEST_DIR=$(mktemp -d)
export SPEC_BARE=$(mktemp -d)/spec.git && git init --bare "$SPEC_BARE"
export STATE_BARE=$(mktemp -d)/state.git && git init --bare "$STATE_BARE"
```

## create-project

**Outputs:**

| Name | Store | Extract |
|---|---|---|
| spec_repo_path | context | `echo $HOME/synchestra-repos/spec` |
| expected_title | context | `echo "Test Project"` |

```bash
synchestra project new --spec-repo "$SPEC_BARE" --state-repo "$STATE_BARE"
```

## verify-configs

**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/new](spec/features/cli/project/new/) | * |

```bash
echo "AC verification handles the assertions"
```

## Teardown

```bash
rm -rf "$TEST_DIR" "$SPEC_BARE" "$STATE_BARE"
```
```

This is not pseudo-code. This is an actual scenario file that the runner executes. The same document that a product owner reads to understand the test flow is the same artifact that CI runs to verify the product works.

### Scenario metadata

| Field | Required | Description |
|---|---|---|
| Title | Yes | `# Scenario: {name}` — human-readable scenario name |
| Description | Yes | One-line summary of what the scenario verifies |
| Tags | No | Comma-separated labels for filtering (`e2e`, `cli`, `smoke`, `regression`, etc.) |

### Step elements

Each step is an `## {step-name}` heading. Steps contain a mix of optional metadata and a bash code block:

| Element | Required | Description |
|---|---|---|
| Heading | Yes | `## {step-name}` — kebab-case, unique within the scenario |
| Depends on | No | Steps that must complete before this one starts |
| Parallel | No | `true` to run this step concurrently with adjacent parallel steps |
| Inputs | No | Table declaring required context/step variables |
| Outputs | No | Table with Name, Store, and Extract columns |
| ACs | No | Table with Feature and ACs columns — links to [acceptance criteria](../../acceptance-criteria/README.md) to verify after this step |
| Include | No | Delegates to a sub-flow `.md` file (mutually exclusive with code block) |
| Code block | Conditional | Bash script to execute (required unless Include is specified) |

A step with neither a code block nor an Include directive is a validation error — every step must do something.

### Step identification and references

Steps are identified by name, not position. Names are kebab-case (e.g., `create-project`, `verify-configs`) and must be unique within a scenario. `Depends on` references use these names, which means reordering steps in the file does not break dependency declarations.

### Reserved steps: Setup and Teardown

`## Setup` and `## Teardown` are reserved step names with special lifecycle behavior:

- **Setup** runs before all named steps. If it fails, no steps execute — the scenario is dead on arrival.
- **Teardown** runs after all steps complete, **always** — even when steps fail, even when Setup fails. It is the unconditional cleanup hook.

Both are optional. Neither supports Outputs, ACs, Parallel, or Depends on — they are lifecycle hooks, not test steps.

### Execution model

Steps execute **sequentially in file order** by default. This is the obvious, predictable behavior — "do A, then B, then C" means exactly that.

**Parallel groups:** Consecutive steps marked `**Parallel:** true` form a concurrent group. The group starts after the preceding sequential step completes, and the next sequential step waits for the entire group to finish:

```markdown
## step-a
(sequential — runs first)

## step-b
**Parallel:** true
(starts concurrently with step-c)

## step-c
**Parallel:** true
(runs alongside step-b)

## step-d
(sequential — waits for both step-b and step-c to complete)
```

**Depends on:** Explicit dependencies refine ordering within parallel groups. A step with `**Depends on:** step-x` will not start until `step-x` completes, even if both are in the same parallel group. Dependencies must reference steps defined earlier in the file — no forward references, no cycles.

### Output model

Steps produce data that later steps consume. Outputs are declared in a table and extracted from the step's execution:

| Store | Scope | Access syntax | Description |
|---|---|---|---|
| `context` | Global | `${{ context.name }}` | Available to all subsequent steps. Names must be unique across the scenario. |
| `step` | Local | `${{ steps.{step-name}.outputs.{name} }}` | Available only via explicit reference. Names can repeat across different steps. |
| `both` | Both | Either syntax | Stored in both scopes for convenience. |

The **Extract** column contains a shell expression evaluated against the step's stdout (`$STEP_STDOUT`) and stderr (`$STEP_STDERR`). The expression's own stdout becomes the output value. This keeps extraction logic visible — no hidden post-processing.

### AC reference syntax

Steps declare which acceptance criteria to verify after execution:

```markdown
**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/remove](spec/features/cli/project/remove/) | * |
| [cli/project/list](spec/features/cli/project/list/) | [in-list](spec/features/cli/project/list/_acs/in-list.md) |
```

| Pattern | Meaning |
|---|---|
| `*` | All ACs under that feature's `_acs/` directory (alphabetical order) |
| `[ac-name](link)` | A specific AC by slug |
| `[ac1](link), [ac2](link)` | Multiple specific ACs, comma-separated (executed in listed order) |

The runner parses only the first two columns (Feature, ACs). Additional columns are allowed for human-readable notes — the runner ignores them. This lets scenario authors annotate without breaking execution.

### Include (sub-flows)

A step can delegate its work to a separate scenario file:

```markdown
## verify-project-exists

**Include:** [flows/verify-project.md](spec/tests/flows/verify-project.md)
```

The included file is a full scenario with its own steps. This enables reuse: a "verify project exists" flow used by five different E2E scenarios is written once and included everywhere.

- **Context is shared.** The sub-flow reads and writes the same context as the parent.
- **Step outputs are namespaced.** Sub-flow outputs are accessible as `${{ steps.{include-step}.{sub-step}.outputs.{name} }}`.
- **Circular includes are detected and rejected.** The runner tracks the include chain and fails fast on cycles.

Sub-flows live in `spec/tests/flows/` for cross-feature reuse, or `spec/features/{feature}/_tests/flows/` for feature-scoped reuse.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Acceptance Criteria](../../acceptance-criteria/README.md) | Scenarios reference ACs via table syntax. AC verification scripts are the atomic assertions that scenarios compose. |
| [Test Runner](../test-runner/README.md) | The runner parses scenario files, resolves AC references, executes steps, and produces structured reports. |
| [Testing Framework](../README.md) | Parent feature — defines file locations, CLI commands, and design principles. |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria not yet defined for this feature.
- Should scenarios support conditional steps (skip-if / run-if based on output values) for handling platform-specific or environment-specific test paths?
- Should there be a maximum nesting depth for includes to prevent overly complex test hierarchies?
- Should step timeouts be configurable per-step via a `**Timeout:**` metadata field, or only at the framework level?
