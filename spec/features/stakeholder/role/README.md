# Feature: Stakeholder / Role

**Status:** Conceptual

## Summary

A role is a named responsibility that maps to one or more [stakeholders](../README.md). Roles are the routing layer for decisions — when a [gate](../gate/README.md) fires or an agent requests a [decision](../decision/README.md), Synchestra resolves the relevant role to determine which stakeholders are assigned.

Roles are defined at the project level and overridden per-feature using `add`/`remove` operations that cascade through the feature hierarchy. This gives projects centralized defaults with fine-grained control where needed.

## Problem

When a development plan needs review or an agent needs input, someone must decide who handles it. Today this is implicit — there is no structured mapping from "this needs a code review" to "these specific people/agents should review it." The result is ad hoc assignment, missed reviews, and no way to enforce that the right people are involved for the right scope.

## Behavior

### Role Definition

Roles are defined in the project configuration (`synchestra-spec-repo.yaml`):

```yaml
roles:
  code-reviewer:
    add:
      - agent-x:model=opus
      - alex@github
  spec-approver:
    add:
      - alex@github
  domain-expert:
    add:
      - carol@github
```

Role names are lowercase, hyphen-separated identifiers. There is no fixed set of role names — projects define whatever roles they need. Common conventions:

| Role | Typical purpose |
|---|---|
| `code-reviewer` | Reviews implementation before merge |
| `spec-approver` | Approves feature specs and development plans |
| `domain-expert` | Provides domain-specific input on decisions |
| `tech-lead` | Final authority on technical direction |
| `product-owner` | Approves feature scope and priorities |

### Feature-Level Overrides

Features can modify inherited role assignments using a `_config.yaml` file in the feature directory:

```yaml
# spec/features/cli/_config.yaml
roles:
  code-reviewer:
    add:
      - bob@github
    remove:
      - agent-x
  spec-approver:
    add:
      - carol@github
```

Operations within a level are applied in order: `add` first, then `remove`.

### Hierarchical Resolution

The effective set of stakeholders for a role at any feature is computed by walking the feature tree from root to target:

1. Start with the project-level role assignments
2. At each feature level from root to target, apply `add` then `remove`
3. Sub-features with no `_config.yaml` (or no role overrides) pass through unchanged

**Example:** resolving `code-reviewer` at `cli/task/claim`

```
Project level:   {agent-x, alex}
  cli/:          +bob, -agent-x  →  {alex, bob}
    task/:       (no override)   →  {alex, bob}
      claim/:    (no override)   →  {alex, bob}
```

**Example:** deep override

```
Project level:        {agent-x, alex}
  api/:               (no override)      →  {agent-x, alex}
    auth/:             +carol            →  {agent-x, alex, carol}
      permissions/:    -agent-x, +dana   →  {alex, carol, dana}
```

### Resolution Rules

- **Adding an already-present stakeholder** is a no-op (idempotent).
- **Removing a stakeholder not in the set** is a no-op (no error).
- **Empty resolved set** is valid — it means no stakeholder holds that role for this feature. A gate requiring this role will create an unassigned decision task that must be manually picked up.

### Future: `set` Operation

A `set` operation that replaces the entire inherited list may be added if `add`/`remove` proves cumbersome for features that need a completely different reviewer set. For now, `add`/`remove` covers the common cases.

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria are not yet defined for this feature.

- Should `_config.yaml` support conditions (e.g., different reviewers for different file types within a feature)?
- Should role definitions support description/purpose metadata for discoverability (e.g., `code-reviewer: { description: "Reviews implementation code", add: [...] }`)?
- How should role resolution work when a feature is referenced by a cross-repo task — does resolution follow the spec repo's feature tree or the code repo's directory structure?
