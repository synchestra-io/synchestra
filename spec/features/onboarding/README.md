# Feature: Onboarding

**Status:** Conceptual

## Summary

Onboarding is a guided wizard — delivered through both the Hub and the CLI — that walks new users through their first Synchestra project setup. It offers two paths: connecting real repositories with a GitHub App installation and AI-powered analysis, or launching a pre-built demo project to explore Synchestra without committing any infrastructure.

## Problem

Synchestra requires several coordinated setup steps before a user can begin: a spec repository, an optional set of code repositories, a state repository, a `synchestra-spec-repo.yaml` configuration, and (for real projects) a GitHub App installation for webhook delivery. Today these steps are manual and undocumented, creating a steep ramp for new users. Onboarding solves three problems:

1. **Discovery friction** — users don't know what Synchestra needs or in what order. The wizard sequences the steps and explains each one.
2. **Infrastructure bootstrapping** — creating a state repo, generating `synchestra-spec-repo.yaml`, and scaffolding the initial feature/spec structure requires multiple git and GitHub API operations that should be automated.
3. **Time-to-value** — users who aren't ready to connect their own repos (or have none) should still be able to experience Synchestra immediately through a demo project.

## Proposed Behavior

### Entry Point

- **Web app** — shown after first login when no projects exist; also accessible from project list as "New project."
- **CLI** — `synchestra init` or triggered automatically on first run when no project is configured.

### Path Selection

The wizard opens with a choice:

1. **Connect your repositories** — set up a real project backed by the user's GitHub repos.
2. **Try the demo** — launch a pre-built sample project to explore Synchestra.

---

### Path 1: Connect Your Repositories

A step-by-step flow that collects repository information, provisions infrastructure, and bootstraps the project.

#### Step 1 — Install the Synchestra GitHub App

The wizard checks whether the [Synchestra GitHub App](../github-app/README.md) is already installed for the user's account or organization.

- **Not installed** — the wizard embeds the GitHub App installation flow:
  - Web app: redirects to GitHub's app installation consent screen; GitHub redirects back with an `installation_id`.
  - CLI: opens the installation URL in the user's browser (or displays it for manual navigation); polls or listens for the callback.
- **Already installed** — the wizard skips to Step 2, displaying the installed orgs/repos.

After installation, Synchestra queries the installation's accessible repositories for use in subsequent steps.

#### Step 2 — Select Spec Repository

The user selects exactly **one** repository to hold project specifications:

- The wizard lists repositories accessible via the GitHub App installation.
- The user picks one. This becomes the spec repository referenced in `synchestra-spec-repo.yaml`.

#### Step 3 — Code Repositories (Optional)

The user may add zero or more code repositories that agents will work in:

- **"Code lives in same repo as specs"** — a checkbox (web) or yes/no prompt (CLI). If checked, the spec repository is also registered as a code repo.
- **Add more repos** — the user can select additional repositories from the accessible list.
- **Skip** — code repos are optional. Users developing specifications without implementation code can skip this step entirely.

#### Step 4 — State Repository

The state repository holds Synchestra's operational data (tasks, claims, status). Two options:

- **Create new** (suggested default) — Synchestra proposes `{project}-synchestra` as the repo name:
  - First attempts to create the repo in the **same organization** as the spec repo.
  - If the user lacks create-repository permissions in that org → falls back to creating in the **user's personal account**.
  - The wizard shows which org/account will be used and lets the user confirm.
- **Choose existing** — the user selects an existing repository from the accessible list.

#### Step 5 — Bring Your Own AI Key

An AI API key is required for the initial repository analysis and structure scaffolding.

- The user provides an API key and selects their AI model provider (e.g., OpenAI, Anthropic, Google).
- **Privacy commitment**: Synchestra does **not** store or log the key by default. It is used only for the current onboarding session and discarded afterward.
- **Opt-in storage**: The wizard offers an explicit opt-in: _"Store this key so Synchestra can use AI for ongoing operations (task analysis, conflict resolution, model routing)."_ If the user consents, the key is persisted securely and linked to their account.
- **Future**: A Synchestra-provided free tier key is planned but out of scope for the initial version. The wizard may display a note: _"Free AI credits coming soon."_

#### Step 6 — Initial Repository Analysis

Using the provided AI key, Synchestra analyzes the selected repositories:

- Scans repo structure, existing documentation, README files, and code organization.
- Proposes an initial set of Synchestra features and spec structure based on what it finds.
- Presents the proposed structure for user review before committing.
- Commits the scaffolded structure (feature directories, README templates) to the spec repo.

#### Step 7 — Project Creation

Synchestra generates and commits the project configuration:

- Creates `synchestra-spec-repo.yaml` in the spec repo with:
  - `title` — derived from the spec repo name (user can edit)
  - `state_repo` — URL of the state repo from Step 4
  - `repos` — list of code repo URLs from Step 3 (if any)
- Initializes the state repo with the base structure (`synchestra-spec-repo.yaml`, `README.md`, `tasks/`).
- Redirects to the project's home screen (web) or prints a success summary with next-step suggestions (CLI).

---

### Path 2: Try the Demo

For users who want to explore before committing:

- Synchestra provisions a **pre-built sample project** with:
  - Example features with specifications and proposals
  - Example tasks in various states (queued, in_progress, completed)
  - A sample development plan
  - Realistic but synthetic content so the user can navigate the full UI
- No GitHub App installation required.
- No AI key required.
- The demo project is fully functional within Synchestra but does not connect to real repositories.
- The user can transition from the demo to a real project at any time via "Connect your repositories."

---

### Platform Differences

Both surfaces implement the same logical flow but adapt to their medium:

| Aspect | Web App | CLI |
|---|---|---|
| GitHub App install | OAuth redirect + callback | Opens browser URL; waits for callback |
| Repo selection | Dropdown / searchable list | Interactive prompt with autocomplete |
| Checkboxes | Native checkbox UI | Yes/no prompt |
| AI key input | Password field (masked) | Secure terminal input (no echo) |
| Progress | Visual stepper / progress bar | Step counter with status messages |
| Repo analysis | Inline progress with preview | Streaming output with confirmation |
| Error handling | Inline validation, retry buttons | Error message + retry prompt |

### Completion Criteria

Onboarding is complete when:

- The user has at least one project (real or demo) visible in the project list.
- For real projects: `synchestra-spec-repo.yaml` exists in the spec repo, the state repo is initialized, and the GitHub App is installed on all referenced repos.
- For demo projects: the sample project is provisioned and navigable.

## Dependencies

- [GitHub App](../github-app/README.md) — installation flow is embedded in Step 1
- [Project Definition](https://github.com/synchestra-io/specscore/blob/main/spec/features/project-definition/README.md) — `synchestra-spec-repo.yaml` format and repository layout
- [UI](../ui/README.md) — Hub and TUI surfaces deliver the wizard
- [CLI](../cli/README.md) — `synchestra init` command entry point
- [API](../api/README.md) — backend endpoints for repo analysis, project creation, and GitHub App callback

## Outstanding Questions

- What AI providers and models should be supported at launch? Just OpenAI and Anthropic, or also Google, Mistral, etc.?
- How deep should the initial repo analysis go — top-level structure only, or should it read code to infer domain features?
- Should the demo project be stored in a real GitHub repo (e.g., a template repo the user forks) or exist only in Synchestra's local/server state?
- Should onboarding support importing an existing `synchestra-spec-repo.yaml` for users migrating from a manual setup?
- How should the CLI handle the GitHub App callback — poll a Synchestra endpoint, use a local HTTP server, or rely on `gh` CLI auth?
