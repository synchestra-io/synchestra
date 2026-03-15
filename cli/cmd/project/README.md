# `cli/cmd/project` — Project Command Group

This package implements the `synchestra project` command group and its subcommands.

## Index

| File | Description |
|------|-------------|
| `project.go` | Defines `GroupCommand`, which returns the `project` cobra command group. |
| `new.go` | Implements `NewCommand` and the `project new` subcommand logic. |
| `new_test.go` | Tests for `project new`, covering config file writing, title derivation, conflict detection, multiple targets, and push-conflict retry. |

## Commands

### `project new`

Creates a new Synchestra project by linking a spec repo, a state repo, and one or more target repos.

**Flags:**

| Flag | Required | Description |
|------|----------|-------------|
| `--spec-repo` | Yes | Repository reference for the spec repo (e.g. `github.com/org/repo`). |
| `--state-repo` | Yes | Repository reference for the state repo. |
| `--target-repo` | At least one | Repository reference for a target repo. Repeatable. |
| `--title` | No | Project title. Derived from `README.md` heading or repo name if omitted. |

**Behaviour:**

1. Parses all repo references via `reporef.Parse`.
2. Clones any repos not already present on disk under `ReposDir` (from global config).
3. Checks for conflicts: if a config file already points to a different spec repo, returns exit code 1.
4. Derives title from `--title` flag, then `# Heading` in `README.md`, then spec repo name.
5. Writes `synchestra-spec.yaml` to the spec repo, `synchestra-state.yaml` to the state repo, and `synchestra-target.yaml` to each target repo.
6. Commits and pushes each repo. On push conflict, pulls and retries the push (without re-committing).

## Outstanding Questions

None at this time.
