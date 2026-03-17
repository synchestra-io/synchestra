package state

// Features implemented: state-store

import "time"

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskStatusPlanning   TaskStatus = "planning"
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusClaimed    TaskStatus = "claimed"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusAborted    TaskStatus = "aborted"
)

// Task represents a unit of work in the state store.
type Task struct {
	Slug      string
	Title     string
	Status    TaskStatus
	Parent    string // parent task slug, empty for root tasks
	DependsOn []string
	Run       string // agent run ID (populated when claimed/in_progress)
	Model     string // agent model ID (populated when claimed/in_progress)
	Requester string
	Reason    string // block/fail/abort reason
	Summary   string // completion summary
	CreatedAt time.Time
	ClaimedAt *time.Time
	UpdatedAt time.Time
}

// TaskCreateParams holds parameters for creating a new task.
type TaskCreateParams struct {
	Slug      string
	Title     string
	Parent    string
	DependsOn []string
	Requester string
}

// ClaimParams holds parameters for claiming a task.
type ClaimParams struct {
	Run   string
	Model string
}

// TaskFilter holds optional filters for listing tasks.
// Nil pointer fields mean "don't filter on this field."
type TaskFilter struct {
	Status *TaskStatus
	Parent *string
}

// BoardView represents a rendered task board.
type BoardView struct {
	Rows []BoardRow
}

// BoardRow represents a single row in the task board.
type BoardRow struct {
	Task      string
	Status    TaskStatus
	DependsOn []string
	Branch    string
	Agent     string
	Requester string
	StartedAt *time.Time
	Duration  *time.Duration
}

// ChatStatus represents the lifecycle state of a chat.
type ChatStatus string

const (
	ChatStatusCreated   ChatStatus = "created"
	ChatStatusActive    ChatStatus = "active"
	ChatStatusFinalized ChatStatus = "finalized"
	ChatStatusAbandoned ChatStatus = "abandoned"
)

// Chat represents a conversational session in the state store.
type Chat struct {
	ID        string
	Anchor    string // what the chat is about
	Workflow  string // workflow name
	Status    ChatStatus
	User      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChatCreateParams holds parameters for creating a new chat.
type ChatCreateParams struct {
	Anchor   string
	Workflow string
	User     string
}

// ChatFilter holds optional filters for listing chats.
type ChatFilter struct {
	Status *ChatStatus
}

// ChatMessage represents a single message in a chat history.
type ChatMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// ArtifactRef describes a named artifact without its content.
type ArtifactRef struct {
	Name string
	Size int64
}

// ProjectConfig holds the project-level configuration stored in the state store.
type ProjectConfig struct {
	Title     string
	SpecRepos []string
}

// StoreOptions holds configuration for constructing a Store.
type StoreOptions struct {
	SpecRepoPaths []string
	StateRepoPath string
}
