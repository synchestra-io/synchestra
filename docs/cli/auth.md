# CLI: `synchestra auth`

Create and manage API tokens for authenticating requests to the Synchestra server.

**See also:** [API: Auth](../api/auth.md) | [Self-Hosting: Auth config](../self-hosting.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [token create](#token-create) | Create a new API token |
| [token list](#token-list) | List existing tokens |
| [token revoke](#token-revoke) | Revoke a token |

---

## token create

Create a new API token. The token value is shown once at creation time — store it securely.

```
synchestra auth token create [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--name` | ✅ | Human-readable name for this token |
| `--scopes` | | Comma-separated list of scopes. Defaults to all scopes. |

### Available scopes

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

### Examples

```bash
# Full-access token (for human operators)
synchestra auth token create --name "my-admin-token"

# Output:
# Token created: tok_abc123xyz...
# ⚠ This token will not be shown again. Store it securely.

# Restricted token for a coder agent
synchestra auth token create \
  --name "coder-agent-token" \
  --scopes "tasks:read,tasks:write,agents:write"

# Read-only token for monitoring dashboard
synchestra auth token create \
  --name "dashboard-readonly" \
  --scopes "tasks:read,agents:read,projects:read"

# Capture token in a script
TOKEN=$(synchestra auth token create --name "ci-token" \
  --scopes "tasks:write" --output json | jq -r .token)
```

**See also:** [POST /api/v1/auth/tokens](../api/auth.md#create-token)

---

## token list

List all API tokens. Token values are never shown after creation.

```
synchestra auth token list [flags]
```

### Examples

```bash
synchestra auth token list
```

Output:

```
ID              Name                  Scopes              Created              Last used
tok_abc123      my-admin-token        admin               2024-01-10 09:00     2024-01-15 14:32
tok_def456      coder-agent-token     tasks:rw,agents:w   2024-01-12 11:00     2024-01-15 14:30
tok_ghi789      dashboard-readonly    tasks:r,agents:r    2024-01-13 15:00     2024-01-15 08:00
```

```bash
synchestra auth token list --output json
```

**See also:** [GET /api/v1/auth/tokens](../api/auth.md#list-tokens)

---

## token revoke

Revoke a token. All subsequent requests using this token will be rejected with `401 Unauthorized`.

```
synchestra auth token revoke <id>
```

### Examples

```bash
synchestra auth token revoke tok_def456

# With confirmation prompt bypass
synchestra auth token revoke tok_def456 --yes
```

**See also:** [DELETE /api/v1/auth/tokens/:id](../api/auth.md#revoke-token)
