# User & Project Skills Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use [superpowers:subagent-driven-development](../../../.claude/plugins/cache/claude-plugins-official/superpowers/5.0.7/skills/subagent-driven-development) (recommended) or [superpowers:executing-plans](../../../.claude/plugins/cache/claude-plugins-official/superpowers/5.0.7/skills/executing-plans) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a two-tier skill system allowing users to save personal skill collections in Firestore and manage project-specific skills in git, with seamless agent discovery at runtime.

**Architecture:** 
- **User Skills:** Firestore-backed personal skill library, managed via Hub UI
- **Project Skills:** Git-backed collection in `synchestra/skills/`, copied to `.claude/skills/` at runtime
- **Integration:** Hub APIs and UI for CRUD, origin metadata tracking for migration path, runner logic to copy skills at clone time

**Tech Stack:** 
- Backend: Go (pkg/skills, API endpoints)
- Database: Firestore (user skills collection)
- Frontend: Angular + PrimeNG (Hub UI)
- Runtime: Runner copies skills to `.claude/skills/`

---

## Phase 1: Core Infrastructure & Data Models

### Task 1: Skill Data Model and Validation

**Files:**
- Create: `pkg/skills/model.go`
- Create: `pkg/skills/model_test.go`

**Context:**
This is the foundation. All other tasks depend on the Skill data structure and frontmatter parsing logic.

- [ ] **Step 1: Write test for skill frontmatter parsing**

```go
package skills

import (
	"testing"
)

func TestParseSkillFrontmatter(t *testing.T) {
	content := `---
skill_id: code-review
name: Code Review
description: Reviews pull requests
origin: code-review@user123@github.com
version: 1.0
tags: [review, quality]
---

# Code Review

Some content here.`

	skill, err := ParseSkill("code-review", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skill.SkillID != "code-review" {
		t.Errorf("expected SkillID 'code-review', got '%s'", skill.SkillID)
	}
	if skill.Name != "Code Review" {
		t.Errorf("expected Name 'Code Review', got '%s'", skill.Name)
	}
	if skill.Description != "Reviews pull requests" {
		t.Errorf("expected Description, got '%s'", skill.Description)
	}
	if skill.Origin != "code-review@user123@github.com" {
		t.Errorf("expected Origin, got '%s'", skill.Origin)
	}
	if skill.Version != "1.0" {
		t.Errorf("expected Version '1.0', got '%s'", skill.Version)
	}
	if len(skill.Tags) != 2 || skill.Tags[0] != "review" {
		t.Errorf("expected Tags [review, quality], got %v", skill.Tags)
	}
	if !string(skill.Content) == `# Code Review

Some content here.` {
		t.Errorf("expected content without frontmatter")
	}
}

func TestParseSkillMissingRequired(t *testing.T) {
	// Missing skill_id
	content := `---
name: Code Review
description: Reviews PRs
---`

	_, err := ParseSkill("some-id", content)
	if err == nil {
		t.Fatal("expected error for missing skill_id")
	}
	if err.Error() != "skill_id mismatch: expected 'some-id', got missing frontmatter" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSerializeSkillFrontmatter(t *testing.T) {
	skill := &Skill{
		SkillID:     "code-review",
		Name:        "Code Review",
		Description: "Reviews pull requests",
		Origin:      "code-review@user123@github.com",
		Version:     "1.0",
		Tags:        []string{"review", "quality"},
		Content:     []byte("# Code Review\n\nContent here."),
	}

	output := skill.Serialize()
	if !contains(output, "skill_id: code-review") {
		t.Errorf("frontmatter missing skill_id")
	}
	if !contains(output, "origin: code-review@user123@github.com") {
		t.Errorf("frontmatter missing origin")
	}
	if !contains(output, "# Code Review\n\nContent here.") {
		t.Errorf("content not preserved in output")
	}
}

func contains(s string, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/alexandertrakhimenok/projects/synchestra-io/synchestra
go test ./pkg/skills -v
```

Expected: FAIL (package doesn't exist yet)

- [ ] **Step 3: Write the Skill data model**

```go
package skills

import (
	"fmt"
	"regexp"
	"strings"
)

// Skill represents a skill definition with frontmatter and content.
type Skill struct {
	SkillID     string   `yaml:"skill_id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Origin      string   `yaml:"origin,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Content     []byte   // Markdown content without frontmatter
}

// ParseSkill parses a SKILL.md file content and extracts frontmatter + body.
// skillID is the expected skill_id; it must match the frontmatter.
func ParseSkill(skillID string, content string) (*Skill, error) {
	// Extract frontmatter
	fm, body, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Parse YAML frontmatter
	parsedFM := parseFrontmatter(fm)

	// Validate required fields
	if parsedFM["skill_id"] == "" {
		return nil, fmt.Errorf("skill_id mismatch: expected '%s', got missing frontmatter", skillID)
	}
	if parsedFM["skill_id"] != skillID {
		return nil, fmt.Errorf("skill_id mismatch: expected '%s', got '%s'", skillID, parsedFM["skill_id"])
	}
	if parsedFM["name"] == "" {
		return nil, fmt.Errorf("missing required field: name")
	}
	if parsedFM["description"] == "" {
		return nil, fmt.Errorf("missing required field: description")
	}

	skill := &Skill{
		SkillID:     parsedFM["skill_id"],
		Name:        parsedFM["name"],
		Description: parsedFM["description"],
		Origin:      parsedFM["origin"],
		Version:     parsedFM["version"],
		Tags:        parseStringArray(parsedFM["tags"]),
		Content:     []byte(body),
	}

	return skill, nil
}

// Serialize converts a Skill back to SKILL.md format with frontmatter.
func (s *Skill) Serialize() string {
	var fm strings.Builder
	fm.WriteString("---\n")
	fm.WriteString(fmt.Sprintf("skill_id: %s\n", s.SkillID))
	fm.WriteString(fmt.Sprintf("name: %s\n", s.Name))
	fm.WriteString(fmt.Sprintf("description: %s\n", s.Description))

	if s.Origin != "" {
		fm.WriteString(fmt.Sprintf("origin: %s\n", s.Origin))
	}
	if s.Version != "" {
		fm.WriteString(fmt.Sprintf("version: %s\n", s.Version))
	}
	if len(s.Tags) > 0 {
		fm.WriteString(fmt.Sprintf("tags: %v\n", s.Tags))
	}
	fm.WriteString("---\n\n")
	fm.Write(s.Content)

	return fm.String()
}

// Helper: extract frontmatter and body from markdown
func extractFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return "", "", fmt.Errorf("missing frontmatter delimiter")
	}

	fmEnd := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			fmEnd = i
			break
		}
	}
	if fmEnd == -1 {
		return "", "", fmt.Errorf("unclosed frontmatter")
	}

	fm := strings.Join(lines[1:fmEnd], "\n")
	body := strings.Join(lines[fmEnd+1:], "\n")
	body = strings.TrimSpace(body)

	return fm, body, nil
}

// Helper: parse YAML-like frontmatter into map
func parseFrontmatter(fm string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			result[key] = val
		}
	}
	return result
}

// Helper: parse tags field (YAML array)
func parseStringArray(raw string) []string {
	if raw == "" {
		return nil
	}
	// Simple parser for [item1, item2] format
	raw = strings.Trim(raw, "[]")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		result = append(result, p)
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/skills -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/skills/model.go pkg/skills/model_test.go
git commit -m "feat: add skill data model and frontmatter parsing"
```

---

## Phase 2: User Skills (Firestore)

### Task 2: Firestore User Skills CRUD

**Files:**
- Create: `pkg/skills/firestore.go`
- Create: `pkg/skills/firestore_test.go`

**Context:**
User skills are stored in Firestore. This task implements create, read, update, delete operations.

- [ ] **Step 1: Write tests for Firestore operations**

```go
package skills

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

// MockFirestore for testing (use firestore emulator in real env)
type mockFS struct{}

func TestCreateUserSkill(t *testing.T) {
	ctx := context.Background()
	repo := &FirestoreSkillRepository{} // initialized with real client in integration test

	skill := &UserSkill{
		SkillID:     "code-review",
		Name:        "Code Review",
		Description: "Reviews pull requests",
		Content:     "# Code Review\n\nContent",
		Tags:        []string{"review"},
	}

	userID := "user123"
	docID, err := repo.CreateUserSkill(ctx, userID, skill)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if docID == "" {
		t.Fatal("expected non-empty docID")
	}
}

func TestGetUserSkill(t *testing.T) {
	ctx := context.Background()
	repo := &FirestoreSkillRepository{}

	userID := "user123"
	skillID := "code-review"
	skill, err := repo.GetUserSkill(ctx, userID, skillID)
	if err != nil && !isNotFound(err) {
		t.Fatalf("unexpected error: %v", err)
	}

	if skill != nil && skill.SkillID != skillID {
		t.Errorf("expected SkillID '%s', got '%s'", skillID, skill.SkillID)
	}
}

func TestListUserSkills(t *testing.T) {
	ctx := context.Background()
	repo := &FirestoreSkillRepository{}

	userID := "user123"
	skills, err := repo.ListUserSkills(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if skills == nil {
		t.Fatal("expected skills list, got nil")
	}
}

func TestUpdateUserSkill(t *testing.T) {
	ctx := context.Background()
	repo := &FirestoreSkillRepository{}

	userID := "user123"
	skillID := "code-review"
	updates := map[string]interface{}{
		"name": "Enhanced Code Review",
	}

	err := repo.UpdateUserSkill(ctx, userID, skillID, updates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUserSkill(t *testing.T) {
	ctx := context.Background()
	repo := &FirestoreSkillRepository{}

	userID := "user123"
	skillID := "code-review"

	err := repo.DeleteUserSkill(ctx, userID, skillID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func isNotFound(err error) bool {
	// Check if error is Firestore "not found"
	return firestore.IsNotFound(err) || err == firestore.ErrNotFound
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/skills -v -run TestCreateUserSkill
```

Expected: FAIL (FirestoreSkillRepository not defined)

- [ ] **Step 3: Write Firestore repository**

```go
package skills

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
)

// UserSkill represents a skill stored in Firestore user collection.
type UserSkill struct {
	SkillID     string    `firestore:"skill_id"`
	Name        string    `firestore:"name"`
	Description string    `firestore:"description"`
	Content     string    `firestore:"content"` // full SKILL.md content
	Tags        []string  `firestore:"tags"`
	Version     string    `firestore:"version,omitempty"`
	CreatedAt   time.Time `firestore:"created_at,serverTimestamp"`
	UpdatedAt   time.Time `firestore:"updated_at,serverTimestamp"`
}

// FirestoreSkillRepository handles user skill CRUD in Firestore.
type FirestoreSkillRepository struct {
	client *firestore.Client
}

// NewFirestoreSkillRepository creates a new repository.
func NewFirestoreSkillRepository(client *firestore.Client) *FirestoreSkillRepository {
	return &FirestoreSkillRepository{client: client}
}

// CreateUserSkill creates a new skill in the user's collection.
// Returns the document ID.
func (r *FirestoreSkillRepository) CreateUserSkill(ctx context.Context, userID string, skill *UserSkill) (string, error) {
	if skill.SkillID == "" {
		return "", fmt.Errorf("skill_id is required")
	}

	doc := r.client.Collection("users").Doc(userID).Collection("skills").Doc(skill.SkillID)
	_, err := doc.Set(ctx, skill)
	if err != nil {
		return "", fmt.Errorf("failed to create skill: %w", err)
	}

	return doc.ID, nil
}

// GetUserSkill retrieves a skill by ID.
func (r *FirestoreSkillRepository) GetUserSkill(ctx context.Context, userID, skillID string) (*UserSkill, error) {
	doc, err := r.client.Collection("users").Doc(userID).Collection("skills").Doc(skillID).Get(ctx)
	if err != nil {
		if firestore.IsNotFound(err) {
			return nil, nil // not found is not an error
		}
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}

	var skill UserSkill
	if err := doc.DataTo(&skill); err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}

	return &skill, nil
}

// ListUserSkills lists all skills for a user.
func (r *FirestoreSkillRepository) ListUserSkills(ctx context.Context, userID string) ([]*UserSkill, error) {
	docs, err := r.client.Collection("users").Doc(userID).Collection("skills").Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}

	var skills []*UserSkill
	for _, doc := range docs {
		var skill UserSkill
		if err := doc.DataTo(&skill); err != nil {
			return nil, fmt.Errorf("failed to parse skill: %w", err)
		}
		skills = append(skills, &skill)
	}

	return skills, nil
}

// UpdateUserSkill updates specific fields of a skill.
func (r *FirestoreSkillRepository) UpdateUserSkill(ctx context.Context, userID, skillID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.client.Collection("users").Doc(userID).Collection("skills").Doc(skillID).Update(ctx, toUpdatePaths(updates))
	if err != nil {
		return fmt.Errorf("failed to update skill: %w", err)
	}

	return nil
}

// DeleteUserSkill deletes a skill.
func (r *FirestoreSkillRepository) DeleteUserSkill(ctx context.Context, userID, skillID string) error {
	_, err := r.client.Collection("users").Doc(userID).Collection("skills").Doc(skillID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete skill: %w", err)
	}

	return nil
}

// toUpdatePaths converts a map to firestore.Update paths
func toUpdatePaths(updates map[string]interface{}) []firestore.Update {
	var paths []firestore.Update
	for key, val := range updates {
		paths = append(paths, firestore.Update{Path: key, Value: val})
	}
	return paths
}
```

- [ ] **Step 4: Run tests (with Firestore emulator)**

```bash
# Start Firestore emulator (if not running)
# gcloud beta emulators firestore start

FIRESTORE_EMULATOR_HOST=localhost:8080 go test ./pkg/skills -v -run TestCreateUserSkill
```

Expected: PASS (with emulator running)

- [ ] **Step 5: Commit**

```bash
git add pkg/skills/firestore.go pkg/skills/firestore_test.go
git commit -m "feat: implement user skills Firestore repository"
```

---

### Task 3: Hub API Endpoints for User Skills

**Files:**
- Create: `synchestra-hub/src/api/skills.go` (or modify existing API router)
- Create: `synchestra-hub/tests/api/skills_test.go`

**Context:**
Hub exposes REST endpoints for user skill CRUD. These are called by the Angular frontend.

- [ ] **Step 1: Write API endpoint tests**

```go
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateUserSkillEndpoint(t *testing.T) {
	// Setup
	handler := setupTestHandler() // returns *http.ServeMux with routes

	// Request body
	body := map[string]interface{}{
		"name":        "Code Review",
		"description": "Reviews pull requests",
		"content":     "# Code Review\n\n...",
		"tags":        []string{"review"},
	}
	bodyBytes, _ := json.Marshal(body)

	// Make request
	req := httptest.NewRequest("POST", "/api/v1/users/user123/skills", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["skill_id"] == "" {
		t.Error("expected skill_id in response")
	}
}

func TestListUserSkillsEndpoint(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/users/user123/skills", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["skills"] == nil {
		t.Error("expected skills in response")
	}
}

func TestGetUserSkillEndpoint(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/v1/users/user123/skills/code-review", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("expected status 200 or 404, got %d", w.Code)
	}
}

func setupTestHandler() *http.ServeMux {
	// In real implementation, wire up dependencies
	return http.NewServeMux()
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./synchestra-hub/tests/api -v -run TestCreateUserSkillEndpoint
```

Expected: FAIL (handlers not defined)

- [ ] **Step 3: Write API handlers**

```go
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"synchestra/pkg/skills"
)

// CreateUserSkillRequest is the request body for creating a skill.
type CreateUserSkillRequest struct {
	SkillID     string   `json:"skill_id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags,omitempty"`
	Version     string   `json:"version,omitempty"`
}

// SkillResponse is the response body for a skill.
type SkillResponse struct {
	SkillID     string `json:"skill_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// UserSkillsHandler handles all user skill endpoints.
type UserSkillsHandler struct {
	skillRepo *skills.FirestoreSkillRepository
}

// HandleCreateUserSkill creates a new user skill.
// POST /api/v1/users/{user_id}/skills
func (h *UserSkillsHandler) HandleCreateUserSkill(w http.ResponseWriter, r *http.Request) {
	userID := extractUserID(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateUserSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Auto-generate skill_id from name if not provided
	if req.SkillID == "" {
		req.SkillID = slugify(req.Name)
	}

	skill := &skills.UserSkill{
		SkillID:     req.SkillID,
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Tags:        req.Tags,
		Version:     req.Version,
	}

	ctx := r.Context()
	docID, err := h.skillRepo.CreateUserSkill(ctx, userID, skill)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create skill: %v", err), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"skill_id": req.SkillID,
		"doc_id":   docID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// HandleListUserSkills lists all user skills.
// GET /api/v1/users/{user_id}/skills
func (h *UserSkillsHandler) HandleListUserSkills(w http.ResponseWriter, r *http.Request) {
	userID := extractUserID(r)
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	skillsList, err := h.skillRepo.ListUserSkills(ctx, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list skills: %v", err), http.StatusInternalServerError)
		return
	}

	if skillsList == nil {
		skillsList = []*skills.UserSkill{}
	}

	resp := map[string]interface{}{
		"skills": skillsList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetUserSkill gets a specific user skill.
// GET /api/v1/users/{user_id}/skills/{skill_id}
func (h *UserSkillsHandler) HandleGetUserSkill(w http.ResponseWriter, r *http.Request) {
	userID := extractUserID(r)
	skillID := extractSkillID(r)
	if userID == "" || skillID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	skill, err := h.skillRepo.GetUserSkill(ctx, userID, skillID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get skill: %v", err), http.StatusInternalServerError)
		return
	}

	if skill == nil {
		http.Error(w, "skill not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skill)
}

// Helper: extract user ID from request context/token
func extractUserID(r *http.Request) string {
	// In real implementation, extract from JWT token
	return "user123" // placeholder
}

// Helper: extract skill_id from URL path
func extractSkillID(r *http.Request) string {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// Helper: convert name to slug
func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
```

- [ ] **Step 4: Register handlers in router**

In your main router setup (e.g., `main.go` or `api/router.go`):

```go
func setupRoutes(skillRepo *skills.FirestoreSkillRepository) *http.ServeMux {
	mux := http.NewServeMux()

	skillHandler := &UserSkillsHandler{skillRepo: skillRepo}

	mux.HandleFunc("POST /api/v1/users/{user_id}/skills", skillHandler.HandleCreateUserSkill)
	mux.HandleFunc("GET /api/v1/users/{user_id}/skills", skillHandler.HandleListUserSkills)
	mux.HandleFunc("GET /api/v1/users/{user_id}/skills/{skill_id}", skillHandler.HandleGetUserSkill)

	return mux
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./synchestra-hub/tests/api -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add synchestra-hub/api/skills.go synchestra-hub/tests/api/skills_test.go
git commit -m "feat: add user skills API endpoints (CRUD)"
```

---

## Phase 3: Project Skills (Git-Backed)

### Task 4: Project Skills Git Integration

**Files:**
- Create: `pkg/skills/project.go`
- Create: `pkg/skills/project_test.go`

**Context:**
Project skills are stored in `synchestra/skills/{skill_id}/SKILL.md`. This task handles reading skills from disk and writing commits when adding skills from user collection.

- [ ] **Step 1: Write tests for project skill operations**

```go
package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverProjectSkills(t *testing.T) {
	// Setup temp project directory
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "synchestra", "skills")
	os.MkdirAll(skillsDir, 0755)

	// Create two skills
	createTestSkill(t, skillsDir, "code-review", "---\nskill_id: code-review\nname: Code Review\ndescription: Reviews PRs\norigin: code-review@user123@github.com\n---\n# Code Review")
	createTestSkill(t, skillsDir, "go-test", "---\nskill_id: go-test\nname: Go Testing\ndescription: Tests Go\n---\n# Go Testing")

	// Discover skills
	skills, err := DiscoverProjectSkills(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}

	ids := []string{}
	for _, s := range skills {
		ids = append(ids, s.SkillID)
	}

	if !contains(ids, "code-review") || !contains(ids, "go-test") {
		t.Errorf("expected skills code-review and go-test, got %v", ids)
	}
}

func TestAddSkillToProject(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "synchestra", "skills")
	os.MkdirAll(skillsDir, 0755)

	skill := &Skill{
		SkillID:     "new-skill",
		Name:        "New Skill",
		Description: "A new skill",
		Origin:      "new-skill@user123@github.com",
		Content:     []byte("# New Skill"),
	}

	err := AddSkillToProject(tmpDir, skill)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	skillFile := filepath.Join(skillsDir, "new-skill", "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Errorf("expected skill file at %s", skillFile)
	}

	// Verify content
	content, _ := os.ReadFile(skillFile)
	if !contains(string(content), "skill_id: new-skill") {
		t.Errorf("skill content missing frontmatter")
	}
}

func createTestSkill(t *testing.T, skillsDir, skillID, content string) {
	dir := filepath.Join(skillsDir, skillID)
	os.MkdirAll(dir, 0755)
	file := filepath.Join(dir, "SKILL.md")
	os.WriteFile(file, []byte(content), 0644)
}

func contains(arr []string, s string) bool {
	for _, item := range arr {
		if item == s {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./pkg/skills -v -run TestDiscoverProjectSkills
```

Expected: FAIL

- [ ] **Step 3: Write project skills operations**

```go
package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiscoverProjectSkills finds all skills in a project's synchestra/skills directory.
func DiscoverProjectSkills(projectRoot string) ([]*Skill, error) {
	skillsDir := filepath.Join(projectRoot, "synchestra", "skills")

	// Check if directory exists
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return nil, nil // no skills directory
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []*Skill

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillFile := filepath.Join(skillsDir, skillID, "SKILL.md")

		// Read SKILL.md
		content, err := os.ReadFile(skillFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read skill %s: %w", skillID, err)
		}

		// Parse skill
		skill, err := ParseSkill(skillID, string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse skill %s: %w", skillID, err)
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// AddSkillToProject writes a skill to the project's synchestra/skills directory.
func AddSkillToProject(projectRoot string, skill *Skill) error {
	skillDir := filepath.Join(projectRoot, "synchestra", "skills", skill.SkillID)

	// Create directory
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write SKILL.md
	skillFile := filepath.Join(skillDir, "SKILL.md")
	serialized := skill.Serialize()

	if err := os.WriteFile(skillFile, []byte(serialized), 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}

// RemoveSkillFromProject deletes a skill directory from the project.
func RemoveSkillFromProject(projectRoot string, skillID string) error {
	skillDir := filepath.Join(projectRoot, "synchestra", "skills", skillID)

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./pkg/skills -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/skills/project.go pkg/skills/project_test.go
git commit -m "feat: implement project skills git operations"
```

---

### Task 5: Git Commit Integration for Adding Skills

**Files:**
- Create: `pkg/gitops/skill_commit.go`
- Create: `pkg/gitops/skill_commit_test.go`

**Context:**
When a user adds a skill to a project via Hub UI, we need to create a git commit. This task handles that.

- [ ] **Step 1: Write test for commit creation**

```go
package gitops

import (
	"testing"
)

func TestCreateSkillAddCommit(t *testing.T) {
	tmpDir := t.TempDir()
	repo := initTestRepo(t, tmpDir)

	skillID := "code-review"
	skillPath := "synchestra/skills/code-review/SKILL.md"
	skillContent := "---\nskill_id: code-review\nname: Code Review\n---\n# Code Review"

	commit, err := CreateSkillAddCommit(repo, skillID, skillContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if commit == "" {
		t.Fatal("expected commit hash")
	}

	// Verify file exists
	file, _ := repo.Workdir()
	if _, err := os.Stat(filepath.Join(file, skillPath)); os.IsNotExist(err) {
		t.Errorf("expected skill file at %s", skillPath)
	}
}

func initTestRepo(t *testing.T, tmpDir string) *git.Repository {
	repo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}
	return repo
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/gitops -v -run TestCreateSkillAddCommit
```

Expected: FAIL

- [ ] **Step 3: Write commit creation logic**

```go
package gitops

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// CreateSkillAddCommit creates a git commit for adding a skill to the project.
// Returns the commit hash.
func CreateSkillAddCommit(repo *git.Repository, skillID string, skillContent string) (string, error) {
	// Get worktree
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Write skill file
	skillPath := filepath.Join("synchestra", "skills", skillID, "SKILL.md")
	skillDir := filepath.Join(wt.Filesystem.Root(), filepath.Dir(skillPath))

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}

	fullPath := filepath.Join(wt.Filesystem.Root(), skillPath)
	if err := os.WriteFile(fullPath, []byte(skillContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write skill file: %w", err)
	}

	// Add to git
	_, err = wt.Add(skillPath)
	if err != nil {
		return "", fmt.Errorf("failed to add file to git: %w", err)
	}

	// Create commit
	sig := &object.Signature{
		Name:  "Synchestra Bot",
		Email: "bot@synchestra.io",
		When:  object.Now(),
	}

	commitMsg := fmt.Sprintf("feat: add %s skill", skillID)
	hash, err := wt.Commit(commitMsg, &git.CommitOptions{Author: sig})
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	return hash.String(), nil
}
```

- [ ] **Step 4: Run test**

```bash
go test ./pkg/gitops -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/gitops/skill_commit.go pkg/gitops/skill_commit_test.go
git commit -m "feat: implement git commit creation for adding skills"
```

---

## Phase 4: Hub UI - User Skills & Project Skills

### Task 6: Angular Skill Model and Service

**Files:**
- Create: `synchestra-hub/src/app/models/skill.model.ts`
- Create: `synchestra-hub/src/app/services/skill.service.ts`

- [ ] **Step 1: Create skill model**

```typescript
// src/app/models/skill.model.ts

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

export interface SkillListResponse {
  skills: Skill[];
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

- [ ] **Step 2: Create skill service**

```typescript
// src/app/services/skill.service.ts

import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Skill, CreateSkillRequest, SkillListResponse } from '../models/skill.model';

@Injectable({
  providedIn: 'root'
})
export class SkillService {
  private apiUrl = '/api/v1/users';

  constructor(private http: HttpClient) {}

  // User Skills
  createUserSkill(userId: string, request: CreateSkillRequest): Observable<Skill> {
    return this.http.post<Skill>(`${this.apiUrl}/${userId}/skills`, request);
  }

  getUserSkills(userId: string): Observable<SkillListResponse> {
    return this.http.get<SkillListResponse>(`${this.apiUrl}/${userId}/skills`);
  }

  getUserSkill(userId: string, skillId: string): Observable<Skill> {
    return this.http.get<Skill>(`${this.apiUrl}/${userId}/skills/${skillId}`);
  }

  updateUserSkill(userId: string, skillId: string, updates: Partial<Skill>): Observable<void> {
    return this.http.put<void>(`${this.apiUrl}/${userId}/skills/${skillId}`, updates);
  }

  deleteUserSkill(userId: string, skillId: string): Observable<void> {
    return this.http.delete<void>(`${this.apiUrl}/${userId}/skills/${skillId}`);
  }

  // Project Skills
  addSkillToProject(projectId: string, skillId: string, userId: string): Observable<void> {
    return this.http.post<void>(`/api/v1/projects/${projectId}/skills/add`, {
      skill_id: skillId,
      user_id: userId,
      source: 'user_collection'
    });
  }

  removeSkillFromProject(projectId: string, skillId: string): Observable<void> {
    return this.http.delete<void>(`/api/v1/projects/${projectId}/skills/${skillId}`);
  }

  saveProjectSkillToFavorites(projectId: string, skillId: string): Observable<void> {
    return this.http.post<void>(`/api/v1/projects/${projectId}/skills/${skillId}/save-favorite`, {});
  }
}
```

- [ ] **Step 3: Commit**

```bash
git add synchestra-hub/src/app/models/skill.model.ts synchestra-hub/src/app/services/skill.service.ts
git commit -m "feat: add skill model and service layer"
```

---

### Task 7: User Skills Dashboard Component

**Files:**
- Create: `synchestra-hub/src/app/features/skills/user-skills-dashboard/user-skills-dashboard.component.ts`
- Create: `synchestra-hub/src/app/features/skills/user-skills-dashboard/user-skills-dashboard.component.html`
- Create: `synchestra-hub/src/app/features/skills/user-skills-dashboard/user-skills-dashboard.component.css`

- [ ] **Step 1: Create component TypeScript**

```typescript
// src/app/features/skills/user-skills-dashboard/user-skills-dashboard.component.ts

import { Component, OnInit } from '@angular/core';
import { SkillService } from '../../../services/skill.service';
import { Skill } from '../../../models/skill.model';

@Component({
  selector: 'app-user-skills-dashboard',
  templateUrl: './user-skills-dashboard.component.html',
  styleUrls: ['./user-skills-dashboard.component.css']
})
export class UserSkillsDashboardComponent implements OnInit {
  skills: Skill[] = [];
  filteredSkills: Skill[] = [];
  searchText = '';
  selectedTags: string[] = [];
  showCreateModal = false;
  loading = false;
  error: string | null = null;

  constructor(private skillService: SkillService) {}

  ngOnInit(): void {
    this.loadSkills();
  }

  loadSkills(): void {
    this.loading = true;
    this.error = null;

    // Get current user ID from auth service
    const userId = this.getCurrentUserId();

    this.skillService.getUserSkills(userId).subscribe({
      next: (response) => {
        this.skills = response.skills || [];
        this.filterSkills();
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Failed to load skills';
        this.loading = false;
      }
    });
  }

  filterSkills(): void {
    this.filteredSkills = this.skills.filter(skill => {
      const matchesSearch = !this.searchText ||
        skill.name.toLowerCase().includes(this.searchText.toLowerCase()) ||
        skill.description.toLowerCase().includes(this.searchText.toLowerCase());

      const matchesTags = this.selectedTags.length === 0 ||
        this.selectedTags.some(tag => skill.tags?.includes(tag));

      return matchesSearch && matchesTags;
    });
  }

  onSearchChange(text: string): void {
    this.searchText = text;
    this.filterSkills();
  }

  onTagSelect(tag: string): void {
    if (this.selectedTags.includes(tag)) {
      this.selectedTags = this.selectedTags.filter(t => t !== tag);
    } else {
      this.selectedTags.push(tag);
    }
    this.filterSkills();
  }

  openCreateModal(): void {
    this.showCreateModal = true;
  }

  closeCreateModal(): void {
    this.showCreateModal = false;
  }

  onSkillCreated(skill: Skill): void {
    this.skills.push(skill);
    this.filterSkills();
    this.closeCreateModal();
  }

  deleteSkill(skillId: string): void {
    if (!confirm('Delete this skill?')) return;

    const userId = this.getCurrentUserId();
    this.skillService.deleteUserSkill(userId, skillId).subscribe({
      next: () => {
        this.skills = this.skills.filter(s => s.skill_id !== skillId);
        this.filterSkills();
      },
      error: () => this.error = 'Failed to delete skill'
    });
  }

  editSkill(skillId: string): void {
    // Navigate to edit page or open modal
  }

  private getCurrentUserId(): string {
    // Get from auth service
    return 'user123'; // placeholder
  }
}
```

- [ ] **Step 2: Create component template**

```html
<!-- src/app/features/skills/user-skills-dashboard/user-skills-dashboard.component.html -->

<div class="skills-dashboard">
  <div class="header">
    <h1>My Skills</h1>
    <button (click)="openCreateModal()" class="btn-primary">+ Create Skill</button>
  </div>

  <div class="controls">
    <input type="text"
           placeholder="Search skills..."
           [(ngModel)]="searchText"
           (input)="onSearchChange($event.target.value)"
           class="search-input">
  </div>

  <div *ngIf="error" class="error-message">{{ error }}</div>

  <div *ngIf="loading" class="loading">Loading skills...</div>

  <div *ngIf="!loading && filteredSkills.length === 0" class="empty-state">
    <p>No skills yet. <a (click)="openCreateModal()">Create one</a></p>
  </div>

  <div *ngIf="!loading && filteredSkills.length > 0" class="skills-grid">
    <div *ngFor="let skill of filteredSkills" class="skill-card">
      <div class="skill-header">
        <h3>{{ skill.name }}</h3>
        <div class="actions">
          <button (click)="editSkill(skill.skill_id)" title="Edit">✎</button>
          <button (click)="deleteSkill(skill.skill_id)" title="Delete">✕</button>
        </div>
      </div>

      <p class="description">{{ skill.description }}</p>

      <div *ngIf="skill.tags && skill.tags.length > 0" class="tags">
        <span *ngFor="let tag of skill.tags" class="tag">{{ tag }}</span>
      </div>

      <div class="meta">
        <span *ngIf="skill.version">v{{ skill.version }}</span>
        <span *ngIf="skill.origin" class="origin">Imported from {{ skill.origin }}</span>
      </div>
    </div>
  </div>

  <app-skill-editor-modal
    *ngIf="showCreateModal"
    [mode]="'create'"
    (close)="closeCreateModal()"
    (save)="onSkillCreated($event)">
  </app-skill-editor-modal>
</div>
```

- [ ] **Step 3: Create component styles**

```css
/* src/app/features/skills/user-skills-dashboard/user-skills-dashboard.component.css */

.skills-dashboard {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.header h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
}

.controls {
  margin-bottom: 24px;
}

.search-input {
  width: 100%;
  max-width: 400px;
  padding: 10px 16px;
  border: 1px solid #ccc;
  border-radius: 6px;
  font-size: 14px;
}

.skills-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.skill-card {
  border: 1px solid #eee;
  border-radius: 8px;
  padding: 16px;
  background: white;
  transition: box-shadow 0.2s;
}

.skill-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.skill-header {
  display: flex;
  justify-content: space-between;
  align-items: start;
  margin-bottom: 12px;
}

.skill-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.actions button {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px 8px;
  color: #666;
}

.actions button:hover {
  color: #000;
}

.description {
  margin: 8px 0;
  color: #666;
  font-size: 14px;
}

.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 12px 0;
}

.tag {
  background: #f0f0f0;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  color: #333;
}

.meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #999;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.error-message {
  padding: 12px;
  background: #fee;
  color: #c33;
  border-radius: 4px;
  margin-bottom: 16px;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  color: #999;
}

.empty-state a {
  color: #0066cc;
  cursor: pointer;
}

.loading {
  text-align: center;
  padding: 48px;
  color: #999;
}

.btn-primary {
  background: #0066cc;
  color: white;
  border: none;
  padding: 10px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.btn-primary:hover {
  background: #0052a3;
}
```

- [ ] **Step 4: Commit**

```bash
git add synchestra-hub/src/app/features/skills/user-skills-dashboard/
git commit -m "feat: implement user skills dashboard component"
```

---

### Task 8: Skill Editor Component (Shared)

**Files:**
- Create: `synchestra-hub/src/app/features/skills/skill-editor/skill-editor.component.ts`
- Create: `synchestra-hub/src/app/features/skills/skill-editor/skill-editor.component.html`

- [ ] **Step 1: Create editor component**

```typescript
// src/app/features/skills/skill-editor/skill-editor.component.ts

import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Skill, CreateSkillRequest } from '../../../models/skill.model';
import { SkillService } from '../../../services/skill.service';

@Component({
  selector: 'app-skill-editor',
  templateUrl: './skill-editor.component.html',
  styleUrls: ['./skill-editor.component.css']
})
export class SkillEditorComponent {
  @Input() skill: Skill | null = null;
  @Input() mode: 'create' | 'edit' = 'create';
  @Output() save = new EventEmitter<Skill>();
  @Output() cancel = new EventEmitter<void>();

  form = {
    skill_id: '',
    name: '',
    description: '',
    content: '',
    tags: '',
    version: ''
  };

  saving = false;
  error: string | null = null;

  constructor(private skillService: SkillService) {}

  ngOnInit(): void {
    if (this.skill) {
      this.form = {
        skill_id: this.skill.skill_id,
        name: this.skill.name,
        description: this.skill.description,
        content: this.skill.content,
        tags: (this.skill.tags || []).join(', '),
        version: this.skill.version || ''
      };
    }
  }

  onSave(): void {
    if (!this.validateForm()) return;

    this.saving = true;
    this.error = null;

    const request: CreateSkillRequest = {
      skill_id: this.form.skill_id || undefined,
      name: this.form.name,
      description: this.form.description,
      content: this.form.content,
      tags: this.form.tags ? this.form.tags.split(',').map(t => t.trim()) : undefined,
      version: this.form.version || undefined
    };

    const userId = this.getCurrentUserId();

    if (this.mode === 'create') {
      this.skillService.createUserSkill(userId, request).subscribe({
        next: (skill) => {
          this.saving = false;
          this.save.emit(skill);
        },
        error: (err) => {
          this.error = 'Failed to save skill';
          this.saving = false;
        }
      });
    } else if (this.skill) {
      this.skillService.updateUserSkill(userId, this.skill.skill_id, request).subscribe({
        next: () => {
          this.saving = false;
          this.save.emit({ ...this.skill, ...request } as Skill);
        },
        error: (err) => {
          this.error = 'Failed to update skill';
          this.saving = false;
        }
      });
    }
  }

  onCancel(): void {
    this.cancel.emit();
  }

  private validateForm(): boolean {
    if (!this.form.name.trim()) {
      this.error = 'Name is required';
      return false;
    }
    if (!this.form.description.trim()) {
      this.error = 'Description is required';
      return false;
    }
    if (!this.form.content.trim()) {
      this.error = 'Content is required';
      return false;
    }
    return true;
  }

  private getCurrentUserId(): string {
    return 'user123'; // from auth service
  }
}
```

- [ ] **Step 2: Create editor template**

```html
<!-- src/app/features/skills/skill-editor/skill-editor.component.html -->

<div class="editor">
  <h2>{{ mode === 'create' ? 'Create Skill' : 'Edit Skill' }}</h2>

  <div *ngIf="error" class="error-message">{{ error }}</div>

  <div class="form-group">
    <label>Name *</label>
    <input type="text" [(ngModel)]="form.name" placeholder="e.g., Code Review">
  </div>

  <div class="form-group">
    <label>Description *</label>
    <textarea [(ngModel)]="form.description" placeholder="Brief description of what this skill does"></textarea>
  </div>

  <div class="form-group">
    <label>Skill Content *</label>
    <textarea [(ngModel)]="form.content" class="monospace" placeholder="Paste SKILL.md content (with or without frontmatter)"></textarea>
  </div>

  <div class="form-row">
    <div class="form-group">
      <label>Tags</label>
      <input type="text" [(ngModel)]="form.tags" placeholder="e.g., review, go, testing (comma-separated)">
    </div>

    <div class="form-group">
      <label>Version</label>
      <input type="text" [(ngModel)]="form.version" placeholder="e.g., 1.0">
    </div>
  </div>

  <div class="actions">
    <button (click)="onCancel()" class="btn-secondary">Cancel</button>
    <button (click)="onSave()" [disabled]="saving" class="btn-primary">
      {{ saving ? 'Saving...' : 'Save' }}
    </button>
  </div>
</div>
```

- [ ] **Step 3: Commit**

```bash
git add synchestra-hub/src/app/features/skills/skill-editor/
git commit -m "feat: implement skill editor component (create/edit)"
```

---

### Task 9: Project Skills Management Component

**Files:**
- Create: `synchestra-hub/src/app/features/projects/project-skills/project-skills.component.ts`
- Create: `synchestra-hub/src/app/features/projects/project-skills/project-skills.component.html`

- [ ] **Step 1: Create project skills component**

```typescript
// src/app/features/projects/project-skills/project-skills.component.ts

import { Component, Input, OnInit } from '@angular/core';
import { SkillService } from '../../../services/skill.service';
import { Skill } from '../../../models/skill.model';

@Component({
  selector: 'app-project-skills',
  templateUrl: './project-skills.component.html',
  styleUrls: ['./project-skills.component.css']
})
export class ProjectSkillsComponent implements OnInit {
  @Input() projectId: string = '';

  projectSkills: Skill[] = [];
  userSkills: Skill[] = [];
  showAddModal = false;
  showCreateModal = false;
  loading = false;
  error: string | null = null;

  constructor(private skillService: SkillService) {}

  ngOnInit(): void {
    this.loadProjectSkills();
    this.loadUserSkills();
  }

  loadProjectSkills(): void {
    // Call API to get project skills
    // this.skillService.getProjectSkills(this.projectId).subscribe(...)
  }

  loadUserSkills(): void {
    const userId = this.getCurrentUserId();
    this.skillService.getUserSkills(userId).subscribe({
      next: (response) => {
        this.userSkills = response.skills || [];
      },
      error: () => this.error = 'Failed to load user skills'
    });
  }

  openAddModal(): void {
    this.showAddModal = true;
  }

  openCreateModal(): void {
    this.showCreateModal = true;
  }

  closeModals(): void {
    this.showAddModal = false;
    this.showCreateModal = false;
  }

  addSkillFromCollection(skillId: string): void {
    const userId = this.getCurrentUserId();
    this.skillService.addSkillToProject(this.projectId, skillId, userId).subscribe({
      next: () => {
        this.closeModals();
        this.loadProjectSkills();
      },
      error: () => this.error = 'Failed to add skill to project'
    });
  }

  onSkillCreated(skill: Skill): void {
    this.projectSkills.push(skill);
    this.closeModals();
  }

  removeSkill(skillId: string): void {
    if (!confirm('Remove this skill from the project?')) return;

    this.skillService.removeSkillFromProject(this.projectId, skillId).subscribe({
      next: () => {
        this.projectSkills = this.projectSkills.filter(s => s.skill_id !== skillId);
      },
      error: () => this.error = 'Failed to remove skill'
    });
  }

  saveToFavorites(skillId: string): void {
    this.skillService.saveProjectSkillToFavorites(this.projectId, skillId).subscribe({
      next: () => {
        alert('Skill saved to your collection');
      },
      error: () => this.error = 'Failed to save skill'
    });
  }

  private getCurrentUserId(): string {
    return 'user123'; // from auth service
  }
}
```

- [ ] **Step 2: Create project skills template**

```html
<!-- src/app/features/projects/project-skills/project-skills.component.html -->

<div class="project-skills-section">
  <div class="header">
    <h2>Project Skills</h2>
    <div class="actions">
      <button (click)="openAddModal()" class="btn-secondary">+ Add from Collection</button>
      <button (click)="openCreateModal()" class="btn-primary">+ Create New</button>
    </div>
  </div>

  <div *ngIf="error" class="error-message">{{ error }}</div>

  <div *ngIf="projectSkills.length === 0" class="empty-state">
    <p>No skills configured yet.</p>
  </div>

  <div *ngIf="projectSkills.length > 0" class="skills-list">
    <div *ngFor="let skill of projectSkills" class="skill-item">
      <div class="skill-info">
        <h3>{{ skill.name }}</h3>
        <p>{{ skill.description }}</p>
        <div *ngIf="skill.origin" class="origin">From: {{ skill.origin }}</div>
      </div>

      <div class="skill-actions">
        <button (click)="saveToFavorites(skill.skill_id)" title="Save to My Collection">★</button>
        <button (click)="removeSkill(skill.skill_id)" title="Remove">✕</button>
      </div>
    </div>
  </div>

  <!-- Add from Collection Modal -->
  <div *ngIf="showAddModal" class="modal-overlay" (click)="closeModals()">
    <div class="modal" (click)="$event.stopPropagation()">
      <h3>Add Skill from Collection</h3>

      <div class="user-skills-list">
        <div *ngFor="let skill of userSkills" class="skill-option">
          <div class="skill-info">
            <strong>{{ skill.name }}</strong>
            <p>{{ skill.description }}</p>
          </div>
          <button (click)="addSkillFromCollection(skill.skill_id)" class="btn-primary">Add</button>
        </div>
      </div>

      <div class="actions">
        <button (click)="closeModals()" class="btn-secondary">Close</button>
      </div>
    </div>
  </div>

  <!-- Create New Skill Modal -->
  <div *ngIf="showCreateModal" class="modal-overlay" (click)="closeModals()">
    <div class="modal" (click)="$event.stopPropagation()">
      <app-skill-editor
        [mode]="'create'"
        (save)="onSkillCreated($event)"
        (cancel)="closeModals()">
      </app-skill-editor>
    </div>
  </div>
</div>
```

- [ ] **Step 3: Commit**

```bash
git add synchestra-hub/src/app/features/projects/project-skills/
git commit -m "feat: implement project skills management component"
```

---

## Phase 5: Runner Integration

### Task 10: Runner - Copy Skills to .claude/skills/

**Files:**
- Modify: `pkg/runner/clone.go` (or relevant runner setup code)
- Create: `pkg/runner/skills.go` (if not already in clone.go)

- [ ] **Step 1: Write test for skill copying**

```go
package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyProjectSkillsToClaude(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project with skills
	projectDir := filepath.Join(tmpDir, "project")
	skillsDir := filepath.Join(projectDir, "synchestra", "skills")
	os.MkdirAll(skillsDir, 0755)

	// Create test skill
	skillDir := filepath.Join(skillsDir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	skillFile := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(skillFile, []byte("---\nskill_id: test-skill\nname: Test\n---\n# Test"), 0644)

	// Create target .claude/skills directory
	claudeSkillsDir := filepath.Join(tmpDir, ".claude", "skills")

	// Copy skills
	err := CopyProjectSkillsToClaude(projectDir, claudeSkillsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify copied
	copiedFile := filepath.Join(claudeSkillsDir, "test-skill", "SKILL.md")
	if _, err := os.Stat(copiedFile); os.IsNotExist(err) {
		t.Errorf("expected skill copied to %s", copiedFile)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/runner -v -run TestCopyProjectSkillsToClaude
```

Expected: FAIL

- [ ] **Step 3: Write skill copying function**

```go
package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyProjectSkillsToClaude copies all skills from project's synchestra/skills/
// to the target .claude/skills/ directory.
func CopyProjectSkillsToClaude(projectRoot string, claudeSkillsDir string) error {
	projectSkillsDir := filepath.Join(projectRoot, "synchestra", "skills")

	// Check if source exists
	if _, err := os.Stat(projectSkillsDir); os.IsNotExist(err) {
		return nil // no skills to copy
	}

	// Create target directory
	if err := os.MkdirAll(claudeSkillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude/skills directory: %w", err)
	}

	// Read source directory
	entries, err := os.ReadDir(projectSkillsDir)
	if err != nil {
		return fmt.Errorf("failed to read project skills: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		sourcePath := filepath.Join(projectSkillsDir, skillID)
		targetPath := filepath.Join(claudeSkillsDir, skillID)

		// Create target skill directory
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create skill directory: %w", err)
		}

		// Copy SKILL.md file
		sourceFile := filepath.Join(sourcePath, "SKILL.md")
		targetFile := filepath.Join(targetPath, "SKILL.md")

		if err := copyFile(sourceFile, targetFile); err != nil {
			return fmt.Errorf("failed to copy skill %s: %w", skillID, err)
		}
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return err
	}

	return nil
}
```

- [ ] **Step 4: Run test**

```bash
go test ./pkg/runner -v
```

Expected: PASS

- [ ] **Step 5: Modify runner to call CopyProjectSkillsToClaude**

In the runner's clone/setup phase (wherever project repo is cloned), add:

```go
// After cloning project repo...
projectRoot := getProjectRoot() // the cloned project directory
claudeSkillsDir := filepath.Join(projectRoot, ".claude", "skills")

if err := CopyProjectSkillsToClaude(projectRoot, claudeSkillsDir); err != nil {
	return fmt.Errorf("failed to setup skills: %w", err)
}
```

- [ ] **Step 6: Commit**

```bash
git add pkg/runner/skills.go pkg/runner/clone.go
git commit -m "feat: copy project skills to .claude/skills/ at runtime"
```

---

## Phase 6: Integration & Final Testing

### Task 11: End-to-End Integration Test

**Files:**
- Create: `tests/e2e/user_project_skills_test.go`

- [ ] **Step 1: Write E2E test**

```go
package e2e

import (
	"context"
	"testing"
	"time"

	"synchestra/pkg/skills"
)

func TestE2EUserProjectSkillsWorkflow(t *testing.T) {
	// 1. Create user skill in Firestore
	skillRepo := setupFirestore(t)
	userID := "test-user"

	userSkill := &skills.UserSkill{
		SkillID:     "code-review",
		Name:        "Code Review",
		Description: "Reviews PRs",
		Content:     "# Code Review\n\n...",
		Tags:        []string{"review"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := skillRepo.CreateUserSkill(ctx, userID, userSkill)
	if err != nil {
		t.Fatalf("failed to create user skill: %v", err)
	}

	// 2. Verify skill is in Firestore
	retrieved, err := skillRepo.GetUserSkill(ctx, userID, "code-review")
	if err != nil {
		t.Fatalf("failed to get user skill: %v", err)
	}

	if retrieved.Name != "Code Review" {
		t.Errorf("expected name 'Code Review', got '%s'", retrieved.Name)
	}

	// 3. Add skill to project (simulated: write to filesystem)
	projectRoot := t.TempDir()
	projectSkill := &skills.Skill{
		SkillID:     "code-review",
		Name:        "Code Review",
		Description: "Reviews PRs",
		Origin:      "code-review@test-user@github.com",
		Content:     []byte("# Code Review\n\n..."),
	}

	if err := skills.AddSkillToProject(projectRoot, projectSkill); err != nil {
		t.Fatalf("failed to add skill to project: %v", err)
	}

	// 4. Verify skill was written
	discovered, err := skills.DiscoverProjectSkills(projectRoot)
	if err != nil {
		t.Fatalf("failed to discover project skills: %v", err)
	}

	if len(discovered) != 1 || discovered[0].SkillID != "code-review" {
		t.Errorf("expected skill not found in project")
	}

	// 5. Copy to .claude/skills/
	claudeDir := t.TempDir()
	if err := runner.CopyProjectSkillsToClaude(projectRoot, claudeDir); err != nil {
		t.Fatalf("failed to copy skills: %v", err)
	}

	// 6. Verify .claude/skills/ has the skill
	claudeSkills, err := skills.DiscoverProjectSkills(claudeDir)
	if err != nil {
		t.Fatalf("failed to discover .claude/skills: %v", err)
	}

	if len(claudeSkills) != 1 {
		t.Errorf("expected 1 skill in .claude/skills, got %d", len(claudeSkills))
	}

	t.Log("✓ E2E test passed: user skill → project skill → .claude/skills")
}

func setupFirestore(t *testing.T) *skills.FirestoreSkillRepository {
	// Use Firestore emulator
	client := // initialize with emulator
	return skills.NewFirestoreSkillRepository(client)
}
```

- [ ] **Step 2: Run E2E test**

```bash
go test ./tests/e2e -v
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add tests/e2e/user_project_skills_test.go
git commit -m "test: add end-to-end integration test for user/project skills"
```

---

## Task 12: Documentation & Final Polish

**Files:**
- Create: `docs/features/user-project-skills.md` (or update existing)

- [ ] **Step 1: Write feature documentation**

```markdown
# User & Project Skills

## Overview

Synchestra provides a two-tier skill system:
- **User Skills:** Personal collection stored in Firestore, managed in Hub UI
- **Project Skills:** Git-backed skills in `synchestra/skills/`, copied to `.claude/skills/` at runtime

## User Skills

Create and organize a personal skill library in the Hub:

1. Go to **My Skills** in your user menu
2. Click **+ Create Skill**
3. Write or import your SKILL.md
4. Save to your collection

**Re-using Skills:** Use your saved skills in multiple projects by adding them to the project via the **Skills** tab.

## Project Skills

Configure skills that agents will use when executing work in your project:

### Add a Skill from Your Collection

1. Go to your project's **Skills** tab
2. Click **+ Add from Collection**
3. Select the skill and confirm

This copies the skill to your project repo in `synchestra/skills/`.

### Create a New Project-Only Skill

1. Go to your project's **Skills** tab
2. Click **+ Create New**
3. Write the skill definition
4. Save

The skill is added only to this project.

### Save a Project Skill to Your Collection

After creating a skill in a project, you can save it to your personal collection for reuse:

1. Click the **★** icon on the skill
2. Confirm save

The skill is now in your **My Skills** collection.

## At Runtime

When the Synchestra runner executes an agent for your project:

1. Runner clones your project repository
2. Discovers all skills in `synchestra/skills/`
3. Copies them to `.claude/skills/`
4. Agent loads skills and can use them during execution

Skills are available without any additional setup or network calls.

## Skill File Format

Skills are stored as `synchestra/skills/{skill_id}/SKILL.md` with frontmatter:

\`\`\`markdown
---
skill_id: code-review
name: Code Review
description: Reviews pull requests with actionable feedback
origin: code-review@user123@github.com
version: 1.0
tags: [review, quality]
---

# Code Review Skill

[Skill content following Claude Code format]
\`\`\`

**Frontmatter fields:**
- `skill_id` — unique identifier
- `name` — display name
- `description` — brief description
- `origin` — where the skill came from (optional)
- `version` — semantic version (optional)
- `tags` — searchable tags (optional)
```

- [ ] **Step 2: Commit**

```bash
git add docs/features/user-project-skills.md
git commit -m "docs: add user and project skills feature documentation"
```

---

## Summary

**Complete Implementation Path:**

| Phase | Focus | Output |
|-------|-------|--------|
| 1 | Core infrastructure | Skill model, validation, parsing |
| 2 | User skills | Firestore CRUD, Hub API, UI dashboard |
| 3 | Project skills | Git operations, commit integration, Hub API |
| 4 | Hub frontend | Shared components, CRUD UI, project skills management |
| 5 | Runtime | Copy skills to `.claude/skills/` |
| 6 | Testing & docs | E2E tests, feature documentation |

**Each task is 2-5 minutes of work with explicit code and commands.**

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-01-user-project-skills.md`.**

Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach would you prefer?