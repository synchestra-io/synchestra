# AC: creates-spec-config

**Status:** implemented
**Feature:** [cli/project/new](../README.md)

## Description

After `synchestra project new`, `synchestra-spec-repo.yaml` exists in the spec repo
with the correct title and state_repo fields.

## Inputs

| Name | Required | Description |
|---|---|---|
| spec_repo_path | Yes | Path to the spec repository |
| expected_title | Yes | Expected project title |

## Verification

```bash
test -f "$spec_repo_path/synchestra-spec-repo.yaml"
title=$(grep 'title:' "$spec_repo_path/synchestra-spec-repo.yaml" | head -1 | sed 's/title: *//')
test "$title" = "$expected_title"
```

## Scenarios

(None yet.)
