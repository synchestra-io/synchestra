# API: Projects

Create and manage projects.

**See also:** [CLI: `synchestra project`](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/project.md)

---

## Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/projects` | [Create project](#create-project) |
| `GET` | `/api/v1/projects` | [List projects](#list-projects) |
| `GET` | `/api/v1/projects/:id` | [Get project](#get-project) |
| `PUT` | `/api/v1/projects/:id` | [Update project](#update-project) |
| `DELETE` | `/api/v1/projects/:id` | [Delete project](#delete-project) |

---

## Create Project

`POST /api/v1/projects`

### Request

```json
{
  "name": "e-commerce-platform",
  "description": "Main customer-facing store and admin backend",
  "org_id": "org_xyz789"
}
```

| Field | Required | Description |
|---|---|---|
| `name` | ✅ | Project name (unique within org) |
| `description` | | What this project is about |
| `org_id` | | Organisation to create under |

### Response `201`

```json
{
  "id": "proj_abc123",
  "name": "e-commerce-platform",
  "description": "Main customer-facing store and admin backend",
  "org_id": "org_xyz789",
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: project create](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/project.md#create)

---

## List Projects

`GET /api/v1/projects`

### Query parameters

| Param | Description |
|---|---|
| `org_id` | Filter by organisation |
| `limit` | Max results (default 20, max 100) |
| `cursor` | Pagination cursor |

### Response `200`

```json
{
  "projects": [
    {
      "id": "proj_abc123",
      "name": "e-commerce-platform",
      "description": "Main customer-facing store and admin backend",
      "org_id": "org_xyz789",
      "created_at": "2024-01-15T09:00:00Z"
    }
  ],
  "next_cursor": null,
  "has_more": false
}
```

**See also:** [CLI: project list](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/project.md#list)

---

## Get Project

`GET /api/v1/projects/:id`

### Response `200`

```json
{
  "id": "proj_abc123",
  "name": "e-commerce-platform",
  "description": "Main customer-facing store and admin backend",
  "org_id": "org_xyz789",
  "created_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T09:00:00Z"
}
```

**See also:** [CLI: project get](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/project.md#get)

---

## Update Project

`PUT /api/v1/projects/:id`

### Request

```json
{
  "name": "e-commerce-platform-v2",
  "description": "Rewritten with microservices architecture"
}
```

All fields optional. Only provided fields are updated.

### Response `200`

Returns the updated project object.

**See also:** [CLI: project update](https://github.com/synchestra-io/synchestra-go/blob/main/docs/cli/project.md#update)

---

## Delete Project

`DELETE /api/v1/projects/:id`

Deletes the project. Associated tasks are preserved but unlinked.

### Response `204`

No content.
