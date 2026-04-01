# Design: User & Project Skills

**Date:** 2026-04-01  
**Status:** Approved  
**Scope:** User skill collections, project skill management, agent discovery at runtime

## Problem

Currently, Synchestra agents rely on skills published globally via the Synchestra repository or MCP servers. Users have no way to:
- Create and organize personal skill collections for reuse across projects
- Manage project-specific skills that override or extend defaults
- Discover skills tailored to their workflow via a familiar Hub UI
- Ensure skills are available to agents at execution time

## Solution

A two-tier skill system: **User Skills** (personal collection in Firestore, managed in Hub) and **Project Skills** (project-specific, git-backed in `synchestra/skills/`). At runtime, the Synchestra runner copies all project skills to `.claude/skills/` so agents discover and use them seamlessly.

Design is forward-compatible with a future paid feature: user-backed git repositories for version control and collaboration.

## System Architecture

```
┌─────────────────┐
│   Hub UI        │  User manages skills & projects
│  (Angular)      │
└────────┬────────┘
         │
    ┌────┴────┐
    │          │
    ▼          ▼
┌──────────────┐  ┌──────────────────┐
│  Firestore   │  │  Project Repo    │
│ User Skills  │  │  synchestra/     │
│              │  │  skills/         │
└──────────────┘  └──────────────────┘
                         │
                         ▼
                  ┌──────────────┐
                  │   Runner     │  Clone, copy skills
                  │   (Agent)    │
                  └──────┬───────┘
                         │
                         ▼
                  ┌──────────────┐
                  │ .claude/     │
                  │ skills/      │  Agent discovers
                  └──────────────┘
```

## 1. User Skills Collection (Firestore)

### Scope

Personal skill library, cloud-backed, discoverable and manageable in Hub UI. Users save skills they create or import, then reference them when adding to projects.

### Firestore Schema

```
users/{user_id}/skills/{skill_id}
  skill_id:   string               // unique identifier (e.g., "code-review")
  name:       string               // display name
  description: string              // one-line description
  content:    string               // full SKILL.md file content
  created_at: timestamp            // when skill was created
  updated_at: timestamp            // when skill was last modified
  metadata: {
    tags:     array<string>        // optional: ["testing", "go", "frontend", ...]
    version:  string               // optional: semantic version
  }
```

### Hub UI: Skills Dashboard

**Location:** Hub → User Menu → "My Skills" (new section)

**Features:**

| Feature | Description |
|---------|-------------|
| **List** | Display all user skills with name, description, tag badges, created/updated dates |
| **Search** | Full-text search on name, description, tags |
| **Create** | Modal with text editor to write new SKILL.md inline |
| **Import** | Paste raw SKILL.md or provide GitHub URL to import skill content |
| **Edit** | Modify skill content, name, tags, description |
| **Delete** | Remove skill from collection (soft delete or hard delete; confirm before removing) |
| **Export** | (Future) Download SKILL.md file |
| **View Usage** | (Future) Show which projects use this skill |

### Skill Lifecycle

1. **Create:** User writes skill in Hub editor or imports from URL/file → saved to Firestore
2. **Reuse:** User adds skill to projects → copied to project's `synchestra/skills/`
3. **Save from Project:** User creates skill in project context, saves to favorites → copied to Firestore
4. **Update:** User modifies skill in user collection → existing project copies are not auto-updated (immutable at project level)

## 2. Project Skills (Project Repo)

### Scope

Skills used by a specific project, git-versioned, authoritative at runtime. Users add skills from their collection, create new ones on-the-fly, or save existing project skills back to their collection.

### Directory Structure

```
project-repo/
  synchestra/
    skills/
      {skill_id}/
        SKILL.md           ← skill definition with frontmatter
      code-review/
        SKILL.md
      go-testing/
        SKILL.md
      my-custom-skill/
        SKILL.md
```

### SKILL.md Format

```markdown
---
skill_id: code-review
name: Code Review
description: Reviews pull requests with actionable feedback
origin: code-review@user_id@github.com
version: 1.0
tags: [review, quality]
---

# Code Review Skill

When to use this skill...

## Usage

[skill content following Claude Code skill format]
```

**Frontmatter Fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `skill_id` | ✅ | Unique identifier (alphanumeric + hyphens) |
| `name` | ✅ | Display name |
| `description` | ✅ | One-line description |
| `origin` | | Origin metadata: `{skill_id}@{user_id}@github.com`. Omitted for project-created skills. |
| `version` | | Optional: semantic version for skill |
| `tags` | | Optional: array of tags for discovery |

### Origin Metadata

Format: `{skill_id}@{user_id}@github.com`

**Purpose:**
- Track where the skill came from (user's Firestore collection)
- Identify which user skills can be updated in Firestore when skill is modified
- Future: support for user-backed git repositories (`{skill_id}@{user_id}@github.com.git`)

**Presence Rules:**
- Present: skill was added from user collection or imported with source
- Absent: skill was created directly in project

### Hub UI: Project Skills Section

**Location:** Hub → Project Details → "Skills" tab (new)

**Features:**

| Action | Flow |
|--------|------|
| **View Skills** | List all skills in `synchestra/skills/`. Show origin metadata if present. |
| **Add from Collection** | "Add Skill" button → modal to browse/search user's Firestore skills → copies selected skill to project repo → creates commit/PR |
| **Create New** | "Create Skill" button → inline editor → saves only to `synchestra/skills/`. No origin metadata. |
| **Save to Favorites** | "Save to My Collection" button on any project skill → copies to user's Firestore collection → available for other projects |
| **Remove from Project** | Delete skill from `synchestra/skills/`. Confirms before removing. |
| **Edit** | Inline editor for `synchestra/skills/{skill_id}/SKILL.md` content. Changes commit to project repo. |

### Skill Lifecycle in Project Context

1. **Add from Collection:** User selects saved skill → copied to `synchestra/skills/{skill_id}/SKILL.md` with origin metadata
2. **Create in Project:** User writes skill directly → saved to `synchestra/skills/{skill_id}/SKILL.md`, no origin
3. **Save to Favorites:** User takes any project skill, saves to Firestore → now in user collection for other projects
4. **Update in Project:** User modifies skill in project → reflected in `synchestra/skills/` (git-versioned)
5. **Git Versioning:** Every change to skills is a commit. History tracked naturally.

## 3. Runtime: Agent Skill Discovery

### Runner Behavior

When Synchestra runner executes an agent for a project:

```
1. Clone project repository
2. Locate all files: synchestra/skills/{skill_id}/SKILL.md
3. Copy each to: .claude/skills/{skill_id}/SKILL.md
   (preserve directory structure)
4. Agent starts with .claude/skills/ available
5. Agent discovers skills via Claude Code's native skill discovery
6. Agent invokes skills as needed during execution
```

### Agent Perspective

- **Availability:** All skills in `.claude/skills/` are discoverable
- **Format:** Frontmatter + content in SKILL.md matches Claude Code skill format
- **Invocation:** Agent uses existing Claude Code Skill tool to invoke
- **No Network:** Skills are local; agent doesn't need to access Firestore or GitHub

### Directory Structure at Runtime

```
project-repo (cloned)
  synchestra/
    skills/
      code-review/SKILL.md
      go-testing/SKILL.md

.claude/
  skills/
    code-review/SKILL.md         ← copied from project
    go-testing/SKILL.md          ← copied from project
```

## 4. Data Flow: Skill Lifecycle

### Creating Skills

**Path 1: User Collection → Project**
```
User writes in Hub UI
  → saves to Firestore: users/{user_id}/skills/{skill_id}
  
User navigates to project, adds skill from collection
  → Hub fetches skill content from Firestore
  → creates commit in project repo: synchestra/skills/{skill_id}/SKILL.md
  → includes origin metadata: skill_id@user_id@github.com
  
Project repo merged
  → skill available in synchestra/skills/
```

**Path 2: Direct Project Creation**
```
User in project, creates new skill
  → Hub editor saves to synchestra/skills/{skill_id}/SKILL.md
  → no origin metadata (it's project-specific)
  
User later saves to favorites
  → copies content to Firestore: users/{user_id}/skills/{skill_id}
  → available for other projects
```

### Updating Skills

**User Collection Update:**
```
User modifies skill in Hub
  → updates Firestore: users/{user_id}/skills/{skill_id}
  → existing project copies are NOT auto-updated (immutable)
  → user must re-add to project if they want new version
```

**Project Skill Update:**
```
User edits skill in project
  → modifies synchestra/skills/{skill_id}/SKILL.md
  → commits to project repo (git-versioned)
  → runner next time picks up new version
```

**Save Modified Project Skill to Favorites:**
```
User in project, modifies skill, then "Save to My Collection"
  → copies current synchestra/skills/{skill_id}/SKILL.md to Firestore
  → overwrites user collection version
  → other projects still have their local copies
```

### At Runtime

```
Runner clones project repo
  ↓
Discovers: synchestra/skills/code-review/SKILL.md
           synchestra/skills/go-testing/SKILL.md
  ↓
Copies to: .claude/skills/code-review/SKILL.md
           .claude/skills/go-testing/SKILL.md
  ↓
Agent starts → Claude Code discovers .claude/skills/
  ↓
Agent can use skills: synchestra-claim-task, code-review, go-testing
```

## 5. Integration Points

### Hub API (New Endpoints)

**User Skills:**
```
POST   /api/v1/users/{user_id}/skills
GET    /api/v1/users/{user_id}/skills
GET    /api/v1/users/{user_id}/skills/{skill_id}
PUT    /api/v1/users/{user_id}/skills/{skill_id}
DELETE /api/v1/users/{user_id}/skills/{skill_id}
```

**Project Skills:**
```
GET    /api/v1/projects/{project_id}/skills
POST   /api/v1/projects/{project_id}/skills/add
       (body: { skill_id, source: "user_collection" | "import" })
       (triggered: creates commit/PR in project repo)

POST   /api/v1/projects/{project_id}/skills/{skill_id}/save-favorite
       (triggered: saves project skill to user collection)

DELETE /api/v1/projects/{project_id}/skills/{skill_id}
       (triggered: deletes synchestra/skills/{skill_id}/ from project repo)
```

**Import from URL:**
```
POST /api/v1/users/{user_id}/skills/import
     (body: { source_url: "..." })
     (fetches SKILL.md, stores to Firestore)
```

### CLI (Not MVP, Future)

```
synchestra skill create --name "..." --description "..."
synchestra skill list
synchestra skill show <skill_id>
synchestra skill delete <skill_id>

synchestra project add-skill --project <id> --skill <name>
synchestra project remove-skill --project <id> --skill <name>
```

### Runner Integration

**No changes needed.** Runner already copies files from project repo to `.claude/skills/`. Works with existing mechanism.

## 6. Forward Compatibility: User Repos (Paid Feature)

### Design Principle

Origin metadata and SKILL.md format are git-serializable. Firestore → git migration requires no breaking changes.

### Future Path (Not Implemented Now)

1. **User creates personal repo:** `github.com/{user}/{user}-synchestra`
2. **Skills stored in git:** `skills/{skill_id}/SKILL.md`
3. **Hub behavior:** Syncs Firestore ↔ git or reads from git as source of truth
4. **Project skills:** Still copied to `synchestra/skills/` (no runner changes)
5. **Origin metadata:** Expanded to track git source: `code-review@user_id@github.com.git`
6. **Monetization:** "Upgrade to user repo for version control, collaboration, and history"

### Why This Matters

- **Starting with Firestore:** Fast to implement, good UX, sufficient for MVP
- **Designing for git:** Avoids rework if user demand grows. Skill definitions are git-ready now.
- **No breaking changes:** SKILL.md format and frontmatter are backward-compatible

## 7. Error Handling

### User Collection

| Scenario | Behavior |
|----------|----------|
| Skill already exists | Return error, suggest overwrite or rename |
| Invalid SKILL.md | Reject with validation errors (missing frontmatter, invalid fields) |
| Skill not found | Return 404 |
| Unauthorized | Return 403 (not the user's skill) |

### Project Skills

| Scenario | Behavior |
|----------|----------|
| Skill not found in user collection | Return 404 |
| Failed to create commit/PR | Return error with git details, suggest manual retry |
| Skill ID already in project | Return error, suggest update or rename |
| No write access to project repo | Return 403 |

## 8. Testing Strategy

### Unit Tests

- Firestore document validation (frontmatter parsing)
- Hub API request/response schemas
- Origin metadata formatting and parsing

### Integration Tests

- User skill CRUD in Firestore
- Add skill from collection to project (commit creation)
- Save project skill to favorites (Firestore write)
- Delete skill from project (file deletion commit)

### E2E Tests

- Full workflow: create user skill → add to project → runner copies to `.claude/skills/`
- Save project skill to favorites → verify in user collection
- Import skill from URL → verify in user collection
- Edit project skill → verify git commit

## 9. Outstanding Questions

None. Design is complete.

## 10. Scope & Non-Scope

### In Scope

- User skill CRUD in Firestore
- Project skill management (add, remove, create, edit)
- Hub UI for both user and project skills
- Runner copies skills to `.claude/skills/`
- Origin metadata tracking
- Save project skill to favorites

### Out of Scope (Future)

- User-backed git repositories
- Skill versioning/history
- Skill publishing/marketplace
- Collaboration (shared skill collections)
- CLI commands (Hub UI driven for MVP)
- Automatic skill updates across projects
