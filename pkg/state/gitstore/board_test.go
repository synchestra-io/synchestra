package gitstore

// Features depended on: state-store/backends/git

import (
	"testing"

	"github.com/synchestra-io/synchestra/pkg/state"
)

func TestBoardRoundTrip(t *testing.T) {
	bd := boardData{
		Rows: []boardRow{
			{
				Slug:      "setup-db",
				Status:    state.TaskStatusCompleted,
				DependsOn: nil,
				Branch:    "agent/run-1",
				Agent:     "Sonnet 4.5",
				Requester: "@alice",
				Time:      "2026-03-12 10:15",
			},
			{
				Slug:      "implement-api",
				Status:    state.TaskStatusInProgress,
				DependsOn: []string{"setup-db"},
				Branch:    "agent/run-2",
				Agent:     "Opus 4",
				Requester: "@alex",
				Time:      "2026-03-12 10:22",
			},
			{
				Slug:      "write-tests",
				Status:    state.TaskStatusQueued,
				DependsOn: []string{"implement-api"},
				Branch:    "",
				Agent:     "",
				Requester: "@alex",
				Time:      "",
			},
		},
	}

	rendered := renderBoard(bd)
	parsed, err := parseBoard(rendered)
	if err != nil {
		t.Fatalf("parseBoard failed: %v", err)
	}

	if len(parsed.Rows) != len(bd.Rows) {
		t.Fatalf("row count mismatch: got %d, want %d", len(parsed.Rows), len(bd.Rows))
	}

	for i, want := range bd.Rows {
		got := parsed.Rows[i]
		if got.Slug != want.Slug {
			t.Errorf("row %d: slug = %q, want %q", i, got.Slug, want.Slug)
		}
		if got.Status != want.Status {
			t.Errorf("row %d: status = %q, want %q", i, got.Status, want.Status)
		}
		if len(got.DependsOn) != len(want.DependsOn) {
			t.Errorf("row %d: deps count = %d, want %d", i, len(got.DependsOn), len(want.DependsOn))
		} else {
			for j := range want.DependsOn {
				if got.DependsOn[j] != want.DependsOn[j] {
					t.Errorf("row %d dep %d: got %q, want %q", i, j, got.DependsOn[j], want.DependsOn[j])
				}
			}
		}
		if got.Branch != want.Branch {
			t.Errorf("row %d: branch = %q, want %q", i, got.Branch, want.Branch)
		}
		if got.Agent != want.Agent {
			t.Errorf("row %d: agent = %q, want %q", i, got.Agent, want.Agent)
		}
		if got.Requester != want.Requester {
			t.Errorf("row %d: requester = %q, want %q", i, got.Requester, want.Requester)
		}
		if got.Time != want.Time {
			t.Errorf("row %d: time = %q, want %q", i, got.Time, want.Time)
		}
	}
}

func TestBoardParseEmpty(t *testing.T) {
	input := []byte("# Tasks\n\n| Task | Status | Depends on | Branch | Agent | Requester | Time |\n|---|---|---|---|---|---|---|\n")
	bd, err := parseBoard(input)
	if err != nil {
		t.Fatalf("parseBoard failed: %v", err)
	}
	if len(bd.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(bd.Rows))
	}
}

func TestBoardParseMultipleStatuses(t *testing.T) {
	input := "# Tasks\n\n" +
		"| Task | Status | Depends on | Branch | Agent | Requester | Time |\n" +
		"|---|---|---|---|---|---|---|\n" +
		"| [a](a/) | 📋 `planning` | — | — | — | — | — |\n" +
		"| [b](b/) | ❌ `failed` | a | `br-1` | GPT-5 | @bob | 2026-01-01 |\n" +
		"| [c](c/) | 🟡 `blocked` | a, b | — | — | @carol | — |\n"

	bd, err := parseBoard([]byte(input))
	if err != nil {
		t.Fatalf("parseBoard failed: %v", err)
	}
	if len(bd.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(bd.Rows))
	}

	if bd.Rows[0].Status != state.TaskStatusPlanning {
		t.Errorf("row 0: status = %q, want planning", bd.Rows[0].Status)
	}
	if bd.Rows[1].Status != state.TaskStatusFailed {
		t.Errorf("row 1: status = %q, want failed", bd.Rows[1].Status)
	}
	if bd.Rows[2].Status != state.TaskStatusBlocked {
		t.Errorf("row 2: status = %q, want blocked", bd.Rows[2].Status)
	}
}

func TestParseStatusCellAllStatuses(t *testing.T) {
	tests := []struct {
		cell string
		want state.TaskStatus
	}{
		{"📋 `planning`", state.TaskStatusPlanning},
		{"⏳ `queued`", state.TaskStatusQueued},
		{"🔒 `claimed`", state.TaskStatusClaimed},
		{"🔵 `in_progress`", state.TaskStatusInProgress},
		{"✅ `completed`", state.TaskStatusCompleted},
		{"❌ `failed`", state.TaskStatusFailed},
		{"🟡 `blocked`", state.TaskStatusBlocked},
		{"⛔ `aborted`", state.TaskStatusAborted},
	}

	for _, tt := range tests {
		got, err := parseStatusCell(tt.cell)
		if err != nil {
			t.Errorf("parseStatusCell(%q): %v", tt.cell, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseStatusCell(%q) = %q, want %q", tt.cell, got, tt.want)
		}
	}
}

func TestParseStatusCellInvalid(t *testing.T) {
	invalid := []string{
		"no backticks",
		"🔵 `unknown_status`",
		"",
		"`",
	}
	for _, cell := range invalid {
		_, err := parseStatusCell(cell)
		if err == nil {
			t.Errorf("parseStatusCell(%q): expected error, got nil", cell)
		}
	}
}

func TestRenderBoardStatuses(t *testing.T) {
	bd := boardData{
		Rows: []boardRow{
			{Slug: "a", Status: state.TaskStatusPlanning},
			{Slug: "b", Status: state.TaskStatusAborted},
		},
	}
	out := string(renderBoard(bd))
	if !containsSubstr(out, "📋 `planning`") {
		t.Error("missing planning status in rendered output")
	}
	if !containsSubstr(out, "⛔ `aborted`") {
		t.Error("missing aborted status in rendered output")
	}
}

func TestDependsOnParsing(t *testing.T) {
	tests := []struct {
		cell string
		want []string
	}{
		{"—", nil},
		{"setup-db", []string{"setup-db"}},
		{"a, b, c", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		got := parseDeps(tt.cell)
		if len(got) != len(tt.want) {
			t.Errorf("parseDeps(%q): got %v, want %v", tt.cell, got, tt.want)
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("parseDeps(%q)[%d]: got %q, want %q", tt.cell, i, got[i], tt.want[i])
			}
		}
	}
}

func TestSlugExtraction(t *testing.T) {
	tests := []struct {
		cell string
		want string
	}{
		{"[setup-db](setup-db/)", "setup-db"},
		{"[my-task](my-task/)", "my-task"},
		{"plain-text", "plain-text"},
	}
	for _, tt := range tests {
		got := extractSlug(tt.cell)
		if got != tt.want {
			t.Errorf("extractSlug(%q) = %q, want %q", tt.cell, got, tt.want)
		}
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
