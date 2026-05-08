# Proposal: Merge user-authentication with host-auth

**Status:** draft
**Target feature:** [user-authentication](../../README.md)
**Related feature:** [host-auth](../../../host-auth/README.md)
**Author:** @alex
**Created:** 2026-04-20

## Summary

`host-auth` and `user-authentication` describe authentication for the same resource (the host) from different callers: hub-to-host server-to-server (host-auth) and end-user-to-host (user-authentication). They share primitives — the host identity, the `user_ids` / `project_ids` ACL, the host registration lifecycle — but use different protocols (Ed25519 signing vs OIDC JWT) and flow in different directions. This proposal evaluates whether the two specs should be merged, and if so, how.

No decision is made here. This proposal surfaces the question and enumerates options so it can be decided after both features have some implementation experience.

## Motivation

Four things make the merger question worth asking:

1. **Shared primitives.** Both features reference the same host identity (`host_id` from registration), the same ACL field (`host.user_ids`), the same scoping field (`project_ids`). Duplicating the vocabulary across two specs risks drift.
2. **Shared lifecycle.** A host registered once should work in both directions. Registration is covered in host-auth; user-authentication currently reuses the same `host_id` and registration flow without explicitly describing the link.
3. **Shared keys and rotation.** Hub signs host-bound server-to-server requests (Ed25519); hub signs OIDC ID tokens for users (JWKS). These are separate key pairs today but rotation policy, key-identifier conventions, and infrastructure (e.g., Google Secret Manager) overlap. A unified spec would say this explicitly.
4. **One cross-cutting mental model.** Reviewers, new contributors, and AI agents encountering the codebase would benefit from a single "authentication" entry point rather than discovering two features by accident.

Four things argue against merging:

1. **Different audiences.** host-auth is read by infrastructure engineers setting up runners and Cloud Run deployments. user-authentication is read by application developers wiring OIDC into the hub UI or a CLI. Merging forces both audiences through a single doc.
2. **Different directions.** Bidirectional mutual auth vs. one-way user-to-host are structurally different. A merged doc has to signpost "this section applies to direction X only" throughout.
3. **Different protocols.** Ed25519 signature verification and OIDC token validation are different machinery with different failure modes. Keeping them apart lets each spec stay focused.
4. **Spec size.** Both features are already sizeable. A merged spec risks being unreadable.

## Options

### Option A: Create an `authentication/` parent feature, keep both as sub-features

```
synchestra/spec/features/authentication/
  README.md                     ← parent: explains the layers, shared primitives, cross-cutting decisions
  host-auth/
    README.md                   ← (moved from spec/features/host-auth/)
  user-authentication/
    README.md                   ← (moved from spec/features/user-authentication/)
  proposals/
    ...                         ← future cross-cutting proposals
```

**Pros.** Keeps each feature's audience served while giving shared primitives a single home. Room to add future auth features (service-to-service, API keys, agent-to-agent identity) without another reshuffle. SpecScore supports nesting cleanly.

**Cons.** Path-identification changes for both features (`host-auth` → `authentication/host-auth`). Every existing plan and proposal that references `synchestra/spec/features/host-auth/` needs updating — currently at minimum `synchestra-servers/spec/plans/host-auth/` and `synchestra-servers/spec/plans/device-flow/`. That's a non-trivial touch.

**When this wins.** When a third auth-related feature is on the near-term roadmap (e.g., service-to-service tokens for agent-to-agent), the parent starts paying for itself immediately.

### Option B: Merge into a single feature with directional sections

```
synchestra/spec/features/authentication/
  README.md                     ← single spec; "Host → Hub" and "Hub → Host" and "User → Host" subsections
```

**Pros.** One spec, one entry point. Shared primitives state once. Best when the content is small enough to scan.

**Cons.** Single largest spec in the repo. Different audiences forced through one doc. Sectioning has to be disciplined to prevent re-interleaving of concerns. Large content reviews become one monolith.

**When this wins.** When the content of both features is modest and overlapping-enough that separation feels artificial.

### Option C: Keep separate, strengthen cross-references

```
synchestra/spec/features/
  host-auth/          ← unchanged location
  user-authentication/← unchanged location
```

Both specs get a "Related" section cross-linking to the other and explicitly documenting the shared primitives (by reference to the `project` / `host` feature specs). No parent feature.

**Pros.** Zero migration. Lowest near-term cost. Two specs serve two audiences without compromise.

**Cons.** Shared primitives can drift. No natural home for cross-cutting proposals. Readers must discover the sibling spec on their own (or through the cross-ref).

**When this wins.** When no third auth feature is imminent, and the existing cross-references are judged sufficient.

## Recommendation (deferred)

No recommendation is made yet. The right answer depends on:

- **Whether a third auth-related feature lands soon.** If yes → lean Option A. If no → Option C is fine.
- **Experience with the current cross-references.** After fix-auth ships and user-authentication is reconciled with the canonical project / host ACL model, re-read both specs end-to-end and ask: does the cross-reference feel sufficient, or does the vocabulary drift? That empirical answer should decide.
- **Proposal-volume signal.** If cross-cutting proposals (proposals that touch both features) start appearing, that's evidence for Option A — a parent feature gives them a home. If every proposal stays cleanly inside one feature, Option C holds.

## Decision deadline

Before either feature graduates from `Draft` to `Stable`. A feature marked `Stable` with unresolved structural ambiguity about its own home invites confusion. Prefer to resolve this question at the `Stable` transition.

## Outstanding Questions

- If Option A is chosen, should the parent `authentication/` feature own cross-cutting concepts like "shared key identity" (the `host_id` used as an audience across both directions), or should that live in the individual sub-features?
- If Option C is chosen, where do cross-cutting proposals live? Under one feature by convention, with a cross-link from the other? Under an `ecosystem/`-style top-level proposals folder?
- Does introducing an `authentication/` parent feature imply other grouping features (e.g., `runtime/`, `data/`) should also exist? Or is auth special because its layering is especially visible?

---
*This document follows the https://specscore.md/feature-specification*
