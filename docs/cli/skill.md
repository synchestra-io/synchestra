# CLI: `synchestra skill`

Define and manage skills — named capability definitions that agents declare and tasks can require.

**See also:** [API: Skills](../api/skills.md) | [Feature: Agent Coordination](../features/agent-coordination.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [create](#create) | Create a skill definition |
| [list](#list) | List skills |
| [get](#get) | Get skill details |

---

## create

Create a skill definition. Skills are how Synchestra knows what an agent can do and what a task requires.

```
synchestra skill create [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--name` | ✅ | Skill identifier (e.g. `go`, `typescript`, `code-review`) |
| `--description` | | What this skill means in practice |
| `--input-schema` | | JSON Schema describing the expected input for this skill |
| `--output-schema` | | JSON Schema describing the expected output for this skill |

### Examples

```bash
# Simple skill
synchestra skill create \
  --name "go" \
  --description "Can write, test, refactor, and build Go code"

# Skill with schemas (useful for strict routing)
synchestra skill create \
  --name "code-review" \
  --description "Reviews pull requests and provides actionable feedback" \
  --input-schema '{
    "type": "object",
    "required": ["pr_url"],
    "properties": {
      "pr_url": {"type": "string", "description": "GitHub PR URL to review"},
      "focus_areas": {"type": "array", "items": {"type": "string"}}
    }
  }' \
  --output-schema '{
    "type": "object",
    "properties": {
      "approved": {"type": "boolean"},
      "comments": {"type": "array", "items": {"type": "string"}},
      "blocking_issues": {"type": "integer"}
    }
  }'
```

**See also:** [POST /api/v1/skills](../api/skills.md#create-skill)

---

## list

List all defined skills.

```
synchestra skill list [flags]
```

### Examples

```bash
synchestra skill list

synchestra skill list --output json
```

**See also:** [GET /api/v1/skills](../api/skills.md#list-skills)

---

## get

Get details for a specific skill.

```
synchestra skill get <id> [flags]
```

### Examples

```bash
synchestra skill get skill_abc123

synchestra skill get skill_abc123 --output json
```

**See also:** [GET /api/v1/skills/:id](../api/skills.md#get-skill)
