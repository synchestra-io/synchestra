# Spec Repository

The spec repository is the **source of truth for what should be built**. It contains:

- **Feature specifications** — what the system should do, acceptance criteria, design decisions
- **Architecture documents** — how the system is structured, trade-offs, constraints
- **Product documentation** — user-facing explanations, API guides, tutorials
- **Project configuration** — `synchestra-spec-repo.yaml`, which defines the project and references the state and code repositories

The spec repo is where humans and agents collaborate on *decisions*. Changes are deliberate and typically reviewed. The directory structure mirrors the product's feature tree — agents navigate `spec/features/` to understand requirements before starting work.

## Why It's Separate

Specifications have a fundamentally different lifecycle from both code and coordination state. They change when the product direction changes, not when an agent claims a task or a build completes. Keeping specs in their own repo (or combined with code for smaller projects) ensures that the product definition isn't buried under machine-generated commits.

## Naming Convention

User's choice. Common patterns: `{project}`, `{project}-spec`, or combined with code in a single repo.

## Example Structure

```
acme/
  synchestra-spec-repo.yaml         # Project config → references state repo + code repos
  README.md
  spec/
    features/
      user-auth/
        README.md
      payment-flow/
        README.md
    architecture/
      ...
  docs/
    ...
```

## Rules

The following rules are mandatory for every spec repository.

1. **README.md per directory** — Every directory MUST have a `README.md`, **except `.github/` itself** (where a `README.md` would override the root one on GitHub's repository page). Subdirectories under `.github/` (e.g., `.github/workflows/`) MUST still have a `README.md`.

2. **Outstanding Questions section** — Every `README.md` MUST have an "Outstanding Questions" section. If there are none, explicitly state "None at this time." — never omit the section.

3. **Child directory summaries** — Every `README.md` that has child directories MUST include a brief summary (1–7 sentences) for each immediate child after an index table.

4. **CLI command directories** — Every CLI command or subcommand defined under `spec/features/cli/` MUST have its own dedicated directory with a `README.md`. Commands are never documented only as subsections of a parent — they get their own feature directory.

5. **Config file** — The spec repo root MUST contain `synchestra-spec-repo.yaml` defining the project and referencing the state and code repositories.

6. **Mermaid diagrams** — Use mermaid diagrams instead of ASCII art in all specification documents.

7. **Feature references in Go files** — Every `.go` file must include structured comments referencing features it implements and depends on (relevant when the spec repo is combined with code). Example:
   ```go
   // Features implemented: cli/task/claim, cli/task/update
   // Features depended on:  state-sync/pull, project-definition/state-repo
   ```

## Outstanding Questions

None at this time.
