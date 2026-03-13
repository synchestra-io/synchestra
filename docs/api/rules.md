# API: Rules

Create and query rules that shape agent behaviour within a scope.

**See also:** [CLI: `synchestra rule`](../cli/rule.md) | [Feature: Human Steering](../features/human-steering.md)

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/rules` | [Create rule](#create-rule) |
| `GET` | `/api/v1/rules` | [List rules](#list-rules) |
| `GET` | `/api/v1/rules/:id` | [Get rule](#get-rule) |

---

## Create Rule

`POST /api/v1/rules`

### Request

```json
{
  "name": "no-direct-prod-deploy",
  "content": "Never deploy directly to production. Always deploy to staging first, run smoke tests, then request human approval before proceeding to production.",
  "scope": "project",
  "scope_id": "proj_abc123"
}
```

| Field | Required | Description |
|---|---|---|
| `name` | ✅ | Short rule name |
| `content` | ✅ | Rule content — a clear, actionable instruction |
| `scope` | ✅ | `human`, `org`, `project`, or `repo` |
| `scope_id` | ✅ | ID of the scoped resource |

### Response `201`

```json
{
  "id": "rule_abc123",
  "name": "no-direct-prod-deploy",
  "content": "Never deploy directly to production. Always deploy to staging first, run smoke tests, then request human approval before proceeding to production.",
  "scope": "project",
  "scope_id": "proj_abc123",
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: rule create](../cli/rule.md#create)

---

## List Rules

`GET /api/v1/rules`

### Query parameters

| Param | Description |
|---|---|
| `scope` | Filter by scope type: `human`, `org`, `project`, `repo` |
| `scope_id` | Filter by specific scope ID |
| `limit` | Max results |
| `cursor` | Pagination cursor |

### Example

```bash
# Get all rules for a project
curl "http://localhost:8080/api/v1/rules?scope=project&scope_id=proj_abc123" \
  -H "Authorization: Bearer $TOKEN"
```

### Response `200`

```json
{
  "rules": [
    {
      "id": "rule_abc123",
      "name": "no-direct-prod-deploy",
      "content": "Never deploy directly to production...",
      "scope": "project",
      "scope_id": "proj_abc123",
      "created_at": "2024-01-15T09:00:00Z"
    },
    {
      "id": "rule_def456",
      "name": "test-coverage-required",
      "content": "All new functions must have at least one unit test...",
      "scope": "project",
      "scope_id": "proj_abc123",
      "created_at": "2024-01-14T11:00:00Z"
    }
  ],
  "next_cursor": null,
  "has_more": false
}
```

**See also:** [CLI: rule list](../cli/rule.md#list)

---

## Get Rule

`GET /api/v1/rules/:id`

### Response `200`

```json
{
  "id": "rule_abc123",
  "name": "no-direct-prod-deploy",
  "content": "Never deploy directly to production. Always deploy to staging first, run smoke tests, then request human approval before proceeding to production.",
  "scope": "project",
  "scope_id": "proj_abc123",
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: rule get](../cli/rule.md#get)
