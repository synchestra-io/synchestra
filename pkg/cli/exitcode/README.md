# exitcode

Shared exit code constants, error type, and constructor helpers for all Synchestra CLI commands.

## Overview

This package centralizes the exit code contract defined in
[spec/features/cli/README.md](../../../../spec/features/cli/README.md). Every CLI command package
imports `exitcode` instead of defining its own error type.

## Usage

```go
import "github.com/synchestra-io/synchestra/pkg/cli/exitcode"

// Use named constructors for standard codes:
return exitcode.InvalidArgsError("--task is required")
return exitcode.NotFoundErrorf("feature not found: %s", id)
return exitcode.UnexpectedErrorf("writing file: %v", err)

// Or use New/Newf for group-specific codes:
return exitcode.Newf(30, "feature-specific error: %s", detail)
```

## Outstanding Questions

None at this time.
