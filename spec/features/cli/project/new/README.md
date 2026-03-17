# Command: `synchestra project new`

**Parent:** [project](../README.md)

## Synopsis

```
synchestra project new --spec-repo <ref> --state-repo <ref> --code-repo <ref> [--code-repo <ref>...] [--title <title>]
```

## Description

Creates a new Synchestra project by linking a spec repo, state repo, and one or more code repos. The command resolves all repo references, clones any that are not already on disk, validates they are git repos, writes the appropriate config files to each, and commits and pushes the changes.

Config files written:

| Repo | File | Content |
|---|---|---|
| Spec | `synchestra-spec.yaml` | Full project definition: `title`, `state_repo`, `repos` |
| State | `synchestra-state.yaml` | Back-reference: `spec_repos` |
| Code(s) | `synchestra-code.yaml` | Pointer: `spec_repos` |

If a repo already contains a config file for a different project, the command fails with exit code `1`.

## Parameters

| Parameter | Required | Description |
|---|---|---|
| [`--spec-repo`](../_args/spec-repo.md) | Yes | Spec repository reference |
| [`--state-repo`](../_args/state-repo.md) | Yes | State repository reference |
| [`--code-repo`](../_args/code-repo.md) | Yes (at least one) | Code repository reference (repeatable) |
| [`--title`](_args/title.md) | No | Project title |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Project created successfully |
| `1` | Conflict — a repo already has a config file for a different project |
| `2` | Invalid arguments (missing required flags, invalid repo reference format) |
| `3` | Repo not found — clone failed or resolved directory does not exist |
| `10+` | Unexpected error |

## Behaviour

1. Read `~/.synchestra.yaml` for `repos_dir` (default: `~/synchestra/repos/`)
2. Resolve each repo reference to `{repos_dir}/{hosting}/{org}/{repo}`
3. Clone any repos not already on disk; exit `3` on clone failure
4. Validate all resolved directories are git repos
5. Check that no repo already has a config file pointing to a different project; exit `1` if so
6. Derive project title: `--title` flag > first `# heading` in spec repo `README.md` > spec repo identifier
7. Write `synchestra-spec.yaml` to spec repo with `title`, `state_repo` (origin URL), and `repos` (list of origin URLs)
8. Write `synchestra-state.yaml` to state repo with `spec_repos` (list of origin URLs)
9. Write `synchestra-code.yaml` to each code repo with `spec_repos` (list of origin URLs)
10. Commit and push changes to all affected repos
11. On push conflict: pull, re-check, retry or fail

## Acceptance Criteria

| AC | Description | Status |
|---|---|---|
| [creates-spec-config](_acs/creates-spec-config.md) | synchestra-spec.yaml created in spec repo | implemented |
| [creates-state-config](_acs/creates-state-config.md) | synchestra-state.yaml created in state repo | implemented |

## Outstanding Questions

- Should the command validate that the spec repo does not already have a `synchestra-spec.yaml`, or should it allow overwriting/updating?
- Should there be a `--no-clone` flag to skip cloning and only operate on repos already on disk?
