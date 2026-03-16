# pkg/state/gitstore

Git-backed implementation of `state.Store`. Maps every interface method to file operations, markdown table rendering, and atomic commit-and-push in a Synchestra state repository.

This is the default state store backend — it requires no external infrastructure beyond a git remote.

## Status

Stub implementation. All methods return `errNotImplemented`. The full implementation is tracked separately.

## Spec

See `spec/features/state-store/backends/git/` for the method-to-git-operation mapping.

## Outstanding Questions

None at this time.
