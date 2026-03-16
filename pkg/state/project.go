package state

// Features implemented: state-store, state-store/project-store

import "context"

// ProjectStore defines operations on project-level state —
// configuration back-references and README generation.
type ProjectStore interface {
	// Config returns the project configuration from the state store.
	Config(ctx context.Context) (ProjectConfig, error)

	// UpdateConfig writes updated project configuration.
	UpdateConfig(ctx context.Context, config ProjectConfig) error

	// RebuildREADME regenerates the auto-generated project overview.
	RebuildREADME(ctx context.Context) error
}
