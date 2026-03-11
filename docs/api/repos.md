# API: Repos

Add and manage repositories linked to projects.

**See also:** [CLI: `synchestra repo`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/repo.md)

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/repos` | [Add repo](#add-repo) |
| `GET` | `/api/v1/repos` | [List repos](#list-repos) |
| `GET` | `/api/v1/repos/:id` | [Get repo](#get-repo) |
| `POST` | `/api/v1/repos/:id/link` | [Link to project](#link-repo) |

---

## Add Repo

`POST /api/v1/repos`

### Request

```json
{
  "url": "https://github.com/my-org/my-api",
  "name": "my-api",
  "project_id": "proj_abc123"
}
```

| Field | Required | Description |
|---|---|---|
| `url` | ✅ | Repository URL |
| `name` | | Display name (defaults to repo slug) |
| `project_id` | | Project to link on creation |

### Response `201`

```json
{
  "id": "repo_def456",
  "url": "https://github.com/my-org/my-api",
  "name": "my-api",
  "projects": ["proj_abc123"],
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: repo add](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/repo.md#add)

---

## List Repos

`GET /api/v1/repos`

### Query parameters

| Param | Description |
|---|---|
| `project_id` | Filter by project |
| `limit` | Max results |
| `cursor` | Pagination cursor |

### Response `200`

```json
{
  "repos": [
    {
      "id": "repo_def456",
      "url": "https://github.com/my-org/my-api",
      "name": "my-api",
      "projects": ["proj_abc123"],
      "created_at": "2024-01-15T09:00:00Z"
    }
  ],
  "next_cursor": null,
  "has_more": false
}
```

**See also:** [CLI: repo list](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/repo.md#list)

---

## Get Repo

`GET /api/v1/repos/:id`

### Response `200`

```json
{
  "id": "repo_def456",
  "url": "https://github.com/my-org/my-api",
  "name": "my-api",
  "projects": ["proj_abc123", "proj_xyz789"],
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: repo get](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/repo.md#get)

---

## Link Repo

`POST /api/v1/repos/:id/link`

Link an existing repo to an additional project. A repo can be linked to multiple projects.

### Request

```json
{
  "project_id": "proj_xyz789"
}
```

| Field | Required | Description |
|---|---|---|
| `project_id` | ✅ | Project to link this repo to |

### Response `200`

```json
{
  "id": "repo_def456",
  "url": "https://github.com/my-org/my-api",
  "name": "my-api",
  "projects": ["proj_abc123", "proj_xyz789"],
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**See also:** [CLI: repo link](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/repo.md#link)
