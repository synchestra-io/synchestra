# Command Group: `synchestra config`

**Parent:** [CLI](../README.md)

Commands for managing the global user configuration stored in [`~/.synchestra.yaml`](../../global-config/README.md).

## Arguments

Shared arguments for `synchestra config` subcommands are documented in the [_args](_args/README.md) directory: [`--repos-dir`](_args/repos-dir.md).

## Commands

| Command | Description |
|---|---|
| [show](show/README.md) | Display the current configuration with defaults applied |
| [set](set/README.md) | Set one or more config values |
| [clear](clear/README.md) | Clear a config value back to its default |

### `show`

Displays the effective configuration from `~/.synchestra.yaml`. Fields that are absent or empty are populated with their default values in the output, so consumers always see the complete picture without needing to know the default logic. See [show/README.md](show/README.md).

### `set`

Sets one or more config values in `~/.synchestra.yaml`. Creates the file if it does not exist. Empty values are not allowed. See [set/README.md](set/README.md).

### `clear`

Removes a config value from `~/.synchestra.yaml`, reverting it to its default. Only fields that support default values can be cleared. See [clear/README.md](clear/README.md).

## Outstanding Questions

None at this time.
