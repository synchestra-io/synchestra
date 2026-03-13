# Synchestra — Project Conventions

## Directory structure

- Every directory MUST have a `README.md` file, **except `.github/`** (where a `README.md` would override the root one on GitHub's repository page; see `.github/README.not.md`).
- Every `README.md` MUST have an "Outstanding Questions" section. If there are none, it explicitly states "None at this time." — never omit the section.
- Every `README.md` that has child directories MUST include a brief summary (1–7 sentences) for each immediate child after the index table. This gives readers high-level context without requiring them to open each child.
- CLI arguments are documented in `_args/` directories under `spec/features/cli/`. Each argument has its own `.md` file at the level where it applies (global, command-group, or command-specific). See the [`_args` convention in the CLI spec](spec/features/cli/README.md#the-args-directory-convention) for the full format and placement rules.
