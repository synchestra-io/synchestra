# globalconfig

Loads the user-level Synchestra configuration from `~/.synchestra.yaml`.

The `globalconfig` package provides the `Load` function, which reads the global configuration file from the user's home directory. If the file does not exist, sensible defaults are returned. The package supports tilde expansion (`~` and `~/`) for relative paths in the configuration.

## Files

- `globalconfig.go` — Core implementation of `Load` and configuration structures.
- `globalconfig_test.go` — Test suite covering file presence, custom configuration, and tilde expansion.

## Outstanding Questions

None at this time.
