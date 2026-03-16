# E2E Testing Framework & Acceptance Criteria

## Summary

Two interconnected design decisions:

1. **Acceptance Criteria (ACs)** become first-class, individually addressable artifacts within feature specs — mandatory, structured, executable, and reusable.
2. **Test scenarios** are markdown-native orchestration files that compose ACs into multi-step integration and E2E flows, with dependency-based parallel execution and sub-flow composition.

Together, they form a bidirectional system: features define *what* to verify (ACs), scenarios define *when* and *in what order* to verify (flows), and each links back to the other.

The test runner is built as `pkg/testscenario/` inside this repo, with the understanding that it may be decoupled into a standalone product in the future.

## Design Principles

- **ACs are never standalone executables.** They are verification blocks that always run as part of a test scenario — even a "single feature" test is a minimal scenario with at least one setup step plus ACs.
- **Scenarios are human-readable markdown.** A product person or AI agent can map them to acceptance criteria without knowing bash.
- **Inputs/outputs flow like GitHub Actions.** Steps declare outputs, downstream steps consume them by reference. No global mutable state.
- **Parallel by default.** Steps without `Depends on` declarations can run concurrently.
- **Composition via inclusion.** Scenarios can reference sub-flow `.md` files, enabling reuse without duplication.
- **The spec root is configurable.** All paths resolve relative to the configured spec root (default: `spec`), supporting projects that use `specifications` or other names.

## Part 1: Acceptance Criteria as First-Class Feature Artifacts

### AC file location and structure

Each AC lives in `spec/features/{feature}/acs/{ac-slug}.md`:

```markdown
# AC: not-in-list

**Status:** planned
**Feature:** [cli/project/remove](../README.md)

## Description

After project deletion, `synchestra project list` output does not contain
the id of the deleted project.

## Inputs

| Name | Description |
|---|---|
| project_id | ID of the project that was deleted |

## Verification

```bash
result=$(synchestra project list --format json)
! echo "$result" | jq -e ".[] | select(.id == \"$project_id\")"
```

## Scenarios

| Scenario | Step |
|---|---|
| [project-lifecycle](../../../tests/project-lifecycle.md) | Step 5 |
| [remove-and-recreate](../_tests/remove-and-recreate.md) | Step 1 |
```

### AC statuses

| Status | Description |
|---|---|
| `planned` | AC is described but has no verification script yet |
| `wip` | Verification script is being written/tested |
| `implemented` | Verification script exists and passes |
| `deprecated` | AC is no longer relevant |

### Feature README: Acceptance Criteria section

The **Acceptance Criteria** section is **mandatory** in every feature README — it is never omitted, following the same convention as Outstanding Questions.

**When no ACs are defined yet:**

```markdown
## Acceptance Criteria

Not defined yet.
```

And a corresponding Outstanding Question must be raised:

```markdown
## Outstanding Questions

- Acceptance criteria not yet defined for this feature.
```

**When ACs exist:**

```markdown
## Acceptance Criteria

| AC | Description | Status |
|---|---|---|
| [not-in-list](acs/not-in-list.md) | Deleted project absent from list | implemented |
| [recreate-same-id](acs/recreate-same-id.md) | Can recreate project with same id | planned |
```

The table is a derived summary — the AC `.md` file is the source of truth.

### Updates to the feature spec

The feature spec's **Required sections** table gains a new entry:

| Section | Required | Notes |
|---|---|---|
| Acceptance Criteria | Yes | Always present. States "Not defined yet." if empty; must also raise an Outstanding Question. |

### Mandatory enforcement

Validation tooling (lint/pre-commit) should check:
- Every feature README has an `## Acceptance Criteria` section
- If the section says "Not defined yet.", the Outstanding Questions section includes the corresponding question
- Every `.md` file in `acs/` is listed in the feature README table
- Every entry in the feature README table has a corresponding `.md` file in `acs/`

## Part 2: Test Scenario Format

### Scenario file structure

A test scenario is a markdown file with ordered steps, dependency declarations, and AC references.

```markdown
# Scenario: Project lifecycle

**Description:** End-to-end test of creating, using, and removing a project.
**Tags:** e2e, cli, project

## Setup

```bash
export SYNCHESTRA_HOME=$(mktemp -d)
synchestra config set --key server.host --value localhost
```

## 1. Create project

**Depends on:** (none)

**Outputs:**

| Name | Extract |
|---|---|
| project_id | `jq -r '.id' $STEP_STDOUT` |

**ACs:**

| AC | |
|---|---|
| cli/project/new/* | |

```bash
synchestra project new --repo https://github.com/example/test --format json
```

## 2. Verify project in list

**Depends on:** Step 1

**Inputs:**

| Name | Source |
|---|---|
| project_id | ${{ steps.1.outputs.project_id }} |

**ACs:**
- cli/project/list/in-list

```bash
synchestra project list --format json
```

## 3. Start container

**Depends on:** Step 1
**Include:** [flows/container-start.md](flows/container-start.md)

**Inputs:**

| Name | Source |
|---|---|
| project_id | ${{ steps.1.outputs.project_id }} |

## 4. Check container status

**Depends on:** Step 3

**ACs:**
- cli/server/project/status/*

```bash
synchestra server project status --project ${{ inputs.project_id }} --format json
```

## 5. Shutdown container

**Depends on:** Step 4

**ACs:**
- cli/server/project/shutdown/*

```bash
synchestra server project shutdown --project ${{ inputs.project_id }}
```

## 6. Remove project

**Depends on:** Step 5

**Inputs:**

| Name | Source |
|---|---|
| project_id | ${{ steps.1.outputs.project_id }} |

**ACs:**

| AC | Note |
|---|---|
| cli/project/remove/* | Validates full removal |

```bash
synchestra project remove --id ${{ inputs.project_id }}
```

## Teardown

```bash
rm -rf $SYNCHESTRA_HOME
```
```

### Step elements

| Element | Required | Description |
|---|---|---|
| Heading | Yes | `## N. Name` — numbered, names the step |
| Depends on | No | References to other steps by number. Absent = no dependencies (parallel-eligible) |
| Inputs | No | Table of inputs consumed from upstream step outputs |
| Outputs | No | Table of outputs extracted from step results (stdout, stderr, exit code) |
| ACs | No | List or table of AC references to verify after the step executes |
| Include | No | Delegates to a sub-flow `.md` file. Mutually exclusive with a code block |
| Code block | No | Bash script to execute. Mutually exclusive with Include |

### AC reference syntax

ACs can be referenced in two formats — the runner accepts either:

**List format (simple):**

```markdown
**ACs:**
- cli/project/remove/*
- cli/project/list/in-list
```

**Table format (extended):**

```markdown
**ACs:**

| AC | Note |
|---|---|
| cli/project/remove/* | Validates full removal |
| cli/project/list/in-list | Confirms absence from list |
```

The runner only parses column 1 of the table. Additional columns are for human readability — authors can name them whatever is useful (`Note`, `Status`, `Expected`, etc.).

**Wildcards and selection:**

| Pattern | Meaning |
|---|---|
| `cli/project/remove/*` | All ACs under `spec/features/cli/project/remove/acs/` |
| `cli/project/remove/not-in-list` | Specific AC by slug |
| `cli/project/remove/not-in-list,recreate-same-id` | Multiple specific ACs |

### Input/Output model

Modeled after GitHub Actions steps:

- **Outputs** are extracted from step results using shell expressions. Available variables: `$STEP_STDOUT` (path to stdout file), `$STEP_STDERR` (path to stderr file), `$STEP_EXIT_CODE`.
- **Inputs** are resolved from upstream outputs via `${{ steps.N.outputs.name }}` syntax.
- Inputs are passed to both the step's code block and its AC verification scripts as environment variables.

### Include (sub-flows)

A step can delegate to a sub-flow `.md` file instead of containing its own code block:

```markdown
## 3. Start container

**Depends on:** Step 1
**Include:** [flows/container-start.md](flows/container-start.md)

**Inputs:**

| Name | Source |
|---|---|
| project_id | ${{ steps.1.outputs.project_id }} |
```

The included file is a full scenario with its own steps. Inputs are passed down, outputs bubble up. Resolution is relative to the referencing file. Circular includes are detected and rejected.

### Setup and Teardown

- `## Setup` — runs before the first step. No step number, no dependencies.
- `## Teardown` — runs after the last step completes (or after failure). Guaranteed execution for cleanup.

### Tags

`**Tags:**` in the scenario header enables filtering:

```
synchestra test run --tag e2e
synchestra test run --tag cli,project
```

### File locations

| Location | Purpose |
|---|---|
| `spec/tests/` | Cross-feature E2E scenarios |
| `spec/tests/flows/` | Reusable sub-flows for cross-feature E2E |
| `spec/features/{feature}/_tests/` | Feature-scoped integration tests |
| `spec/features/{feature}/_tests/flows/` | Feature-scoped reusable sub-flows |

## Part 3: The Runner (`pkg/testscenario/`)

### Package structure

```
pkg/
  testscenario/
    parser.go       ← markdown scenario parser
    graph.go        ← dependency graph builder, cycle detection, topological sort
    runner.go       ← step execution, parallelization, input/output passing
    ac.go           ← AC reference resolution, verification script extraction/execution
    include.go      ← sub-flow resolution, recursive inclusion
    types.go        ← Scenario, Step, ACRef, Output, etc.
    reporter.go     ← results collection, pass/fail reporting
```

### CLI commands

```
synchestra test run [path]                          ← run scenario file or directory
synchestra test run spec/tests/                     ← run all cross-feature E2E scenarios
synchestra test run spec/tests/project-lifecycle.md ← run one scenario
synchestra test run --tag e2e                       ← filter by tag
synchestra test list                                ← list available scenarios
synchestra test list --tag e2e                      ← list filtered by tag
```

Follows the existing `synchestra <resource> <action>` command pattern.

### Execution model

1. Parse the scenario markdown into a `Scenario` struct
2. Build a dependency graph from `Depends on` declarations
3. Detect cycles; reject if found
4. Resolve `Include` references recursively (cycle-detected)
5. Run `Setup` block
6. Execute steps in topological order, parallelizing independent steps
7. For each step:
   a. Resolve inputs from upstream outputs
   b. Execute the code block (or delegate to included sub-flow)
   c. Capture stdout, stderr, exit code
   d. Extract declared outputs
   e. Resolve AC references → find AC `.md` files → extract verification scripts
   f. Execute each AC verification script with step inputs + outputs as env vars
   g. Record per-step and per-AC pass/fail
8. Run `Teardown` block (always, even on failure)
9. Report results

### AC execution detail

When a step declares `cli/project/remove/*`:

1. Resolve path: `{spec_root}/features/cli/project/remove/acs/`
2. Read each `.md` file in the directory
3. Extract the `## Verification` code block from each
4. Extract `## Inputs` to validate required inputs are available
5. Pass step inputs + outputs as environment variables
6. Execute each verification script
7. Report per-AC pass/fail

### Future decoupling

The test runner is built as `pkg/testscenario/` with the understanding that it may become a standalone product. Design for this:

- No dependencies on Synchestra-specific packages (other than path resolution via config)
- The spec root path and feature path conventions are configurable, not hardcoded
- The markdown format is generic — not Synchestra-specific

## Part 4: Reserved `_` Prefix Convention

### Formalization

Update the feature spec to include:

> Directories prefixed with `_` are reserved for Synchestra tooling and are not sub-features. They are excluded from the feature index and Contents table.

### Known reserved directories

| Directory | Purpose | Introduced by |
|---|---|---|
| `_args/` | CLI argument documentation | CLI feature |
| `_tests/` | Feature-scoped test scenarios | Test scenario feature |

### Configurable spec root

The spec root directory name is configurable in `synchestra-spec.yaml`:

```yaml
project_dirs:
  specifications: spec    # default: "spec"
```

All path resolution (AC references, scenario includes, feature lookups) uses this configured root. This allows projects to use `specifications/`, `specs/`, or any other name.

## Part 5: Dogfooding — CLI Project Lifecycle

The first scenario to implement, exercising the full system end-to-end:

### ACs to define

For features that are already implemented (`cli/project/new`, `cli/project/list`, etc.), define initial ACs:

```
spec/features/cli/project/new/acs/
  creates-spec-repo.md
  creates-state-repo.md
  returns-project-id.md

spec/features/cli/project/remove/acs/
  not-in-list.md
  recreate-same-id.md
```

### Scenario to create

```
spec/tests/
  README.md
  project-lifecycle.md
  flows/
    README.md
```

`project-lifecycle.md` follows the format from Part 2 — create, list, start container, check status, shutdown, remove.

### Implementation sequence

1. Define AC format and initial ACs for existing features
2. Build the markdown parser (`pkg/testscenario/parser.go`)
3. Build the dependency graph (`pkg/testscenario/graph.go`)
4. Build the step runner with input/output passing (`pkg/testscenario/runner.go`)
5. Build AC resolution and execution (`pkg/testscenario/ac.go`)
6. Build include/sub-flow support (`pkg/testscenario/include.go`)
7. Build the reporter (`pkg/testscenario/reporter.go`)
8. Wire up `synchestra test run` and `synchestra test list` CLI commands
9. Write the project-lifecycle scenario
10. Run it end-to-end

## Outstanding Questions

- Should AC verification scripts be bash-only, or should the format support other interpreters (e.g., `python`, `node`) via a shebang or language annotation on the code block?
- Should the runner support a `--dry-run` mode that parses and validates scenarios without executing them?
- What is the exact reporting format — TAP, JUnit XML, JSON, or a custom markdown report?
- Should the `Scenarios` back-reference table in AC files be manually maintained or auto-generated by the runner?
- How should the runner handle AC verification scripts that require external tools (e.g., `jq`) — should it validate tool availability before execution?
- Should there be a `synchestra ac list` CLI command for querying ACs across features?
