# reporef

Repository reference parsing and resolution.

This package provides utilities for parsing Synchestra repository references in multiple formats (short form, HTTPS URLs, HTTP URLs, and SSH URLs) and resolving them to local filesystem paths and canonical HTTPS origin URLs.

## Usage

```go
ref, err := reporef.Parse("github.com/acme/acme-spec")
if err != nil {
	// handle error
}

// Get local filesystem path
localPath := ref.LocalPath("/home/user/synchestra/repos")
// => /home/user/synchestra/repos/github.com/acme/acme-spec

// Get canonical HTTPS origin URL
originURL := ref.OriginURL()
// => https://github.com/acme/acme-spec
```

## Supported Formats

The `Parse` function accepts repository references in the following formats:

- **Short form:** `github.com/org/repo`
- **HTTPS URL:** `https://github.com/org/repo` or `https://github.com/org/repo.git`
- **HTTP URL:** `http://github.com/org/repo`
- **SSH URL:** `git@github.com:org/repo` or `git@github.com:org/repo.git`

The `.git` suffix is automatically stripped if present. The reference must have exactly three path components (hosting/org/repo); sub-paths and incomplete references are rejected with an error.

## Outstanding Questions

None at this time.
