package state

// Features implemented: state-store, state-store/chat-store

import "context"

// ChatStore defines operations for chat lifecycle management,
// append-only message history, and chat-scoped artifacts.
type ChatStore interface {
	// Create creates a new chat in created status. The store generates the chat ID.
	Create(ctx context.Context, params ChatCreateParams) (Chat, error)

	// Get returns a chat by its ID. Returns ErrNotFound if the chat does not exist.
	Get(ctx context.Context, chatID string) (Chat, error)

	// List returns chats matching the given filter.
	List(ctx context.Context, filter ChatFilter) ([]Chat, error)

	// Finalize transitions a chat to finalized, flushing message history
	// to durable storage.
	Finalize(ctx context.Context, chatID string) error

	// Abandon transitions a chat to abandoned.
	Abandon(ctx context.Context, chatID string) error

	// AppendMessages appends messages to the chat history.
	// On the first call for a created chat, this implicitly transitions
	// the chat status to active.
	AppendMessages(ctx context.Context, chatID string, messages []ChatMessage) error

	// Messages returns the full message history for a chat.
	Messages(ctx context.Context, chatID string) ([]ChatMessage, error)

	// Artifact returns the artifact sub-interface scoped to the given chat.
	Artifact(ctx context.Context, chatID string) ArtifactStore
}
