# API: Skills

Define and query skill definitions.

**See also:** [CLI: `synchestra skill`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/skill.md) | [Feature: Agent Coordination](../features/agent-coordination.md)

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/skills` | [Create skill](#create-skill) |
| `GET` | `/api/v1/skills` | [List skills](#list-skills) |
| `GET` | `/api/v1/skills/:id` | [Get skill](#get-skill) |

---

## Create Skill

`POST /api/v1/skills`

### Request

```json
{
  "name": "code-review",
  "description": "Reviews pull requests and provides actionable, constructive feedback",
  "input_schema": {
    "type": "object",
    "required": ["pr_url"],
    "properties": {
      "pr_url": {
        "type": "string",
        "description": "GitHub PR URL to review"
      },
      "focus_areas": {
        "type": "array",
        "items": {"type": "string"},
        "description": "Optional areas to focus review on"
      }
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "approved": {"type": "boolean"},
      "summary": {"type": "string"},
      "blocking_issues": {"type": "integer"},
      "suggestions": {"type": "array", "items": {"type": "string"}}
    }
  }
}
```

| Field | Required | Description |
|---|---|---|
| `name` | ✅ | Skill identifier (e.g. `go`, `code-review`) |
| `description` | | Human-readable description |
| `input_schema` | | JSON Schema for skill input |
| `output_schema` | | JSON Schema for skill output |

### Response `201`

```json
{
  "id": "skill_abc123",
  "name": "code-review",
  "description": "Reviews pull requests and provides actionable, constructive feedback",
  "input_schema": { ... },
  "output_schema": { ... },
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: skill create](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/skill.md#create)

---

## List Skills

`GET /api/v1/skills`

### Response `200`

```json
{
  "skills": [
    {
      "id": "skill_abc123",
      "name": "code-review",
      "description": "Reviews pull requests and provides actionable, constructive feedback",
      "created_at": "2024-01-15T09:00:00Z"
    },
    {
      "id": "skill_def456",
      "name": "go",
      "description": "Can write, test, refactor, and build Go code",
      "created_at": "2024-01-14T09:00:00Z"
    }
  ],
  "next_cursor": null,
  "has_more": false
}
```

**See also:** [CLI: skill list](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/skill.md#list)

---

## Get Skill

`GET /api/v1/skills/:id`

### Response `200`

```json
{
  "id": "skill_abc123",
  "name": "code-review",
  "description": "Reviews pull requests and provides actionable, constructive feedback",
  "input_schema": {
    "type": "object",
    "required": ["pr_url"],
    "properties": {
      "pr_url": {"type": "string"},
      "focus_areas": {"type": "array", "items": {"type": "string"}}
    }
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "approved": {"type": "boolean"},
      "summary": {"type": "string"},
      "blocking_issues": {"type": "integer"}
    }
  },
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: skill get](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/skill.md#get)
