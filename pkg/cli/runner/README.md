# runner CLI package

Implements deterministic caller operations for the frozen `synchestra.dispatch.v1` protocol. The package resolves immutable Git and SpecScore inputs without changing the checkout, sends caller requests to the public Hub API, and renders stable text or single-object JSON output.

`runner invoke` adds a typed adapter for the fixed WB handlers. It reads bounded
opaque JSON, derives integrity and routing metadata, and deliberately projects
invocation responses without serializing the reserved compatibility envelope or
synthetic handler selectors. Invocation status, logs, retry, and cancellation
continue to use the ordinary dispatch endpoints and attempt lifecycle. The
typed projection retains structured state, fencing, failure codes, and artifact
or log references while omitting terminal summaries, failure text,
cancellation reasons, and log messages that could contain opaque payload data.
Ordinary dispatch output remains unchanged.

The HTTP client is intentionally limited to caller endpoints. Scheduler claim, lease, heartbeat, and attempt-owner mutations are not part of this package.

## Open Questions

None at this time.
