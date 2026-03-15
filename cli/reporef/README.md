# reporef

Parses repository references in any of three formats (HTTPS URL, SSH URL, short `hosting/org/repo` path), resolves them to local disk paths under `repos_dir`, and provides canonical HTTPS origin URLs.

The `Ref` type carries the parsed `Hosting`, `Org`, and `Repo` fields and exposes `OriginURL()`, `DiskPath(reposDir)`, and `Identifier()` methods.

## Outstanding Questions

None at this time.
