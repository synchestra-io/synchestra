# GitHub Actions workflows

This directory contains GitHub Actions workflows that validate repository changes in automation.

Go CI passes the organization `GO_PRIVATE_TOKEN` to the shared workflow and scopes `GOPRIVATE` to `github.com/synchestra-io/*`, allowing private organization modules to be fetched without broadening the token to unrelated hosts.

## Outstanding Questions

None at this time.
