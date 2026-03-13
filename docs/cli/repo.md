# CLI: `synchestra repo`

Add and manage repositories linked to projects.

**See also:** [API: Repos](../api/repos.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [add](#add) | Add a repository |
| [list](#list) | List repos |
| [get](#get) | Get repo details |
| [link](#link) | Link a repo to a project |

---

## add

Add a repository to Synchestra.

```
synchestra repo add [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--url` | ✅ | Repository URL (e.g. `https://github.com/org/repo`) |
| `--name` | | Display name (defaults to repo slug) |
| `--project` | | Project ID or name to link the repo to on creation |

### Examples

```bash
synchestra repo add --url https://github.com/my-org/my-api

synchestra repo add \
  --url https://github.com/my-org/shared-libs \
  --name "shared-libraries" \
  --project my-api

REPO_ID=$(synchestra repo add --url https://github.com/my-org/my-api --output json | jq -r .id)
```

**See also:** [POST /api/v1/repos](../api/repos.md#add-repo)

---

## list

List all repos, optionally filtered by project.

```
synchestra repo list [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--project` | Filter by project ID or name |

### Examples

```bash
synchestra repo list

synchestra repo list --project my-api

synchestra repo list --output json
```

**See also:** [GET /api/v1/repos](../api/repos.md#list-repos)

---

## get

Get details for a specific repo.

```
synchestra repo get <id> [flags]
```

### Examples

```bash
synchestra repo get repo_abc123
```

**See also:** [GET /api/v1/repos/:id](../api/repos.md#get-repo)

---

## link

Link an existing repo to a project. A repo can be linked to multiple projects.

```
synchestra repo link <id> [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--project` | ✅ | Project ID or name to link to |

### Examples

```bash
# Link shared-libs repo to multiple projects
synchestra repo link repo_abc123 --project my-api
synchestra repo link repo_abc123 --project my-frontend
```

**See also:** [POST /api/v1/repos/:id/link](../api/repos.md#link-repo)
