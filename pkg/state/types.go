package state

// Features implemented: state-store

import (
	"time"

	"github.com/synchestra-io/specscore/pkg/task"
)

// Type aliases for backward compatibility — callers can use either
// state.TaskStatus or task.TaskStatus interchangeably.
type TaskStatus = task.TaskStatus

// Re-export status constants so existing code using state.TaskStatusXxx
// continues to compile without modification.
const (
	TaskStatusPlanning   = task.StatusPlanning
	TaskStatusQueued     = task.StatusQueued
	TaskStatusClaimed    = task.StatusClaimed
	TaskStatusInProgress = task.StatusInProgress
	TaskStatusCompleted  = task.StatusCompleted
	TaskStatusFailed     = task.StatusFailed
	TaskStatusBlocked    = task.StatusBlocked
	TaskStatusAborted    = task.StatusAborted
)

// Type aliases for backward compatibility.
type TaskCreateParams = task.CreateParams
type TaskFilter = task.Filter
type BoardView = task.BoardView
type BoardRow = task.BoardRow

// CoordinatedTask embeds specscore's Task with coordination-only fields
// that belong in the orchestration layer (synchestra), not the library.
type CoordinatedTask struct {
	task.Task
	Run       string     // agent run ID (populated when claimed/in_progress)
	Model     string     // agent model ID (populated when claimed/in_progress)
	ClaimedAt *time.Time // when the task was claimed
}

// ClaimParams holds parameters for claiming a task.
type ClaimParams struct {
	Run   string
	Model string
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
	Sync          SyncConfig
}

// SyncPolicy controls when the store automatically syncs with the remote.
type SyncPolicy string

const (
	// SyncOnCommit syncs after every merge to local main. Default.
	SyncOnCommit SyncPolicy = "on_commit"

	// SyncOnInterval syncs on a timer.
	SyncOnInterval SyncPolicy = "on_interval"

	// SyncOnSessionEnd syncs when the agent session ends.
	SyncOnSessionEnd SyncPolicy = "on_session_end"

	// SyncManual syncs only via explicit Pull/Push/Sync calls.
	SyncManual SyncPolicy = "manual"
)

// SyncConfig holds the sync policy for automatic pull/push behaviour.
// Pull and push policies are independent. Both default to SyncOnCommit.
type SyncConfig struct {
	Pull         SyncPolicy
	PullInterval time.Duration // used when Pull is SyncOnInterval
	Push         SyncPolicy
	PushInterval time.Duration // used when Push is SyncOnInterval
}
