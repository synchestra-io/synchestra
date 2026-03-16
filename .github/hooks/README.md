# Git Hooks

Custom git hooks for the Synchestra repository. Activate with:

```bash
git config core.hooksPath .github/hooks
```

## Contents

| Hook | Description |
|------|-------------|
| [pre-commit](pre-commit) | Blocks `.github/README.md`, enforces README.md in every directory, runs Go validation |

## Outstanding Questions

- Should a setup script or Makefile target automatically set `core.hooksPath`?
- Should the pre-commit hook also validate that README.md files contain an "Outstanding Questions" section?
