# ADR-0001: Extract AI plugin to a dedicated repository

**Status:** Approved
**Date:** 2026-04-18

## Context

The Synchestra AI plugin (`synchestra/ai-plugin/`) lived inside the `synchestra` CLI repository, packaged as a Claude Code plugin with 24 skills wrapping `synchestra` CLI commands. A peer plugin, [`ai-plugin-sdd`](https://github.com/synchestra-io/ai-plugin-sdd) (spec-driven development methodology), already lived in a dedicated repository following a `synchestra-io/ai-plugin-{topic}` naming pattern.

The co-location produced three observable issues:

1. **Discoverability.** Skills were buried three levels deep inside a CLI-focused repository (`synchestra/ai-plugin/skills/`). They were easy to forget — the project owner reported actively forgetting they existed when thinking about the ecosystem.
2. **Inconsistency.** One plugin was decoupled, the other was nested. No principle explained the split; it was historical.
3. **Cognitive framing.** Contributors had to parse "skills" and "CLI" as one thing, even though they are separable concerns with different release cadences and audiences.

The previous plan, recorded in `ai-plugin/skills/README.md`, was to extract the plugin *post-beta*, once the CLI contract stabilized. Two assumptions shifted that plan:

- **CLI stability is not a prerequisite.** The CLI evolves additively (new commands; breaking changes rare), and a self-update mechanism is planned. Cross-repo coordination cost is low.
- **The coupling cost is present, not future.** Discoverability pain is already affecting the primary contributor. The cost of waiting for "beta" is paid every day.

## Decision

Extract `synchestra/ai-plugin/` into a dedicated repository, **`synchestra-io/ai-plugin-synchestra`**, matching the pattern set by `ai-plugin-sdd`.

The repository name uses `ai-plugin-*` rather than `synchestra-skills` because the plugin will grow beyond skills: slash commands (e.g., `/synchestra:dispatch`) and event hooks are planned. The name describes the container, not one of its current components.

Phase 1 of the extraction (creating the new repository with the plugin contents) is complete at the time this ADR is accepted. Phase 2 — removing `synchestra/ai-plugin/` from the `synchestra` repository and redirecting all references — is a follow-up task.

## Consequences

**Easier**

- Skills become a first-class, discoverable product with their own README, homepage, and release stream.
- The `synchestra-io/ai-plugin-*` naming pattern becomes a recognizable ecosystem convention. Anyone browsing the organization's repositories can spot the plugins at a glance.
- Plugin distribution can be branded separately (planned: `install.synchestra.io/skills` as a redirect in front of GitHub releases).
- External contributors can propose new skills without touching the CLI repository.
- Slash commands, subagents, and hooks can be added to the plugin without polluting the CLI source tree.

**Harder**

- Cross-repo pull requests when a new CLI command needs a corresponding skill (expected cost, bounded).
- Two release pipelines instead of one.
- Documentation must explain that the CLI and the plugin are separate installs.

**Mitigations**

- The CLI's additive evolution minimizes the frequency of coordinated changes.
- `synchestra skill install` (already specified) hides the source repository from end users; they will not need to know the plugin lives elsewhere.
- The `ai-plugin-sdd` repository already proves the pattern is viable.

## Alternatives considered

1. **Keep co-located indefinitely.** Rejected — the discoverability cost is concrete and already causing the owner to forget the skills exist. Tight coupling is not earning its keep.
2. **Keep co-located until beta, then extract.** Original plan. Rejected — the "beta" trigger is vague and always-in-the-future. The cost of extracting now is similar to extracting later, while the pain of waiting is incurred every day until then.
3. **Name the new repository `synchestra-skills`.** Rejected — clearer as a product name, but breaks the `ai-plugin-{topic}` pattern and undersells the future-expansion surface (slash commands, hooks, subagents). The owner explicitly confirmed plans for all three categories, which made the generic plugin naming the right choice.

## Follow-ups

- **Phase 2:** Remove `synchestra/ai-plugin/` from the `synchestra` repository after the new repository is verified working. Update references in `AGENTS.md`, feature specs that point at `ai-plugin/skills/`, and the Copilot CLI symlink (`.github/skills`).
- **Distribution branding:** Set up `install.synchestra.io/skills` as a redirect to the latest GitHub release of `ai-plugin-synchestra`.
- **Skill naming / prefix convention:** Tracked as sub-decision C in the remote-dispatch ideation thread; resolve separately.

---
*This document follows the https://specscore.md/decision-specification*
