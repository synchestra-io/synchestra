# Feature: Stakeholder / Gate

**Status:** Conceptual

## Summary

A gate is a named decision point in a workflow where one or more [stakeholders](../README.md) must weigh in before the process continues. Gates connect Synchestra's workflow transitions (plan status changes, task completions) to [decisions](../decision/README.md) routed through [roles](../role/README.md).

Synchestra ships with built-in gates for the most common decision points. The data model is designed to support user-defined custom gates in the future.

## Problem

Synchestra has implicit approval points — development plans transition through `in_review → approved`, code gets reviewed before merge. But nothing enforces these checkpoints or routes them to the right people. A plan can be approved by anyone, a code review can be skipped entirely, and there is no way to require that specific roles sign off before a workflow proceeds.

## Behavior

### Built-in Gates

| Gate | Trigger | Default role | Default policy |
|---|---|---|---|
| `plan-review` | Plan status → `in_review` | `spec-approver` | `all` |
| `code-review` | Implementation task → `completed` | `code-reviewer` | `min: 1` |

`plan-review` covers the full plan approval cycle: when a plan enters `in_review`, a decision is created for spec approvers. If they approve, the plan transitions to `approved`. If they reject or provide custom feedback, the plan returns to `draft` for revision.

Built-in gates fire automatically when their trigger condition is met. They create [decision](../decision/README.md) tasks assigned to the resolved stakeholders for the relevant feature scope.

### Gate Configuration

Gates are configured in the project configuration (`synchestra-spec-repo.yaml`), alongside [role definitions](../role/README.md). Both `roles:` and `gates:` are top-level keys in the project configuration file. Built-in gates have sensible defaults but are fully overridable:

```yaml
gates:
  code-review:
    requires: [code-reviewer]
    policy: min
    min: 2
    options:
      type: approve-reject
      allow_custom: true

  plan-review:
    requires: [spec-approver, domain-expert]
    policy: all
    options:
      type: approve-reject
      allow_custom: true
```

### Configuration Fields

| Field | Required | Description |
|---|---|---|
| `requires` | Yes | List of [role](../role/README.md) names. Stakeholders are resolved from these roles at the relevant feature scope. |
| `policy` | Yes | Resolution policy: `all`, `any`, `min`, `majority` |
| `min` | If policy is `min` | Minimum number of responses required. Error if fewer stakeholders are resolved. |
| `options` | No | Default [options](../decision/options/README.md) for decisions created by this gate. If omitted, defaults to `approve-reject` with `allow_custom: true`. |

### Resolution Policies

| Policy | Behavior |
|---|---|
| `all` | Every resolved stakeholder must respond |
| `any` | First response satisfies the gate |
| `min: N` | At least N stakeholders must respond |
| `majority` | More than half of resolved stakeholders must respond |

`min: 1` is equivalent to `any`. `min` equal to the resolved set size is equivalent to `all`. The `min` policy is the general case; `all`, `any`, and `majority` are readable shortcuts.

If `min` exceeds the number of resolved stakeholders, the gate creation fails with an error — the project configuration is invalid.

### Multiple Roles

When a gate requires multiple roles, stakeholders are resolved for each role independently. The policy applies to the combined set:

```yaml
gates:
  spec-review:
    requires: [spec-approver, domain-expert]
    policy: all
```

If `spec-approver` resolves to `{alex}` and `domain-expert` resolves to `{carol}`, the gate requires responses from both Alex and Carol.

If a stakeholder appears in multiple roles (e.g., Alex is both `spec-approver` and `domain-expert`), they appear once in the combined set — one response satisfies both roles.

### Disabling a Gate

A built-in gate can be disabled at the project level:

```yaml
gates:
  code-review:
    enabled: false
```

This is useful for small projects or early-stage work where formal review overhead is not yet warranted.

### Future: Custom Gates

The data model supports user-defined gates with arbitrary trigger conditions. The initial implementation ships only the built-in gates; the trigger expression language is a future enhancement that will be designed once real usage patterns clarify what conditions people actually need.

A custom gate would look like:

```yaml
gates:
  security-review:
    trigger: "task.labels contains 'security'"
    requires: [security-lead]
    policy: all
    options:
      type: approve-reject
      allow_custom: true
```

The `trigger` field is reserved but not implemented. Built-in gates use hardcoded trigger logic.

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria are not yet defined for this feature.

- Should gates be configurable per-feature (e.g., CLI features require 2 code reviewers, UI features require 1), or only at the project level?
- Should there be a `bypass` mechanism for emergencies — e.g., a project owner can force-approve past a gate with an audit trail entry?
- How should gates interact with the fast-path in [chat workflows](../../chat/workflow/README.md) — should maintainer fast-path skip gates, or should gates apply regardless?
