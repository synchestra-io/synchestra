# Feature: GitHub App

**Status:** Conceptual

## Summary

The Synchestra GitHub App is a GitHub-registered application that Synchestra installs on users' organizations and repositories to receive webhook notifications and perform authenticated operations. It is the bridge between GitHub-hosted repositories and Synchestra's coordination layer.

## Problem

Synchestra needs to react to events in users' repositories — new issues, pull requests, pushes, and comment activity — to keep task state synchronized and trigger automated workflows. Without a GitHub App, Synchestra must poll repositories or rely on users to manually push state, creating latency and friction. A GitHub App provides:

1. **Real-time event delivery** — GitHub sends webhooks as events occur, eliminating polling.
2. **Fine-grained permissions** — the app requests only the access it needs, scoped to selected repositories.
3. **Authenticated API access** — installation tokens let Synchestra read repository content, create branches, and open PRs on the user's behalf without storing personal access tokens.
4. **Organization-level management** — org admins install once; all selected repos are covered.

## Proposed Behavior

### App Identity

The Synchestra GitHub App is registered under the `synchestra-io` GitHub organization. It has a public listing page where users can review permissions before installing.

### Permissions

The app requests the minimum permissions required for Synchestra's coordination features:

| Permission | Access | Purpose |
|---|---|---|
| Repository contents | Read & Write | Read specs, features, tasks; push state changes to state repos |
| Issues | Read & Write | Sync issues with Synchestra tasks; create issues from proposals |
| Pull requests | Read & Write | Track PR status; create PRs from development plans |
| Webhooks | Read-only | Receive event notifications |
| Metadata | Read-only | Repository metadata for project discovery |

### Webhook Events

The app subscribes to the following events:

| Event | Purpose |
|---|---|
| `issues` | New issues, status changes — sync with task queue |
| `pull_request` | PR opened, merged, closed — update task progress |
| `push` | Commits to tracked branches — detect state repo updates |
| `installation` | App installed, uninstalled, or permissions changed |
| `installation_repositories` | Repositories added to or removed from installation |

### Installation Flow

Installation follows GitHub's standard app installation flow, embedded into the Synchestra [onboarding](../onboarding/README.md) wizard:

1. **Initiate** — user clicks "Install GitHub App" in the onboarding wizard (Hub) or follows a URL (CLI).
2. **GitHub consent screen** — GitHub displays the app's requested permissions and lets the user choose:
   - **Organization-level install** — select which org and which repositories (all or a subset).
   - **Personal account install** — for repos owned by the user directly.
3. **Callback** — GitHub redirects to Synchestra's callback URL with an `installation_id`.
4. **Registration** — Synchestra stores the installation record, associating the `installation_id` with the user's Synchestra account.
5. **Repository discovery** — Synchestra queries the installation's accessible repositories and presents them in the onboarding flow for project setup.

### Installation Token Lifecycle

After installation, Synchestra uses GitHub's installation token API to obtain short-lived tokens for authenticated operations:

- Tokens are requested on demand, scoped to the repositories needed for the current operation.
- Tokens expire after 1 hour (GitHub default); Synchestra refreshes as needed.
- No long-lived credentials are stored — only the `installation_id` and app private key.

### Consuming Installation Status

Other features query the GitHub App's installation status:

- **Onboarding** — checks whether the app is installed and which repos are accessible before proceeding to repo selection.
- **State sync** — uses installation tokens to push/pull state repo changes.
- **Issue sync** (future) — listens to `issues` webhooks to create/update Synchestra tasks.
- **PR tracking** (future) — listens to `pull_request` webhooks to update task progress.

### Uninstallation

When a user uninstalls the GitHub App from their org or removes repositories:

1. GitHub sends an `installation` or `installation_repositories` webhook.
2. Synchestra marks affected projects as disconnected — tasks and specs remain but real-time sync stops.
3. Users can reinstall at any time to resume sync.

## Dependencies

- [API](../api/README.md) — callback endpoint for GitHub's OAuth redirect
- [Project Definition](https://github.com/synchestra-io/specscore/blob/main/spec/features/project-definition/README.md) — installation maps to project repo configuration

## Outstanding Questions

- Should the app be listed on the GitHub Marketplace, or installed only via direct URL during onboarding?
- What additional permissions will be needed for future features (e.g., GitHub Actions integration, commit status checks)?
- How should Synchestra handle partial installations where the user grants access to only some repos in an org — should it warn that some project repos are not covered?
- Should the app support GitHub Enterprise Server installations, or only github.com for the initial version?
