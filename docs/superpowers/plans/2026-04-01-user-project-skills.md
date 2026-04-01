# User & Project Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use [superpowers:subagent-driven-development](../../../.claude/plugins/cache/claude-plugins-official/superpowers/5.0.7/skills/subagent-driven-development) (recommended) or [superpowers:executing-plans](../../../.claude/plugins/cache/claude-plugins-official/superpowers/5.0.7/skills/executing-plans) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a two-tier skill system: personal user skill collections (Firestore) and project-specific skills (git-backed in `synchestra/skills/`), with seamless agent discovery at runtime.

**Architecture:**
- Shared skill types and SKILL.md parsing live in `synchestra` repo (`pkg/skills/`)
- Hub API and Firestore integration live in `synchestra-cloud` repo
- Hub frontend (Angular/PrimeNG) lives in `synchestra-hub` repo
- Runner skill copying lives in `synchestra-servers` repo

**Tech Stack:** Go (`gopkg.in/yaml.v3`, `os/exec` for git), Angular + PrimeNG (Hub UI), Firestore (user skills)

**Cross-repo map:**

| Repo | What changes |
|---|---|
| `synchestra` (this repo) | Shared `pkg/skills/` library: Skill type, SKILL.md parsing, project discovery |
| `synchestra-cloud` | Firestore user skills CRUD, Hub API endpoints (user + project skills) |
| `synchestra-hub` | Angular/PrimeNG components: user skills dashboard, project skills tab |
| `synchestra-servers` | Runner: copy `synchestra/skills/` to `.claude/skills/` at clone time |

---

## Phase 1: Shared Skill Library (synchestra repo)

### Task 1: Skill Data Model and SKILL.md Parsing

**Files:**
- Create: `pkg/skills/skill.go`
- Create: `pkg/skills/skill_test.go`

**Context:**
Foundation type used by all repos. Follows codebase conventions: `gopkg.in/yaml.v3` for YAML, `// Features implemented:` header, standard library testing with `t.TempDir()` / `t.Fatal` / `t.Errorf`.

- [ ] **Step 1: Write the failing tests**

```go
// pkg/skills/skill_test.go
package skills

// Features implemented: agent-skills

import (
	"strings"
	"testing"
)

func TestParseSkill_Valid(t *testing.T) {
	raw := "---\nskill_id: code-review\nname: Code Review\ndescription: Reviews pull requests\norigin: code-review@user123@github.com\nversion: \"1.0\"\ntags:\n  - review\n  - quality\n---\n\n# Code Review\n\nSome content here."

	skill, err := ParseSkill("code-review", []byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.SkillID != "code-review" {
		t.Errorf("SkillID = %q, want %q", skill.SkillID, "code-review")
	}
	if skill.Name != "Code Review" {
		t.Errorf("Name = %q, want %q", skill.Name, "Code Review")
	}
	if skill.Description != "Reviews pull requests" {
		t.Errorf("Description = %q", skill.Description)
	}
	if skill.Origin != "code-review@user123@github.com" {
		t.Errorf("Origin = %q", skill.Origin)
	}
	if skill.Version != "1.0" {
		t.Errorf("Version = %q, want %q", skill.Version, "1.0")
	}
	if len(skill.Tags) != 2 || skill.Tags[0] != "review" || skill.Tags[1] != "quality" {
		t.Errorf("Tags = %v, want [review quality]", skill.Tags)
	}
	if !strings.Contains(string(skill.Body), "# Code Review") {
		t.Errorf("Body missing content, got %q", string(skill.Body))
	}
}

func TestParseSkill_MissingSkillID(t *testing.T) {
	raw := "---\nname: Code Review\ndescription: Reviews PRs\n---\n\ncontent"

	_, err := ParseSkill("code-review", []byte(raw))
	if err == nil {
		t.Fatal("expected error for missing skill_id")
	}
}

func TestParseSkill_SkillIDMismatch(t *testing.T) {
	raw := "---\nskill_id: wrong-id\nname: Code Review\ndescription: Reviews PRs\n---\n\ncontent"

	_, err := ParseSkill("code-review", []byte(raw))
	if err == nil {
		t.Fatal("expected error for skill_id mismatch")
	}
}

func TestParseSkill_MissingFrontmatter(t *testing.T) {
	raw := "# Just markdown\n\nNo frontmatter."

	_, err := ParseSkill("test", []byte(raw))
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParseSkill_NoOrigin(t *testing.T) {
	raw := "---\nskill_id: custom\nname: Custom Skill\ndescription: Project-created skill\n---\n\n# Custom"

	skill, err := ParseSkill("custom", []byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Origin != "" {
		t.Errorf("Origin = %q, want empty", skill.Origin)
	}
}

func TestSkill_Serialize(t *testing.T) {
	skill := Skill{
		SkillID:     "code-review",
		Name:        "Code Review",
		Description: "Reviews pull requests",
		Origin:      "code-review@user123@github.com",
		Version:     "1.0",
		Tags:        []string{"review", "quality"},
		Body:        []byte("# Code Review\n\nContent here."),
	}

	data := skill.Serialize()
	s := string(data)

	if !strings.Contains(s, "skill_id: code-review") {
		t.Error("missing skill_id in output")
	}
	if !strings.Contains(s, "origin: code-review@user123@github.com") {
		t.Error("missing origin in output")
	}
	if !strings.Contains(s, "# Code Review\n\nContent here.") {
		t.Error("missing body in output")
	}

	// Round-trip: serialize then parse should produce the same skill.
	parsed, err := ParseSkill("code-review", data)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}
	if parsed.SkillID != skill.SkillID {
		t.Errorf("round-trip SkillID = %q, want %q", parsed.SkillID, skill.SkillID)
	}
	if parsed.Origin != skill.Origin {
		t.Errorf("round-trip Origin = %q, want %q", parsed.Origin, skill.Origin)
	}
}

func TestSkill_Serialize_NoOptionalFields(t *testing.T) {
	skill := Skill{
		SkillID:     "simple",
		Name:        "Simple Skill",
		Description: "A simple skill",
		Body:        []byte("# Simple"),
	}

	data := skill.Serialize()
	s := string(data)

	if strings.Contains(s, "origin:") {
		t.Error("should not contain origin when empty")
	}
	if strings.Contains(s, "version:") {
		t.Error("should not contain version when empty")
	}
	if strings.Contains(s, "tags:") {
		t.Error("should not contain tags when empty")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/skills/ -v
```

Expected: FAIL — package does not exist.

- [ ] **Step 3: Write the Skill type and parsing logic**

```go
// pkg/skills/skill.go
package skills

// Features implemented: agent-skills

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Skill represents a skill definition parsed from a SKILL.md file.
// The file format is YAML frontmatter delimited by "---" followed by
// a markdown body.
type Skill struct {
	SkillID     string   `yaml:"skill_id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Origin      string   `yaml:"origin,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Body        []byte   `yaml:"-"` // markdown content after frontmatter
}

var frontmatterDelim = []byte("---")

// ParseSkill parses a SKILL.md file. expectedID is the skill_id from the
// directory name; it must match the frontmatter skill_id.
func ParseSkill(expectedID string, data []byte) (*Skill, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var s Skill
	if err := yaml.Unmarshal(fm, &s); err != nil {
		return nil, fmt.Errorf("parsing skill frontmatter: %w", err)
	}

	if s.SkillID == "" {
		return nil, fmt.Errorf("skill_id is required in frontmatter")
	}
	if s.SkillID != expectedID {
		return nil, fmt.Errorf("skill_id mismatch: frontmatter has %q, directory is %q", s.SkillID, expectedID)
	}
	if s.Name == "" {
		return nil, fmt.Errorf("name is required in frontmatter")
	}
	if s.Description == "" {
		return nil, fmt.Errorf("description is required in frontmatter")
	}

	s.Body = body
	return &s, nil
}

// Serialize renders the skill back to SKILL.md format (frontmatter + body).
func (s *Skill) Serialize() []byte {
	fm, _ := yaml.Marshal(s) // Skill yaml tags control output

	var buf bytes.Buffer
	buf.Write(frontmatterDelim)
	buf.WriteByte('\n')
	buf.Write(fm)
	buf.Write(frontmatterDelim)
	buf.WriteByte('\n')
	if len(s.Body) > 0 {
		buf.WriteByte('\n')
		buf.Write(s.Body)
		// Ensure trailing newline.
		if s.Body[len(s.Body)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes()
}

// splitFrontmatter separates YAML frontmatter from the body.
// Expects the file to start with "---\n".
func splitFrontmatter(data []byte) (frontmatter, body []byte, err error) {
	data = bytes.TrimLeft(data, "\n")
	if !bytes.HasPrefix(data, frontmatterDelim) {
		return nil, nil, fmt.Errorf("missing frontmatter: file must start with ---")
	}

	// Find closing delimiter.
	rest := data[len(frontmatterDelim)+1:] // skip "---\n"
	idx := bytes.Index(rest, append(frontmatterDelim, '\n'))
	if idx < 0 {
		return nil, nil, fmt.Errorf("unclosed frontmatter: missing closing ---")
	}

	fm := rest[:idx]
	body = bytes.TrimLeft(rest[idx+len(frontmatterDelim)+1:], "\n")
	return fm, body, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/skills/ -v
```

Expected: PASS — all tests green.

- [ ] **Step 5: Commit**

```bash
git add pkg/skills/skill.go pkg/skills/skill_test.go
git commit -m "feat(skills): add Skill type and SKILL.md parsing with yaml.v3"
```

---

### Task 2: Project Skills Discovery and Mutation

**Files:**
- Create: `pkg/skills/project.go`
- Create: `pkg/skills/project_test.go`

**Context:**
Read all skills from a project's `synchestra/skills/` directory, and write skills into it. Used by the runner (read) and by cloud API (write + git commit). Follows existing `pkg/cli/gitops/` pattern of using `os/exec` for git.

- [ ] **Step 1: Write the failing tests**

```go
// pkg/skills/project_test.go
package skills

// Features implemented: agent-skills

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestSkill(t *testing.T, dir, skillID, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, "synchestra", "skills", skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, SkillFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverProjectSkills(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, root, "code-review",
		"---\nskill_id: code-review\nname: Code Review\ndescription: Reviews PRs\n---\n\n# Code Review")
	writeTestSkill(t, root, "go-test",
		"---\nskill_id: go-test\nname: Go Testing\ndescription: Tests Go code\n---\n\n# Go Testing")

	skills, err := DiscoverProjectSkills(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("got %d skills, want 2", len(skills))
	}

	ids := map[string]bool{}
	for _, s := range skills {
		ids[s.SkillID] = true
	}
	if !ids["code-review"] || !ids["go-test"] {
		t.Errorf("missing expected skill IDs, got %v", ids)
	}
}

func TestDiscoverProjectSkills_NoDir(t *testing.T) {
	root := t.TempDir() // no synchestra/skills/ created

	skills, err := DiscoverProjectSkills(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("got %d skills, want 0 for missing directory", len(skills))
	}
}

func TestWriteProjectSkill(t *testing.T) {
	root := t.TempDir()
	skill := &Skill{
		SkillID:     "new-skill",
		Name:        "New Skill",
		Description: "A new skill",
		Origin:      "new-skill@user123@github.com",
		Body:        []byte("# New Skill\n\nContent."),
	}

	if err := WriteProjectSkill(root, skill); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists and round-trips.
	path := filepath.Join(root, "synchestra", "skills", "new-skill", SkillFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("skill file not found: %v", err)
	}

	parsed, err := ParseSkill("new-skill", data)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}
	if parsed.Origin != "new-skill@user123@github.com" {
		t.Errorf("Origin = %q, want %q", parsed.Origin, "new-skill@user123@github.com")
	}
}

func TestRemoveProjectSkill(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, root, "doomed",
		"---\nskill_id: doomed\nname: Doomed\ndescription: Will be removed\n---\n\n# Doomed")

	if err := RemoveProjectSkill(root, "doomed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := filepath.Join(root, "synchestra", "skills", "doomed")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("skill directory should be removed, got err=%v", err)
	}
}

func TestCopySkillsToTarget(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()

	writeTestSkill(t, root, "s1",
		"---\nskill_id: s1\nname: S1\ndescription: Skill 1\n---\n\n# S1")
	writeTestSkill(t, root, "s2",
		"---\nskill_id: s2\nname: S2\ndescription: Skill 2\n---\n\n# S2")

	n, err := CopySkillsToTarget(root, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("copied %d skills, want 2", n)
	}

	// Verify target has the files.
	for _, id := range []string{"s1", "s2"} {
		path := filepath.Join(target, id, SkillFile)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s at target, got err: %v", id, err)
		}
	}
}

func TestCopySkillsToTarget_NoSkills(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()

	n, err := CopySkillsToTarget(root, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("copied %d skills, want 0", n)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/skills/ -v -run "TestDiscover|TestWrite|TestRemove|TestCopy"
```

Expected: FAIL — functions not defined.

- [ ] **Step 3: Write the project skills operations**

```go
// pkg/skills/project.go
package skills

// Features implemented: agent-skills

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// SkillFile is the filename for skill definitions.
const SkillFile = "SKILL.md"

// ProjectSkillsDir returns the path to the skills directory inside a project.
func ProjectSkillsDir(projectRoot string) string {
	return filepath.Join(projectRoot, "synchestra", "skills")
}

// DiscoverProjectSkills reads all skills from projectRoot/synchestra/skills/.
// Returns an empty slice (not error) if the directory does not exist.
func DiscoverProjectSkills(projectRoot string) ([]*Skill, error) {
	dir := ProjectSkillsDir(projectRoot)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading skills directory: %w", err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillID := entry.Name()
		data, err := os.ReadFile(filepath.Join(dir, skillID, SkillFile))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue // directory without SKILL.md, skip
			}
			return nil, fmt.Errorf("reading skill %s: %w", skillID, err)
		}
		skill, err := ParseSkill(skillID, data)
		if err != nil {
			return nil, fmt.Errorf("parsing skill %s: %w", skillID, err)
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

// WriteProjectSkill writes a skill to projectRoot/synchestra/skills/{skill_id}/SKILL.md.
func WriteProjectSkill(projectRoot string, skill *Skill) error {
	dir := filepath.Join(ProjectSkillsDir(projectRoot), skill.SkillID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}
	data := skill.Serialize()
	if err := os.WriteFile(filepath.Join(dir, SkillFile), data, 0644); err != nil {
		return fmt.Errorf("writing skill file: %w", err)
	}
	return nil
}

// RemoveProjectSkill deletes the skill directory from the project.
func RemoveProjectSkill(projectRoot string, skillID string) error {
	dir := filepath.Join(ProjectSkillsDir(projectRoot), skillID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing skill %s: %w", skillID, err)
	}
	return nil
}

// CopySkillsToTarget copies all skills from projectRoot/synchestra/skills/
// to targetDir (e.g., .claude/skills/). Returns the number of skills copied.
func CopySkillsToTarget(projectRoot string, targetDir string) (int, error) {
	srcDir := ProjectSkillsDir(projectRoot)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading skills directory: %w", err)
	}

	n := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillID := entry.Name()
		src := filepath.Join(srcDir, skillID, SkillFile)
		if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
			continue
		}

		dstDir := filepath.Join(targetDir, skillID)
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			return n, fmt.Errorf("creating target directory for %s: %w", skillID, err)
		}

		if err := copyFile(src, filepath.Join(dstDir, SkillFile)); err != nil {
			return n, fmt.Errorf("copying skill %s: %w", skillID, err)
		}
		n++
	}
	return n, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/skills/ -v
```

Expected: PASS — all tests green.

- [ ] **Step 5: Commit**

```bash
git add pkg/skills/project.go pkg/skills/project_test.go
git commit -m "feat(skills): add project skill discovery, write, remove, and copy"
```

---

## Phase 2: Cloud API (synchestra-cloud repo)

> **Note:** Tasks 3-5 target the `synchestra-cloud` repo. The implementing agent must explore that repo's conventions (router, middleware, Firestore patterns, auth) before writing code. The interfaces below define the contract; adapt to the repo's actual patterns.

### Task 3: Firestore User Skills Repository

**Files:**
- Create: `pkg/skills/repository.go` (in synchestra-cloud)
- Create: `pkg/skills/repository_test.go`

**Context:**
CRUD for `users/{user_id}/skills/{skill_id}` subcollection. Check how `synchestra-cloud` initializes its Firestore client and follow that pattern.

- [ ] **Step 1: Write failing tests for the repository interface and Firestore implementation**

The repository interface:

```go
// UserSkillDoc is the Firestore document shape for users/{uid}/skills/{sid}.
type UserSkillDoc struct {
	SkillID     string    `firestore:"skill_id"`
	Name        string    `firestore:"name"`
	Description string    `firestore:"description"`
	Content     string    `firestore:"content"` // raw SKILL.md content (frontmatter + body)
	Tags        []string  `firestore:"tags,omitempty"`
	Version     string    `firestore:"version,omitempty"`
	CreatedAt   time.Time `firestore:"created_at"`
	UpdatedAt   time.Time `firestore:"updated_at"`
}

// UserSkillRepository defines CRUD for user skill collections.
type UserSkillRepository interface {
	Create(ctx context.Context, userID string, doc *UserSkillDoc) error
	Get(ctx context.Context, userID, skillID string) (*UserSkillDoc, error)
	List(ctx context.Context, userID string) ([]*UserSkillDoc, error)
	Update(ctx context.Context, userID, skillID string, updates map[string]any) error
	Delete(ctx context.Context, userID, skillID string) error
}
```

Tests should use the Firestore emulator (`FIRESTORE_EMULATOR_HOST`). Write one test per method: create, get after create, list, update, delete, get-not-found.

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement the Firestore-backed repository**

Use `client.Collection("users").Doc(userID).Collection("skills").Doc(skillID)` path. Set `created_at` on create, `updated_at` on create and update.

- [ ] **Step 4: Run tests with emulator**

```bash
FIRESTORE_EMULATOR_HOST=localhost:8080 go test ./pkg/skills/ -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/skills/
git commit -m "feat(skills): add Firestore user skills repository"
```

---

### Task 4: User Skills API Endpoints

**Files:**
- Create or modify: API routes file (in synchestra-cloud)
- Create: handler file and tests

**Context:**
REST endpoints called by Hub frontend. Follow the existing auth middleware pattern in synchestra-cloud to extract user ID from the request (JWT token or session).

**Endpoints to implement:**

| Method | Path | Handler |
|---|---|---|
| `POST` | `/api/v1/users/{user_id}/skills` | Create user skill |
| `GET` | `/api/v1/users/{user_id}/skills` | List user skills |
| `GET` | `/api/v1/users/{user_id}/skills/{skill_id}` | Get user skill |
| `PUT` | `/api/v1/users/{user_id}/skills/{skill_id}` | Update user skill |
| `DELETE` | `/api/v1/users/{user_id}/skills/{skill_id}` | Delete user skill |

- [ ] **Step 1: Write failing handler tests**

One test per endpoint. Use `httptest.NewRequest` + `httptest.NewRecorder`. Mock the repository interface — do NOT use a concrete Firestore dependency in handler tests.

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement handlers**

Each handler: parse request, validate, call repository, return JSON response. Auth: verify the requesting user matches `{user_id}` in the path (or is admin). Return 400 for bad input, 404 for not found, 409 for duplicate skill_id.

Create request body example:
```json
{
  "name": "Code Review",
  "description": "Reviews pull requests",
  "content": "---\nskill_id: code-review\n...",
  "tags": ["review"],
  "version": "1.0"
}
```

If `skill_id` is not provided, derive it from name via slugification.

- [ ] **Step 4: Register routes in the router**

- [ ] **Step 5: Run tests**

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add <handler files>
git commit -m "feat(skills): add user skills API endpoints"
```

---

### Task 5: Project Skills API Endpoints

**Files:**
- Create or modify: API routes file (in synchestra-cloud)
- Create: handler file and tests

**Context:**
These endpoints modify the project repo on disk (or via GitHub API). When a user adds a skill, the cloud service must: clone/fetch the project repo, write the SKILL.md, commit, and push. Follow whatever git integration pattern synchestra-cloud already uses.

**Endpoints to implement:**

| Method | Path | Handler |
|---|---|---|
| `GET` | `/api/v1/projects/{project_id}/skills` | List skills in project repo |
| `POST` | `/api/v1/projects/{project_id}/skills/add` | Add skill from user collection to project |
| `POST` | `/api/v1/projects/{project_id}/skills` | Create new skill directly in project |
| `POST` | `/api/v1/projects/{project_id}/skills/{skill_id}/save-favorite` | Save project skill to user collection |
| `DELETE` | `/api/v1/projects/{project_id}/skills/{skill_id}` | Remove skill from project |

- [ ] **Step 1: Write failing handler tests**

For the `add` endpoint, mock both the user skill repository (to fetch the source skill) and the git operations (to avoid real clones in tests).

- [ ] **Step 2: Run tests to verify they fail**

- [ ] **Step 3: Implement handlers**

**Add from collection** flow:
1. Fetch skill from Firestore: `users/{user_id}/skills/{skill_id}`
2. Parse the content, inject origin metadata: `{skill_id}@{user_id}@github.com`
3. Use `skills.WriteProjectSkill()` (imported from `synchestra` repo's `pkg/skills/`)
4. Git add, commit, push using existing gitops pattern
5. Return 201

**Create new** flow:
1. Parse request body (name, description, content)
2. Write to `synchestra/skills/{skill_id}/SKILL.md` — no origin metadata
3. Git add, commit, push
4. Return 201

**Save to favorites** flow:
1. Read `synchestra/skills/{skill_id}/SKILL.md` from project repo
2. Create Firestore doc in `users/{user_id}/skills/{skill_id}`
3. Strip or ignore origin metadata in the Firestore copy
4. Return 201

- [ ] **Step 4: Run tests**

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add <handler files>
git commit -m "feat(skills): add project skills API endpoints (add, create, save-favorite, remove)"
```

---

## Phase 3: Hub Frontend (synchestra-hub repo)

> **Note:** Tasks 6-8 target the `synchestra-hub` repo (Angular + PrimeNG). The implementing agent must explore the existing component patterns, routing, and PrimeNG usage before writing code.

### Task 6: Skill Model and Service

**Files:**
- Create: `src/app/models/skill.model.ts`
- Create: `src/app/services/skill.service.ts`

- [ ] **Step 1: Create the TypeScript model**

```typescript
export interface Skill {
  skill_id: string;
  name: string;
  description: string;
  content: string;
  tags?: string[];
  version?: string;
  origin?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateSkillRequest {
  skill_id?: string;
  name: string;
  description: string;
  content: string;
  tags?: string[];
  version?: string;
}
```

- [ ] **Step 2: Create the HTTP service**

Service methods:
- `createUserSkill(userId, request)` → POST `/api/v1/users/{userId}/skills`
- `getUserSkills(userId)` → GET `/api/v1/users/{userId}/skills`
- `getUserSkill(userId, skillId)` → GET `/api/v1/users/{userId}/skills/{skillId}`
- `updateUserSkill(userId, skillId, updates)` → PUT
- `deleteUserSkill(userId, skillId)` → DELETE
- `addSkillToProject(projectId, body)` → POST `/api/v1/projects/{projectId}/skills/add`
- `createProjectSkill(projectId, body)` → POST `/api/v1/projects/{projectId}/skills`
- `removeSkillFromProject(projectId, skillId)` → DELETE
- `saveProjectSkillToFavorites(projectId, skillId)` → POST `.../save-favorite`

Follow the existing `HttpClient` injection pattern in the hub.

- [ ] **Step 3: Commit**

```bash
git add src/app/models/skill.model.ts src/app/services/skill.service.ts
git commit -m "feat(skills): add skill model and API service"
```

---

### Task 7: User Skills Dashboard Page

**Files:**
- Create: component files under `src/app/features/skills/` (or wherever the hub puts feature pages)
- Modify: routing module to add the new route

**Context:**
Use PrimeNG components: `p-table` or `p-dataView` for the list, `p-dialog` for modals, `p-button`, `p-inputText`, `p-chips` for tags, `p-tag` for tag badges.

- [ ] **Step 1: Create the dashboard component**

Features:
- List all user skills in a `p-dataView` or `p-table` with columns: name, description, tags, updated date
- Search input (`p-inputText`) that filters client-side
- "Create Skill" button opens a `p-dialog` with the skill editor form
- Each row has actions: Edit (opens dialog), Delete (with `p-confirmDialog`)
- Empty state message when no skills exist

- [ ] **Step 2: Add route**

Register in the appropriate routing module (e.g., `/skills` under the user section).

- [ ] **Step 3: Verify the page renders**

```bash
ng serve
# Navigate to /skills
```

Expected: Page renders with empty state or skill list.

- [ ] **Step 4: Commit**

```bash
git add src/app/features/skills/
git commit -m "feat(skills): add user skills dashboard page with PrimeNG"
```

---

### Task 8: Project Skills Tab

**Files:**
- Create: component files for the project skills section
- Modify: project detail page to add the "Skills" tab

**Context:**
This is a new tab on the existing project detail page. Use PrimeNG `p-tabView` if the project page already uses tabs, or add one.

- [ ] **Step 1: Create the project skills component**

Features:
- List skills currently in the project (calls `GET /api/v1/projects/{id}/skills`)
- "Add from Collection" button opens `p-dialog` listing user's saved skills
- "Create New" button opens `p-dialog` with skill editor (no origin metadata)
- Each skill row has:
  - Star icon to "Save to My Collection" (calls save-favorite endpoint)
  - Delete icon to remove from project (with confirmation)
  - Origin badge if `origin` is present

- [ ] **Step 2: Wire into project detail page**

Add the component as a new tab or section on the existing project page.

- [ ] **Step 3: Verify the tab renders**

```bash
ng serve
# Navigate to a project, check the Skills tab
```

Expected: Tab renders, skills list loads.

- [ ] **Step 4: Commit**

```bash
git add src/app/features/projects/
git commit -m "feat(skills): add project skills tab with add/create/remove/favorite actions"
```

---

## Phase 4: Runner Integration (synchestra-servers repo)

### Task 9: Copy Project Skills to .claude/skills/ at Clone Time

**Files:**
- Modify: runner setup/clone logic (in synchestra-servers)

**Context:**
The runner already clones project repos. After cloning, it needs one additional step: copy skills. Import `github.com/synchestra-io/synchestra/pkg/skills` and use `skills.CopySkillsToTarget()`.

- [ ] **Step 1: Write a failing test**

Test that after runner setup, `.claude/skills/` contains the skills from `synchestra/skills/`. Use a temp directory with a fake project repo.

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Add the copy call**

In the runner's post-clone setup, add:

```go
import "github.com/synchestra-io/synchestra/pkg/skills"

// After cloning the project repo...
claudeSkillsDir := filepath.Join(projectRoot, ".claude", "skills")
n, err := skills.CopySkillsToTarget(projectRoot, claudeSkillsDir)
if err != nil {
    return fmt.Errorf("copying project skills: %w", err)
}
if n > 0 {
    log.Printf("copied %d project skills to .claude/skills/", n)
}
```

- [ ] **Step 4: Run test**

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add <runner files>
git commit -m "feat(runner): copy project skills to .claude/skills/ at clone time"
```

---

## Phase 5: Integration Testing & Documentation

### Task 10: End-to-End Smoke Test

**Files:**
- Create: test file in synchestra repo (or synchestra-cloud depending on where E2E tests live)

- [ ] **Step 1: Write E2E test covering the full lifecycle**

Test steps:
1. Create a user skill via API → verify 201
2. List user skills → verify the skill appears
3. Add skill to project → verify `synchestra/skills/{id}/SKILL.md` exists with origin metadata
4. Create a new skill directly in project → verify it exists without origin
5. Save project skill to favorites → verify it appears in user skill list
6. Remove skill from project → verify deleted
7. Simulate runner: call `CopySkillsToTarget()` → verify `.claude/skills/` populated

- [ ] **Step 2: Run the test**

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add <test files>
git commit -m "test(skills): add end-to-end smoke test for user/project skills lifecycle"
```

---

### Task 11: Feature Documentation

**Files:**
- Create: `docs/features/user-project-skills.md` (in synchestra repo)

- [ ] **Step 1: Write feature docs**

Cover:
- Overview of the two-tier system (user skills + project skills)
- How to manage user skills (Hub → My Skills)
- How to manage project skills (Hub → Project → Skills tab)
- SKILL.md format reference (frontmatter fields, origin metadata)
- How the runner makes skills available to agents
- Directory structure at runtime

- [ ] **Step 2: Commit**

```bash
git add docs/features/user-project-skills.md
git commit -m "docs: add user and project skills feature documentation"
```
