# globalconfig

Reads the global Synchestra configuration from `~/.synchestra.yaml` and resolves the `repos_dir` setting with `~` expansion and default fallback to `~/synchestra/repos`.

Returns a zero-value config (no error) when the file does not exist, so callers always get usable defaults.

## Outstanding Questions

None at this time.
