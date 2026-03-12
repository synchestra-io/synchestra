# API: Auth

Create and manage API tokens.

**See also:** [CLI: `synchestra auth`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/auth.md)

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/auth/tokens` | [Create token](#create-token) |
| `GET` | `/api/v1/auth/tokens` | [List tokens](#list-tokens) |
| `DELETE` | `/api/v1/auth/tokens/:id` | [Revoke token](#revoke-token) |

---

## Create Token

`POST /api/v1/auth/tokens`

Creates a new API token. The `token` value is returned once only —  it is never retrievable after creation.

### Request

```json
{
  "name": "coder-agent-token",
  "scopes": ["tasks:read", "tasks:write", "agents:write"]
}
```

| Field | Required | Description |
|---|---|---|
| `name` | ✅ | Human-readable name for this token |
| `scopes` | | Array of scopes. Defaults to all scopes (`["admin"]`) if omitted. |

### Available Scopes

| Scope | Access |
|---|---|
| `tasks:read` | Read tasks and history |
| `tasks:write` | Create, update, and complete tasks |
| `agents:read` | Read agent information |
| `agents:write` | Register, update, and deregister agents |
| `projects:read` | Read projects |
| `projects:write` | Create and update projects |
| `repos:read` | Read repos |
| `repos:write` | Add and link repos |
| `skills:read` | Read skills |
| `skills:write` | Create skills |
| `rules:read` | Read rules |
| `rules:write` | Create rules |
| `admin` | Full access (includes all above) |

### Response `201`

```json
{
  "id": "tok_abc123",
  "name": "coder-agent-token",
  "token": "tok_abc123_secretvalue_neveragain",
  "scopes": ["tasks:read", "tasks:write", "agents:write"],
  "created_at": "2024-01-15T09:00:00Z",
  "last_used_at": null
}
```

> ⚠️ The `token` field is only present in this response. Store it securely.

**See also:** [CLI: auth token create](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/auth.md#token-create)

---

## List Tokens

`GET /api/v1/auth/tokens`

Lists all tokens. Token values are never returned.

### Response `200`

```json
{
  "tokens": [
    {
      "id": "tok_abc123",
      "name": "coder-agent-token",
      "scopes": ["tasks:read", "tasks:write", "agents:write"],
      "created_at": "2024-01-15T09:00:00Z",
      "last_used_at": "2024-01-15T14:32:00Z"
    },
    {
      "id": "tok_def456",
      "name": "dashboard-readonly",
      "scopes": ["tasks:read", "agents:read", "projects:read"],
      "created_at": "2024-01-13T15:00:00Z",
      "last_used_at": "2024-01-15T08:00:00Z"
    }
  ]
}
```

**See also:** [CLI: auth token list](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/auth.md#token-list)

---

## Revoke Token

`DELETE /api/v1/auth/tokens/:id`

Revokes a token. All subsequent requests using this token will be rejected with `401 Unauthorized`. This is irreversible.

### Response `204`

No content.

**See also:** [CLI: auth token revoke](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/auth.md#token-revoke)
