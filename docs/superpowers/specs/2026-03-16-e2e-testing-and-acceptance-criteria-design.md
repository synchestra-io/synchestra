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
- **Inputs/outputs flow like GitHub Actions.** Steps declare outputs to context (global) or step scope, downstream steps consume them by reference.
- **Sequential by default, parallel opt-in.** Steps execute in file order. Steps marked `Parallel: true` form concurrent groups.
- **Composition via inclusion.** Scenarios can reference sub-flow `.md` files, enabling reuse without duplication.
- **The spec root is configurable.** All paths resolve relative to the configured spec root (default: `spec`), supporting projects that use `specifications` or other names.

## Part 1: Acceptance Criteria as First-Class Feature Artifacts

### AC file location and structure

Each AC lives in `spec/features/{feature}/_acs/{ac-slug}.md`:

```markdown
# AC: not-in-list

**Status:** planned
**Feature:** [cli/project/remove](../README.md)

## Description

After project deletion, `synchestra project list` output does not contain
the id of the deleted project.

## Inputs

| Name | Required | Description |
|---|---|---|
| project_id | Yes | ID of the project that was deleted |

Inputs are matched by name against the step's declared Inputs and Outputs. If a required input is not available from the step's context, the runner reports an error for that AC (not a test failure — a configuration error). Optional inputs (marked `No`) default to empty string if not provided.

## Verification

```bash
result=$(synchestra project list --format json)
! echo "$result" | jq -e ".[] | select(.id == \"$project_id\")"
```

## Scenarios

| Scenario | Step |
|---|---|
| [project-lifecycle](../../../tests/project-lifecycle.md) | remove-project |
| [remove-and-recreate](../_tests/remove-and-recreate.md) | delete-and-verify |
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
| [not-in-list](_acs/not-in-list.md) | Deleted project absent from list | implemented |
| [recreate-same-id](_acs/recreate-same-id.md) | Can recreate project with same id | planned |
```

The table is a derived summary — the AC `.md` file is the source of truth.

### Updates to the feature spec

The feature spec's **Required sections** table gains a new entry:

| Section | Required | Notes |
|---|---|---|
| Acceptance Criteria | Yes | Always present. States "Not defined yet." if empty; must also raise an Outstanding Question. |

The feature directory structure gains two new optional directories:

```
spec/features/{feature-slug}/
  README.md                   ← feature specification
  _acs/                       ← acceptance criteria (optional, present when ACs are defined)
    {ac-slug}.md
  _tests/                     ← feature-scoped test scenarios (optional)
    {scenario-slug}.md
    flows/
  proposals/                  ← change requests (optional)
    README.md
    {proposal-slug}/
  {sub-feature-slug}/         ← sub-feature (optional)
    README.md
```

The `_acs/` directory uses the reserved `_` prefix convention, consistent with `_tests/` and `_args/`.

### Relationship to development plan ACs

The development plan spec defines acceptance criteria at two levels: plan-level (cross-cutting) and step-level (per-deliverable). These are **different from feature-level ACs** and serve different purposes:

| AC type | Lives in | Answers | Lifecycle |
|---|---|---|---|
| **Feature AC** | `spec/features/{feature}/_acs/` | "How do we verify this feature works correctly?" | Evolves with the feature; long-lived |
| **Plan-level AC** | `spec/plans/{plan}/README.md` (inline or `_acs/` subdir) | "How do we verify this plan's goals were achieved?" | Frozen with the plan; immutable |
| **Plan step-level AC** | Within each plan step | "How do we verify this step's deliverable?" | Frozen with the plan; immutable |

Plan ACs are scoped to a specific implementation effort and are frozen once the plan is approved. Feature ACs are scoped to the feature itself and evolve over time. A plan step AC may *reference* a feature AC (e.g., "the feature AC `cli/project/remove/not-in-list` must pass after this step"), but they are not the same artifact.

When generating tasks from a plan, both plan step ACs and any referenced feature ACs are copied into the task description, giving agents clear targets.

### Mandatory enforcement

Validation tooling (lint/pre-commit) should check:
- Every feature README has an `## Acceptance Criteria` section
- If the section says "Not defined yet.", the Outstanding Questions section includes the corresponding question
- Every `.md` file in `_acs/` is listed in the feature README table
- Every entry in the feature README table has a corresponding `.md` file in `_acs/`

## Part 2: Test Scenario Format

### Scenario file structure

A test scenario is a markdown file with named steps, dependency declarations, and AC references. Steps execute in file order by default (sequential), with opt-in parallel groups.

```markdown
# Scenario: Project lifecycle

**Description:** End-to-end test of creating, using, and removing a project.
**Tags:** e2e, cli, project

## Setup

```bash
export SYNCHESTRA_HOME=$(mktemp -d)
synchestra config set --key server.host --value localhost
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
synchestra project new --repo https://github.com/example/test --format json
```

## verify-project-in-list

**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/list](spec/features/cli/project/list/) | [in-list](spec/features/cli/project/list/_acs/in-list.md) |

```bash
synchestra project list --format json
```

## start-container

**Include:** [flows/container-start.md](flows/container-start.md)

## check-container-status

**ACs:**

| Feature | ACs |
|---|---|
| [cli/server/project/status](spec/features/cli/server/project/status/) | * |

```bash
synchestra server project status --project ${{ context.project_id }} --format json
```

## shutdown-container

**ACs:**

| Feature | ACs |
|---|---|
| [cli/server/project/shutdown](spec/features/cli/server/project/shutdown/) | * |

```bash
synchestra server project shutdown --project ${{ context.project_id }}
```

## remove-project

**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/remove](spec/features/cli/project/remove/) | * |

```bash
synchestra project remove --id ${{ context.project_id }}
```

## Teardown

```bash
rm -rf $SYNCHESTRA_HOME
```
```

### Step elements

| Element | Required | Description |
|---|---|---|
| Heading | Yes | `## {step-name}` — kebab-case, unique within the scenario |
| Depends on | No | References to other steps by name. Used for data dependencies or ordering constraints beyond file order |
| Parallel | No | `true` to mark this step as part of a concurrent group (see [Execution model](#execution-model)) |
| Outputs | No | Table of outputs with storage scope (`context`, `step`, or `both`) |
| ACs | No | Table of AC references to verify after the step executes |
| Include | Conditional | Delegates to a sub-flow `.md` file. Mutually exclusive with Code block |
| Code block | Conditional | Bash script to execute. Mutually exclusive with Include |

Every step must have exactly one of **Include** or **Code block**. A step with neither is a validation error.

### AC reference syntax

ACs are referenced using a **table format** with two required columns: Feature and ACs. The Feature column links to the feature spec, the ACs column lists which ACs to run (linked to their AC files) or `*` for all.

```markdown
**ACs:**

| Feature | ACs |
|---|---|
| [cli/project/remove](spec/features/cli/project/remove/) | * |
| [cli/project/list](spec/features/cli/project/list/) | [in-list](spec/features/cli/project/list/_acs/in-list.md), [has-metadata](spec/features/cli/project/list/_acs/has-metadata.md) |
```

The runner parses these two columns. Additional columns are allowed for human readability — the runner ignores them:

```markdown
| Feature | ACs | Note |
|---|---|---|
| [cli/project/remove](spec/features/cli/project/remove/) | * | Validates full removal |
```

**Wildcards and selection:**

| Pattern | Meaning |
|---|---|
| `*` | All ACs under that feature's `_acs/` directory |
| `[ac-name](link)` | Specific AC by slug (linked to the AC file) |
| `[ac1](link), [ac2](link)` | Multiple specific ACs, comma-separated in the ACs column |

When specific ACs are listed, they execute in the order specified in the table. When `*` is used, ACs execute in alphabetical order by slug.

### Step naming and references

Steps are identified by their name (the heading text). Names must be:

- **Unique** within the scenario
- **Kebab-case** (lowercase, hyphen-separated)
- **Descriptive** — the name should convey what the step does

References:

- `**Depends on:**` references step names: `**Depends on:** create-project`
- Multiple dependencies use comma separation: `**Depends on:** create-project, add-repo`
- `${{ steps.{step-name}.outputs.{name} }}` for step-scoped outputs
- `${{ context.{name} }}` for context-scoped outputs

### Output model

Steps can store outputs to **context** (global), **step** (local), or **both**:

```markdown
**Outputs:**

| Name | Store | Extract |
|---|---|---|
| project_id | context | `jq -r '.id' $STEP_STDOUT` |
| raw_response | step | `cat $STEP_STDOUT` |
| exit_status | both | `echo $STEP_EXIT_CODE` |
```

| Store | Access syntax | Scope |
|---|---|---|
| `context` | `${{ context.project_id }}` | Available to all subsequent steps. Context names must be unique across the scenario — duplicate writes are a validation error. |
| `step` | `${{ steps.create-project.outputs.raw_response }}` | Available only to steps that declare `Depends on` this step (direct or transitive). |
| `both` | Either syntax | Stored in both scopes. |

Available variables in Extract expressions: `$STEP_STDOUT` (path to stdout file), `$STEP_STDERR` (path to stderr file), `$STEP_EXIT_CODE`.

Outputs (both context and step) are passed to AC verification scripts as environment variables.

### Execution model

Steps execute **sequentially in file order** by default. This is the simplest mental model — read top to bottom, that is the execution order.

**Parallel groups:** Consecutive steps marked `**Parallel:** true` form a parallel group. The group starts after the preceding non-parallel step completes, and the next non-parallel step waits for all steps in the parallel group to finish.

```markdown
## create-project
...

## add-repo-a
**Parallel:** true
...

## add-repo-b
**Parallel:** true
...

## verify-both-repos
...  ← waits for add-repo-a and add-repo-b to complete
```

**`Depends on` within parallel groups:** Steps in a parallel group can declare `Depends on` to establish ordering constraints within the group or reference earlier steps for data dependencies. A parallel step that declares `Depends on` a step outside its group implicitly waits for that dependency regardless of parallelism.

**`Depends on` outside parallel groups:** For sequential steps, `Depends on` is primarily for declaring data dependencies (which step's outputs this step reads). It does not change execution order — file order already guarantees the dependency ran first. The runner validates that any referenced step appears earlier in the file.

### Include (sub-flows)

A step can delegate to a sub-flow `.md` file instead of containing its own code block:

```markdown
## start-container

**Include:** [flows/container-start.md](flows/container-start.md)
```

The included file is a full scenario with its own steps. Context is shared — the sub-flow can read and write to the same `context.*` namespace. Step outputs from the sub-flow are namespaced under the including step's name (e.g., `steps.start-container.outputs.*` exposes the sub-flow's step outputs). Resolution is relative to the referencing file. Circular includes are detected and rejected.

### Setup and Teardown

- `## Setup` — runs before the first step. No step name constraints, no outputs. Purely environment-level: sets env vars, creates temp dirs, starts services. Any state it creates is available to all steps via the shared environment.
- `## Teardown` — runs after the last step completes (or after failure). Guaranteed execution for cleanup. No outputs. Has access to the same environment as Setup (e.g., `$SYNCHESTRA_HOME`).

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
    graph.go        ← step ordering, parallel group detection, validation
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

### Spec root resolution

The runner resolves the spec root from the project's `synchestra-spec.yaml` configuration (`project_dirs.specifications`, default: `spec`). All AC references (e.g., `cli/project/remove/*`) resolve to `{spec_root}/features/{ac_path}/_acs/`. This configuration is read once at runner initialization and passed to the AC resolver.

### Execution model

1. Parse the scenario markdown into a `Scenario` struct
2. Validate: unique step names, no circular includes, no duplicate context keys, `Depends on` references point to earlier steps
3. Resolve `Include` references recursively (cycle-detected)
4. Run `Setup` block
5. Execute steps in file order:
   - **Sequential steps:** execute one at a time in file order
   - **Parallel groups:** consecutive `Parallel: true` steps execute concurrently; the runner waits for all to complete before continuing
   - For each step:
     a. Resolve context and step output references
     b. Execute the code block (or delegate to included sub-flow)
     c. Capture stdout, stderr, exit code
     d. Extract declared outputs, store to context and/or step scope
     e. Resolve AC references → find AC `.md` files → extract verification scripts
     f. Execute each AC verification script with context + step outputs as env vars
     g. Record per-step and per-AC pass/fail
6. Run `Teardown` block (always, even on failure)
7. Report results

### AC execution detail

When a step declares `cli/project/remove/*`:

1. Resolve path: `{spec_root}/features/cli/project/remove/_acs/`
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
spec/features/cli/project/new/_acs/
  creates-spec-repo.md
  creates-state-repo.md
  returns-project-id.md

spec/features/cli/project/remove/_acs/
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
