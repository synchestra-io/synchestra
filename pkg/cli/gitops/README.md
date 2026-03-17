# gitops

Thin wrapper around git CLI operations used by commands that mutate repositories. Provides clone, commit-and-push (with retry on push conflict), pull, repo validation, and origin URL retrieval.

All functions shell out to the `git` binary via `os/exec`.

## Outstanding Questions

None at this time.
