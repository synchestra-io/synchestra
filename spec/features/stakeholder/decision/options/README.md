# Feature: Stakeholder / Decision / Options

**Status:** Conceptual

## Summary

The structured format for presenting choices to stakeholders in a [decision](../README.md) task. Options define what a stakeholder can choose from, how choices are identified programmatically, and how they should be displayed. The format gives UIs enough information to render appropriate controls without prescribing specific implementations.

## Problem

When AI agents present choices, they typically output unstructured text — "Here are 3 options: A) Go left, B) Go right, C) Go straight." This works in a conversational context but fails for async decisions:

- No machine-readable way to record which option was chosen
- No way for a UI to render buttons or selects
- No distinction between the option identifier and its display text
- No way to indicate whether the stakeholder can provide an alternative

## Behavior

### Option Type

The `type` field in the decision's frontmatter `options` block indicates what kind of response is expected:

| Type | Behavior |
|---|---|
| `pick-one` | Stakeholder selects exactly one option |
| `pick-many` | Stakeholder selects one or more options |
| `approve-reject` | Shorthand for `pick-one` with items `[{key: approve}, {key: reject}]` |
| `free-text` | No predefined options — stakeholder provides open-ended input |

### Option Items

Each item has a `key` and an optional `label`:

```yaml
options:
  type: pick-one
  items:
    - key: jwt
      label: "JWT bearer tokens"
    - key: api-key
      label: "Static API keys"
    - key: oauth
      label: "OAuth 2.0 with PKCE"
  allow_custom: true
```

| Field | Required | Description |
|---|---|---|
| `key` | Yes | Machine-readable identifier, recorded in [audit log](../audit/README.md). Lowercase, hyphen-separated. |
| `label` | No | Human-readable display text. If absent, `key` is displayed. |

The `key` is what gets stored, compared, and used programmatically. The `label` is only for display.

### `allow_custom`

Controls whether the stakeholder can provide a free-text response instead of picking from the listed options.

- `true` (default) — stakeholder can pick a listed option OR provide custom text. UI renders the options plus a text input field.
- `false` — constrained to listed options only. Use when choices are exhaustive (approve/reject, pick a license, select from enum values).

When `type` is `free-text`, `allow_custom` is implicitly `true` and `items` is empty.

### UI Rendering Hints

The options format does not prescribe UI layout. Instead, UIs infer the appropriate control from the option metadata:

| Condition | Suggested rendering |
|---|---|
| `approve-reject` | Green/red button pair |
| `pick-one`, 2-3 items, no long labels | Side-by-side buttons or pill group |
| `pick-one`, 4+ items or long labels | Radio group or select dropdown |
| `pick-many` | Checkbox group |
| `free-text` or `allow_custom` | Text input/textarea |
| `pick-one` + `allow_custom` | Options list with "Other" text input |

These are suggestions, not requirements. Each UI surface (web app, TUI, bot) adapts to its constraints.

### Examples

**Simple approve/reject:**

```yaml
options:
  type: approve-reject
  allow_custom: true
```

Renders as approve/reject buttons with an optional text field for comments.

**Short option list:**

```yaml
options:
  type: pick-one
  items:
    - key: a
    - key: b
    - key: c
  allow_custom: false
```

Renders as three buttons labeled A, B, C. No free-text escape.

**Labeled options with escape hatch:**

```yaml
options:
  type: pick-one
  items:
    - key: jwt
      label: "JWT bearer tokens"
    - key: api-key
      label: "Static API keys"
    - key: oauth
      label: "OAuth 2.0 with PKCE"
  allow_custom: true
```

Renders as a list/radio group with labels, plus an "Other" text input.

**Multi-select:**

```yaml
options:
  type: pick-many
  items:
    - key: unit-tests
      label: "Unit tests"
    - key: integration-tests
      label: "Integration tests"
    - key: e2e-tests
      label: "End-to-end tests"
  allow_custom: true
```

Renders as checkboxes with an optional text field.

## Acceptance Criteria

Not defined yet.

## Outstanding Questions

- Acceptance criteria are not yet defined for this feature.

- Should options support a `default` field to pre-select a recommended choice?
- Should there be a `description` field per option for cases where even the label is too short to convey meaning, or is that always better handled in the decision's markdown body?
- Should `pick-many` support `min`/`max` constraints on how many items can be selected?
