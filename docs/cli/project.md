# CLI: `synchestra project`

Create and manage projects — containers for related work.

**See also:** [API: Projects](../api/projects.md)

---

## Commands

| Subcommand | Description |
|---|---|
| [create](#create) | Create a new project |
| [list](#list) | List projects |
| [get](#get) | Get project details |
| [update](#update) | Update a project |

---

## create

Create a new project.

```
synchestra project create [flags]
```

### Flags

| Flag | Required | Description |
|---|---|---|
| `--name` | ✅ | Project name (unique within the org) |
| `--description` | | What this project is about |
| `--org` | | Organisation ID or name to create the project under |

### Examples

```bash
synchestra project create --name "my-api"

synchestra project create \
  --name "e-commerce-platform" \
  --description "Main customer-facing store and admin backend" \
  --org my-org

PROJECT_ID=$(synchestra project create --name "my-api" --output json | jq -r .id)
```

**See also:** [POST /api/v1/projects](../api/projects.md#create-project)

---

## list

List all projects.

```
synchestra project list [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--org` | Filter by organisation ID or name |

### Examples

```bash
synchestra project list

synchestra project list --org my-org

synchestra project list --output json
```

**See also:** [GET /api/v1/projects](../api/projects.md#list-projects)

---

## get

Get details for a specific project.

```
synchestra project get <id> [flags]
```

### Examples

```bash
synchestra project get proj_abc123

synchestra project get proj_abc123 --output json
```

**See also:** [GET /api/v1/projects/:id](../api/projects.md#get-project)

---

## update

Update a project's name or description.

```
synchestra project update <id> [flags]
```

### Flags

| Flag | Description |
|---|---|
| `--name` | New name for the project |
| `--description` | New description |

### Examples

```bash
synchestra project update proj_abc123 --name "my-api-v2"

synchestra project update proj_abc123 \
  --description "Rewritten API with gRPC support and new auth layer"
```

**See also:** [PUT /api/v1/projects/:id](../api/projects.md#update-project)
