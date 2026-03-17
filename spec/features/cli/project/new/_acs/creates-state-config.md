# AC: creates-state-config

**Status:** implemented
**Feature:** [cli/project/new](../README.md)

## Description

After `synchestra project new`, `synchestra-state.yaml` exists in the state repo
with the spec_repos field pointing to the spec repo.

## Inputs

| Name | Required | Description |
|---|---|---|
| state_repo_path | Yes | Path to the state repository |

## Verification

```bash
test -f "$state_repo_path/synchestra-state.yaml"
grep -q 'spec_repos:' "$state_repo_path/synchestra-state.yaml"
```

## Scenarios

(None yet.)
