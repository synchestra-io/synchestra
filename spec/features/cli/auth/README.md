# Feature: CLI Auth

**Status:** In Progress

## Summary

Command group under the `synchestra` CLI for authenticating the calling user against the Synchestra Hub. Authentication gates any command that reads from or writes to the Hub — at present, [CLI Runner](../runner/README.md) and [CLI Session](../session/README.md). Commands that only operate on the local state repository (tasks, spec lint, feature queries) do not require authentication.

This feature is distinct from [host-auth](../../host-auth/README.md): host-auth is machine-to-Hub (a runner registering itself); CLI Auth is user-to-Hub (a developer authenticating their CLI).

## Problem

Users and agents running the CLI need a way to prove their identity to the Hub before invoking runner or session commands. Without a dedicated auth surface, every Hub-touching command would need to carry credentials inline — breaking the CLI's flag discipline and making scripted use fragile.

Credentials also need a clear lifecycle: acquiring them, rotating or revoking them, and querying the current identity for debugging. A single entry point keeps that lifecycle visible.

## Behavior

### Command group structure

```
synchestra auth <verb>
```

Credentials are stored transparently in the standard config location (`~/.synchestra.yaml` or OS keychain where available — details deferred to implementation). Hub-touching subcommands in other groups read credentials from this store; they do not accept auth flags of their own.

### Authentication model

User authentication is a separate identity from host authentication. A logged-in user can enumerate the runners they own, dispatch to any of them, and observe sessions they created — regardless of how those runners registered themselves (which is governed by [host-auth](../../host-auth/README.md)).

The MVP flow is the device authorization flow (browser-based code approval), matching the pattern already used by [`synchestra-host hub connect`](../../host-auth/README.md#path-2-interactive-device-flow). Non-interactive login (`--token`) is deferred pending a CI use case.

## Contents

Each verb is a subfeature. Detailed specs are planned; this README documents the surface.

| Verb | Description |
|---|---|
| `login` | Authenticate the CLI against the Synchestra Hub (device flow) |
| `logout` | Clear stored credentials |
| `whoami` | Print the authenticated identity |

### login

```
synchestra auth login [--hub <url>]
```

Initiates a device authorization flow. The CLI prints a verification URL and code; the user approves in their browser. On success, the token is persisted and the authenticated identity is printed. `--hub` targets a non-default Hub URL (useful against a local Hub during development) and is remembered for the lifetime of the stored credentials.

### logout

```
synchestra auth logout
```

Clears stored credentials for the currently active Hub. Idempotent — exits 0 whether credentials were present or not.

### whoami

```
synchestra auth whoami [--format text|json]
```

Prints the authenticated user's identity (email, user ID, Hub URL) or exits with code `101` if unauthenticated.

### Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success |
| `2` | Invalid arguments (e.g., malformed `--hub` URL) |
| `100` | Login flow failed (denied, timed out, or user cancelled) |
| `101` | Not authenticated |
| `102` | Hub unreachable |

Exit codes `100`–`109` are reserved for the CLI Auth feature. This range is registered in the [CLI](../README.md#command-group-reserved-ranges) parent feature. Other CLI groups return `101` to signal an unauthenticated call.

## Dependencies

- [cli](../README.md) — parent feature; defines command conventions and exit code contract
- [host-auth](../../host-auth/README.md) — parallel feature for machine-to-Hub auth; shares no credentials but shares the device-flow UX pattern
- [cli/runner](../runner/README.md), [cli/session](../session/README.md) — the consumers of authentication

## Acceptance Criteria

1. `auth login` with network access succeeds via device flow without requiring any pre-existing credentials on disk.
2. After a successful `auth login`, subsequent calls to runner and session subcommands proceed without re-authenticating.
3. `auth logout` clears stored credentials; `auth whoami` immediately afterwards exits 101.
4. `auth whoami` without stored credentials exits 101 and prints nothing on stdout (diagnostic text on stderr only).
5. `auth login --hub <url>` persists the hub URL so that follow-up `whoami` and other commands target the same Hub.
6. A non-default `--hub` URL that is unreachable fails with exit code 102 and does not persist a partial credential record.

## Outstanding Questions

1. Where exactly are credentials stored — a YAML file under `~/.synchestra/`, the OS keychain, or both with keychain preferred? Defer to implementation, but the feature contract must specify visibility semantics (e.g., multi-user workstation expectations).
2. Should `auth login` support `--token <value>` for non-interactive flows (CI, headless servers)? Out of MVP scope, but the subfeature spec should explicitly state deferral so it is easy to add later.
3. Token refresh strategy — silent refresh before expiry vs. fail-on-expiry-and-prompt — and how it surfaces to the CLI user.
4. Does `auth` need a `token` subfeature (e.g., `auth token list`, `auth token revoke`) for users who have multiple active sessions across machines?
