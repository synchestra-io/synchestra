# ADR-0004: Layered plugin architecture — CLI wrappers and methodology plugins

**Status:** Accepted
**Date:** 2026-04-19

## Context

The Synchestra.io ecosystem publishes Claude Code plugins for two distinct kinds of capability:

1. **CLI wrappers** — plugins whose skills map 1:1 onto commands of a specific binary. [`ai-plugin-synchestra`](https://github.com/synchestra-io/ai-plugin-synchestra) (manifest name `synchestra-cli`) wraps the `synchestra` binary. The [`synchestra-io/specscore`](https://github.com/synchestra-io/specscore) repository also ships a CLI binary (`specscore`), but no plugin wraps it yet.
2. **Methodology plugins** — plugins whose skills encode *process*, not command surface. [`ai-plugin-sdd`](https://github.com/synchestra-io/ai-plugin-sdd) (manifest name `spec-driven-development`) encodes the spec-driven development workflow: how to turn an idea into a SpecScore feature, when to lint, when to enqueue a task. It references SpecScore and Synchestra concepts but does not itself wrap either CLI's commands 1:1.

Today, `ai-plugin-sdd` conflates these two kinds. Its `specscore:design` and `specscore:ideate` skills are methodology skills, but their names imply a CLI wrapping that does not exist. Users installing `spec-driven-development` have no way to also get CLI-wrapping skills for `specscore` or `synchestra` — they have to install each plugin separately and know that they belong together.

Claude Code shipped native plugin dependency support in v2.1.110 ([docs](https://code.claude.com/docs/en/plugin-dependencies)), which removes the coordination burden:

> "When you install a plugin that declares dependencies, Claude Code resolves and installs them automatically."

Cross-marketplace dependencies are blocked unless allowlisted, so co-locating dependent plugins in a single marketplace is the simplest path. The [`synchestra-io/ai-marketplace`](https://github.com/synchestra-io/ai-marketplace) repository already exists and lists `synchestra-cli` and `spec-driven-development`.

## Decision

The `synchestra-io` ecosystem adopts a **two-layer plugin architecture**:

- **Base layer — CLI wrapper plugins.** One plugin per CLI binary. Each wraps its binary's commands using the resource + references structure defined in [ADR-0002](0002-progressive-disclosure-skills.md) and follows the naming rules in [ADR-0003](0003-skill-naming-plugin-namespace.md). These plugins carry no methodology — each skill is a thin, mechanical translation of a CLI invocation.
- **Methodology layer — process plugins.** Plugins whose skills encode multi-step workflows across one or more CLI binaries. They depend on the relevant CLI-wrapper plugins via `plugin.json` `dependencies`. Methodology skills call into CLI skills as building blocks.

Initial population of each layer:

| Layer | Plugin | Repository | Wraps / Depends on |
|---|---|---|---|
| Base | `synchestra-cli` | `ai-plugin-synchestra` (existing) | `synchestra` binary |
| Base | `specscore-cli` | `ai-plugin-specscore` (new) | `specscore` binary |
| Methodology | `spec-driven-development` | `ai-plugin-sdd` (existing, restructured) | depends on `synchestra-cli` and `specscore-cli` |

All three plugins live in the single `synchestra-io/ai-marketplace` marketplace. This keeps dependency resolution in-marketplace (no allowlisting overhead) and gives users one place to discover the ecosystem.

Dependencies declared in `spec-driven-development`'s `plugin.json`:

```json
{
  "name": "spec-driven-development",
  "version": "1.0.0",
  "dependencies": [
    { "name": "synchestra-cli", "version": "^0.1.0" },
    { "name": "specscore-cli",  "version": "^0.1.0" }
  ]
}
```

Installing `/plugin install spec-driven-development@synchestra-io` auto-installs both CLI wrappers. Installing either CLI wrapper alone remains valid — methodology is optional, but the CLI wrappers are self-sufficient.

## Consequences

**Easier**

- One install gets a coherent toolkit. A user who wants spec-driven development runs one command and receives the methodology plus both CLIs.
- Clean separation of concerns. CLI wrappers evolve with their binaries; the methodology plugin evolves with the process. Neither holds the other back.
- Version constraints surface breakage. If `synchestra-cli` ships a breaking change, `spec-driven-development`'s semver range either permits the upgrade or pins users to the last compatible version with a clear error message.
- Discoverability. All three plugins listed in the same marketplace, named consistently (`<tool>-cli` for wrappers, `<methodology>` for process).
- Methodology skills can assume CLI skills are present — no defensive checks, no conditional instructions for users who "might not have it installed."

**Harder**

- Three repositories to maintain instead of two. Each CLI-wrapper plugin requires its own release cadence, README, and roadmap.
- The `ai-plugin-specscore` repository does not yet exist and must be created.
- Existing `ai-plugin-sdd` content must be audited and restructured: skills that wrap specscore CLI commands move to `ai-plugin-specscore`; methodology skills stay.
- Users on Claude Code older than v2.1.110 do not get automatic dependency installation. They must install each plugin manually.
- Release tagging discipline is non-trivial: the marketplace must carry `{plugin-name}--v{version}` tags for dependency resolution to work (`specscore-cli--v0.1.0`, `synchestra-cli--v0.1.0`). Simple version tags (`v0.1.0`) are insufficient.

**Mitigations**

- The `ai-plugin-*` repository pattern (established in [ADR-0001](0001-extract-ai-plugin.md)) already treats CLI-wrapper plugins as a repeatable shape. Creating `ai-plugin-specscore` reuses the template.
- The marketplace already scaffolds multi-plugin listings. Adding one entry is trivial.
- Minimum Claude Code version (v2.1.110) is documented in each plugin's README.
- A CI check in `ai-marketplace` can validate that every plugin has tags in the `{name}--v*` convention before a release is considered valid.

## Alternatives considered

1. **Keep everything in one plugin** (the `spec-driven-development` plugin grows to include CLI-wrapping skills). Rejected — violates separation of concerns. A user who wants the CLI without the methodology is forced to install the methodology. Future CLI changes force methodology releases.
2. **Publish each plugin in a separate marketplace.** Rejected — cross-marketplace dependencies require allowlisting in the root marketplace's `marketplace.json`, adding coordination overhead for zero gain. Co-located plugins stay in-marketplace and resolve trivially.
3. **Skip the dependency declaration; let users install plugins manually.** Rejected — the whole point of the meta-methodology plugin is to bundle a coherent experience. Without auto-install, users have to read installation instructions in three places and get the version matrix right by hand. The native dependency mechanism exists specifically for this.
4. **Wait until `specscore` CLI is more mature before creating `specscore-cli`.** Rejected — the CLI binary already exists in `synchestra-io/specscore`, and creating the wrapper plugin now avoids a second audit of `ai-plugin-sdd` later. Early version of `specscore-cli` can be 0.0.x, marking its pre-stable surface.

## Follow-ups

- **Create `synchestra-io/ai-plugin-specscore` repository.** Follow the `ai-plugin-*` pattern: MIT license, `.claude-plugin/plugin.json` with `name: specscore-cli`, `skills/` at plugin root, same resource + references structure as `ai-plugin-synchestra`. Populate with CLI-wrapper skills for each `specscore` command.
- **Audit `ai-plugin-sdd` content.** Identify any skills that wrap specscore CLI commands directly; move them to `ai-plugin-specscore`. Keep methodology skills (`design`, `ideate`, future: `brainstorm`, `ship`, etc.) in the sdd plugin. Rename them per [ADR-0003](0003-skill-naming-plugin-namespace.md) — no colons in frontmatter `name`, no plugin-name prefix on directories.
- **Add `dependencies` to `spec-driven-development`'s `plugin.json`.** Declare both CLI wrappers. Pin to the first stable minor version (e.g., `^0.1.0`) once each CLI wrapper has one.
- **Update `ai-marketplace/.claude-plugin/marketplace.json`** to list all three plugins.
- **Adopt the `{plugin-name}--v{version}` tagging convention** for all three plugins. Document it in each repo's release checklist.
- **Document minimum Claude Code version** (v2.1.110) in each plugin's README.
- **Separately: decide whether the methodology plugin name `spec-driven-development` should be shortened** (e.g., to `sdd`) for more ergonomic slash-menu invocation. Out of scope for this ADR; record as a question in `ai-plugin-sdd`'s own backlog.
