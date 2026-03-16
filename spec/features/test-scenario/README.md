# Feature: Test Scenario

**Status:** Conceptual

## Summary

A markdown-native test scenario format and runner for composing acceptance criteria into multi-step integration and E2E test flows. Scenarios are human-readable `.md` files with named steps, dependency declarations, input/output passing, and AC references. The runner executes steps sequentially (with opt-in parallel groups), resolves AC verification scripts from feature specs, and reports pass/fail results.

## Problem

Synchestra has feature specs with behavioral descriptions and development plans with acceptance criteria, but no structured way to:

- **Verify features work end-to-end.** Individual Go tests validate package-level behavior, but there is no harness for black-box testing of the compiled CLI binary across a multi-step workflow (create project → add repo → start container → verify → shut down → remove).
- **Reuse verification logic.** The same assertion ("deleted project is not in list") appears in multiple contexts — a feature-scoped test, an E2E lifecycle test, a regression suite. Without a reusable unit, each test re-implements the check.
- **Connect specs to tests.** Acceptance criteria in plans and features are prose today. There is no executable link from "this feature should do X" to "this script verifies X."

## Behavior

### Acceptance criteria as first-class artifacts

Each feature can define acceptance criteria in an `_acs/` subdirectory. Each AC is a separate `.md` file that serves as the source of truth:

```
spec/features/{feature}/_acs/{ac-slug}.md
```

AC files contain: status, description, typed inputs (required/optional), a bash verification script, and back-references to scenarios that use the AC.

The feature README includes a mandatory **Acceptance Criteria** section — a table summarizing all ACs with links. If no ACs are defined yet, the section states "Not defined yet." and an Outstanding Question is raised.

### AC statuses

| Status | Description |
|---|---|
| `planned` | AC is described but has no verification script yet |
| `wip` | Verification script is being written/tested |
| `implemented` | Verification script exists and passes |
| `deprecated` | AC is no longer relevant |

### Relationship to development plan ACs

Feature ACs and plan ACs are different artifacts:

| AC type | Lives in | Lifecycle |
|---|---|---|
| Feature AC | `spec/features/{feature}/_acs/` | Evolves with the feature; long-lived |
| Plan AC | `spec/plans/{plan}/` (inline or `_acs/` subdir) | Frozen with the plan; immutable |

Plan step ACs may *reference* feature ACs, but they are not the same artifact.

### Test scenario format

A test scenario is a markdown file with named steps:

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

### Step elements

| Element | Description |
|---|---|
| Heading | `## {step-name}` — kebab-case, unique within the scenario |
| Depends on | References to other steps by name |
| Parallel | `true` to mark this step as part of a concurrent group |
| Outputs | Table with Name, Store (`context`/`step`/`both`), and Extract columns |
| ACs | Table with Feature and ACs columns |
| Include | Delegates to a sub-flow `.md` file |
| Code block | Bash script to execute |

### Execution model

Steps execute **sequentially in file order** by default. Consecutive steps marked `**Parallel:** true` form a parallel group — the group starts after the preceding step completes, and the next non-parallel step waits for the group to finish.

### Output model

Steps store outputs to **context** (global, `${{ context.name }}`), **step** (local, `${{ steps.{name}.outputs.{name} }}`), or **both**. Context names must be unique across the scenario.

### AC reference syntax

ACs are referenced via a table with Feature and ACs columns. The ACs column accepts `*` (all) or specific AC names as markdown links. The runner only parses the first two columns; additional columns are for human readability.

### Include (sub-flows)

A step can delegate to a sub-flow `.md` file. The included file is a full scenario with its own steps. Context is shared; step outputs are namespaced. Circular includes are detected and rejected.

### File locations

| Location | Purpose |
|---|---|
| `spec/tests/` | Cross-feature E2E scenarios |
| `spec/tests/flows/` | Reusable sub-flows |
| `spec/features/{feature}/_tests/` | Feature-scoped integration tests |

### CLI commands

```
synchestra test run [path]       — run scenario file or directory
synchestra test run --tag e2e    — filter by tag
synchestra test list             — list available scenarios
synchestra test list --tag e2e   — list filtered by tag
```

### Configurable spec root

The spec root directory name is configurable via `project_dirs.specifications` in `synchestra-spec.yaml` (default: `spec`). All path resolution uses this configured root.

## Interaction with Other Features

| Feature | Interaction |
|---|---|
| [Feature](../feature/README.md) | Features gain a mandatory Acceptance Criteria section and `_acs/` directory convention |
| [Development Plan](../development-plan/README.md) | Plan step ACs may reference feature ACs |
| [CLI](../cli/README.md) | New `synchestra test` command group: `run`, `list` |

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria not yet defined for this feature.
- Should AC verification scripts be bash-only, or should the format support other interpreters via a shebang or language annotation?
- Should the runner support a `--dry-run` mode that parses and validates scenarios without executing them?
- What is the exact reporting format — TAP, JUnit XML, JSON, or a custom text report?
- Should the `Scenarios` back-reference table in AC files be manually maintained or auto-generated by the runner?
