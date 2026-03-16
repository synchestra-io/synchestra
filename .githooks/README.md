# Git Hooks

Custom git hooks for the Synchestra repository. Activate with:

```bash
git config core.hooksPath .githooks
```

## Contents

| Hook | Description |
|------|-------------|
| [pre-commit](pre-commit) | Enforces AGENTS.md rules: every new directory must include a README.md |

## Outstanding Questions

- Should we add a post-clone or setup script that automatically sets `core.hooksPath`?
- Should the pre-commit hook also validate that README.md files contain an "Outstanding Questions" section?
