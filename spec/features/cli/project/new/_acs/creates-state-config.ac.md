# AC: creates-state-config

**Status:** implemented
**Feature:** [cli/project/new](../README.md)

## Description

After `synchestra project new`, `synchestra-state-repo.yaml` exists in the state repo
with the spec_repos field pointing to the spec repo.

## Inputs

| Name | Required | Description |
|---|---|---|
| state_repo_path | Yes | Path to the state repository |

## Verification

```bash
test -f "$state_repo_path/synchestra-state-repo.yaml"
grep -q 'spec_repos:' "$state_repo_path/synchestra-state-repo.yaml"
```

## Scenarios

(None yet.)
