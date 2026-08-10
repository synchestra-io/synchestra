# runner CLI package

Implements deterministic caller operations for the frozen `synchestra.dispatch.v1` protocol. The package resolves immutable Git and SpecScore inputs without changing the checkout, sends caller requests to the public Hub API, and renders stable text or single-object JSON output.

The HTTP client is intentionally limited to caller endpoints. Scheduler claim, lease, heartbeat, and attempt-owner mutations are not part of this package.

## Open Questions

None at this time.
