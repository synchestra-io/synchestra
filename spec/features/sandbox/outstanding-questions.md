# Sandbox Spec — Outstanding Questions

Consolidated summary of all unresolved design questions from the sandbox feature spec.
Each question is grouped by theme, linked to its source document, and accompanied by a
recommended resolution.

> **How to use this document:** Review questions by theme. When a decision is made, move
> the resolved item into the source document's "Outstanding Questions" section with a
> `**Resolved**` annotation, and remove it from this file.

---

## Credential Management

Questions about credential storage, encryption, lifecycle, and API surface.

### 1. Should credentials support expiry and auto-rotation? What is the timeline?

**Source:** [README.md](README.md)

Credential expiry is already implemented; the open question is whether the system should
automatically rotate (re-encrypt with a new key, refresh external tokens) without manual
intervention.

- **Recommended: Implement expiry notifications first (v1); defer auto-rotation to v2.**
  Expiry is the safety net; rotation is the convenience layer. Ship the event that fires
  on expiry so integrators can build rotation themselves, then add built-in rotation once
  the event bus is proven.

### 2. Should there be a `ListCredentials` RPC that returns identifiers (not values) for UI display?

**Source:** [credentials.md](credentials.md)

Without a list operation, UIs and CLI tools have no way to show which credentials exist
for a given container without attempting to read each by name.

- **Recommended: Yes — add `ListCredentials` returning `(identifier, created_at, expires_at)` tuples.**
  This is a small surface area increase with high UX value. Never return plaintext values
  in the list response.

### 3. Should credential expiry trigger a notification/event via the event bus?

**Source:** [credentials.md](credentials.md)

If credentials silently expire, commands that depend on them will fail with opaque errors.
Proactive notification lets project owners rotate before breakage.

- **Recommended: Yes — emit a `credential.expiring` event 24 h before expiry and a `credential.expired` event on expiry.**
  This follows the principle of making errors explicit and informative. The event bus
  already exists; adding two event types is low-effort.

### 4. Should the encryption key be derivable from a user-provided passphrase (PBKDF2/Argon2)?

**Source:** [credentials.md](credentials.md)

Deriving the encryption key from a passphrase adds a layer of defense: even if the host
database leaks, credentials stay encrypted under a secret only the user knows.

- Option A: **Derive from passphrase using Argon2id.** Strongest security posture, but
  requires the user to supply the passphrase on every container start (or cache it in a
  session). Adds UX friction.
- Option B (**Recommended**): **Keep the current host-generated key for v1; offer passphrase
  derivation as an opt-in feature in v2.** This avoids blocking the MVP on key-management
  UX while leaving the door open. Document the threat model trade-off so users can decide.

### 5. Should `DeleteCredential` be added as an explicit RPC?

**Source:** [credentials.md](credentials.md)

Currently credentials can be overwritten via `SetCredential` but not explicitly removed.
Without delete, stale credentials linger as environment variables inside the container.

- **Recommended: Yes — add `DeleteCredential(identifier)`.** This is essential for
  credential hygiene. It should also emit a `credential.deleted` event for audit purposes.

### 6. What is the convention for mapping `api_key` identifiers to environment variable names?

**Source:** [credentials.md](credentials.md)

Credentials are injected as environment variables, but the mapping from an identifier like
`github_token` to `GITHUB_TOKEN` needs to be deterministic and documented.

- **Recommended: Uppercase the identifier and replace non-alphanumeric characters with `_`.**
  For example, `my-api.key` → `MY_API_KEY`. Document this transformation in the credential
  spec and agent implementation guide, and log the resulting env var name at injection time.

### 7. Should we support credential rotation (re-encrypt all credentials with a new key)?

**Source:** [agent-implementation-guide.md](agent-implementation-guide.md)

If the encryption key is compromised or rotated as policy, all stored credentials need
re-encryption. Without a bulk operation this is painful and error-prone.

- **Recommended: Yes, but defer to v2.** Implement a `RotateEncryptionKey` internal
  operation (not an RPC — triggered by the orchestrator) that decrypts all credentials
  with the old key and re-encrypts with the new one in a single transaction. This depends
  on having a working credential store first.

---

## Container Lifecycle & Resources

Questions about container states, auto-pause behavior, resource management, and eviction.

### 8. Should there be a "maintenance" state where the container is running but not accepting new commands?

**Source:** [lifecycle.md](lifecycle.md)

During image updates or workspace migrations, the container is alive but shouldn't accept
user commands. Without a dedicated state, the orchestrator must choose between "running"
(misleading) and "stopped" (disruptive).

- **Recommended: Yes — add a `maintenance` state.** It should reject new `ExecuteCommand`
  RPCs with a `UNAVAILABLE` status code and a human-readable reason. The orchestrator
  sets this state before performing updates, then transitions back to `running`. This is
  a small state-machine addition with clear operational value.

### 9. Should auto-pause be disabled for specific high-priority projects?

**Source:** [lifecycle.md](lifecycle.md)

Some projects run long-lived background services (dev servers, watchers) where pausing
would be disruptive. A blanket auto-pause policy doesn't fit all workloads.

- **Recommended: Yes — make auto-pause policy per-project configurable via project metadata.**
  Add a `disable_auto_pause: bool` field to the project configuration. Default to
  auto-pause enabled (safe default). Projects that opt out accept higher resource
  consumption.

### 10. Should the workspace cache TTL be per-project configurable, or global only?

**Source:** [lifecycle.md](lifecycle.md)

Different projects have different workspace sizes and rebuild costs. A global TTL either
wastes disk on small projects or evicts large projects too aggressively.

- **Recommended: Per-project configurable with a global default.** Use a global default
  (e.g., 7 days) that individual projects can override via project config. This follows
  the pattern of "opinionated defaults, configurable details" already established in the
  project.

### 11. Should lifecycle hooks support custom scripts (pre-start, post-stop) for project-specific initialization?

**Source:** [lifecycle.md](lifecycle.md)

Projects may need to run database migrations, install extra tools, or warm caches before
the container is ready for user commands.

- Option A (**Recommended**): **Support `pre-start` and `post-stop` hooks defined in
  project config.** Run them inside the container with the same user and resource limits.
  Keep the hook interface minimal: a shell command string with a timeout. This is a
  well-understood pattern (Docker entrypoint, Git hooks) that avoids inventing a plugin
  system.
- Option B: Defer hooks and rely on the existing `docker-entrypoint.sh` for
  initialization. Simpler but less flexible for per-project customization.

### 12. Should eviction emit a user-facing notification to the project owner?

**Source:** [lifecycle.md](lifecycle.md)

If a container is evicted due to resource pressure, the project owner may not know until
their next command fails. Silent eviction degrades trust.

- **Recommended: Yes — emit both an internal event and a user-facing notification.**
  The internal `container.evicted` event feeds monitoring; the user-facing notification
  (via the project's configured notification channel) tells the owner what happened and
  how to restart. This aligns with the "make errors explicit" principle.

### 13. What happens when a paused container's host runs low on memory — evict or OOM?

**Source:** [lifecycle.md](lifecycle.md)

Paused containers (SIGSTOP'd / frozen cgroups) still consume memory. Under memory pressure
the kernel OOM killer may terminate them unpredictably, or the orchestrator can proactively
stop them.

- **Recommended: Proactively stop paused containers before OOM.** The orchestrator should
  monitor host memory and transition paused containers to `stopped` (with workspace
  cached) when memory drops below a configurable threshold. This is more predictable than
  relying on the OOM killer, preserves workspace state, and avoids corrupted container
  filesystems.

---

## API & Protocol

Questions about the HTTP API surface, WebSocket behavior, and gRPC protocol extensions.

### 14. Should the execute endpoint support SSE (Server-Sent Events) as an alternative to WebSocket?

**Source:** [http-api.md](http-api.md)

SSE is simpler to implement for clients (plain HTTP, auto-reconnect built into browsers)
but is unidirectional (server → client only). WebSocket supports bidirectional
communication.

- **Recommended: Yes — offer SSE as a read-only alternative for output streaming.**
  Many consumers only need to tail output and don't need stdin. SSE is easier to integrate
  from curl, browser `EventSource`, and simple HTTP clients. Keep WebSocket as the
  full-featured option for interactive sessions. This is additive, not a replacement.

### 15. Should the WebSocket connection support client→server messages (stdin for interactive processes)?

**Source:** [http-api.md](http-api.md)

Without stdin support, interactive tools (REPLs, editors, debuggers) cannot be used
through the WebSocket connection.

- **Recommended: Yes — support stdin frames on the WebSocket.** Use a typed message
  envelope (`{ "type": "stdin", "data": "..." }`) to distinguish stdin from control
  messages. This is the primary reason WebSocket exists over SSE; not supporting stdin
  would eliminate WebSocket's key advantage.

### 16. Should session output replay on WebSocket reconnection support byte-offset–based resume?

**Source:** [http-api.md](http-api.md)

On reconnect, clients need to resume from where they left off. Byte offsets are precise
but require the server to track output position; sequence numbers are simpler but coarser.

- **Recommended: Use sequence-number–based resume for v1; consider byte-offset for v2.**
  Sequence numbers (per-message incrementing IDs) are simpler to implement on both server
  and client, and sufficient for the vast majority of use cases. Byte-offset resume adds
  complexity (partial message handling, encoding concerns) for marginal benefit.

### 17. Should we add Prometheus metrics support (command execution count, latencies)?

**Source:** [protocol.md](protocol.md)

Metrics at the gRPC layer provide visibility into command execution patterns, error rates,
and latency distributions.

- **Recommended: Yes — expose standard gRPC server metrics via a Prometheus `/metrics` endpoint.**
  Use the `go-grpc-prometheus` interceptor for automatic per-RPC metrics. Add custom
  counters for `commands_executed_total`, `commands_failed_total`, and a histogram for
  `command_duration_seconds`. This is low-effort with high operational value.

### 18. Should we support streaming from multiple log streams (stdout + stderr merged)?

**Source:** [protocol.md](protocol.md)

Currently it's unclear whether stdout and stderr are delivered as separate tagged streams
or merged. Separate streams allow clients to render errors differently; merged streams
are simpler.

- **Recommended: Tag each output chunk with its stream (`stdout` / `stderr`) and let the client decide whether to merge.**
  The gRPC `ExecuteResponse` stream message should include a `stream` field
  (`STDOUT`, `STDERR`). Clients that want merged output can ignore the field. This
  preserves information without forcing a presentation decision.

---

## Observability & Monitoring

Questions about metrics collection, log aggregation, tracing, and alerting.

### 19. Should container-side metrics be pushed (via event bus) or pulled (via orchestrator proxy)?

**Source:** [monitoring.md](monitoring.md)

Push-based metrics (container → event bus → collector) work well for ephemeral containers
that may disappear before a scrape. Pull-based (Prometheus scraping a proxy) is the
standard Prometheus model.

- **Recommended: Push-based for v1 via the event bus.** Sandbox containers are ephemeral
  and may be paused/stopped at any time, which makes pull-based scraping unreliable. The
  event bus already exists for lifecycle events; adding metric payloads is incremental.
  If the project grows to support long-lived containers, add a pull-based option later.

### 20. Should there be a dedicated log aggregation pipeline (ELK/Loki) or stdout-based collection?

**Source:** [monitoring.md](monitoring.md)

A dedicated pipeline provides search, retention, and dashboards. Stdout-based collection
(Docker log driver → external collector) is simpler but less capable.

- **Recommended: Stdout-based (structured JSON) collection for v1; defer dedicated pipeline.**
  Structured JSON to stdout works with any log driver (Docker → Loki, Fluentd, CloudWatch).
  This avoids coupling the sandbox to a specific log backend. Document a recommended Loki
  setup as an optional deployment guide for teams that need search.

### 21. Should session command output be indexed for search?

**Source:** [monitoring.md](monitoring.md)

Indexing command output would allow users and operators to search past session history
("find the session where the build failed on Tuesday").

- Option A: Index output in a search backend (Elasticsearch/Loki). Powerful but adds
  operational overhead and storage costs.
- Option B (**Recommended**): **Don't index for v1.** Session output is already stored in
  the session log files and available via the API for recent sessions. Full-text search
  across all sessions is a nice-to-have, not a must-have. Revisit when there's a concrete
  user need.

### 22. What OpenTelemetry exporter should be the default?

**Source:** [monitoring.md](monitoring.md)

OpenTelemetry supports multiple exporters (OTLP, Jaeger, Zipkin, stdout). The default
affects out-of-the-box experience and documentation.

- **Recommended: OTLP (gRPC) as the default, with stdout as the zero-config fallback.**
  OTLP is the vendor-neutral standard and works with Jaeger, Tempo, Datadog, and most
  backends. Stdout exporter (JSON) requires no infrastructure and is useful for local
  development. Configure via `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable,
  falling back to stdout if unset.

### 23. Should alert thresholds be configurable at runtime or fixed at deploy time?

**Source:** [monitoring.md](monitoring.md)

Runtime-configurable thresholds allow operators to tune alerting without redeploying.
Deploy-time thresholds are simpler and less prone to accidental misconfiguration.

- **Recommended: Deploy-time configuration via config file for v1; runtime API for v2.**
  Alert thresholds change infrequently. A YAML/JSON config file (loaded at startup) is
  sufficient and keeps the system simple. Runtime configurability adds an API surface
  and persistence requirement that isn't justified until the project has a multi-tenant
  operations team.

---

## Database & Multi-tenancy

Questions about schema design, audit logging, retention, and tenant isolation.

### 24. What is the retention policy for access mapping history?

**Source:** [database-schema.md](database-schema.md)

Access mappings (which users accessed which containers) accumulate over time. Without a
retention policy, the table grows unboundedly.

- **Recommended: 90-day retention for access mapping history, enforced by a periodic cleanup job.**
  This is long enough for incident investigation and auditing, short enough to avoid
  unbounded growth. Implement as a simple SQL `DELETE WHERE updated_at < NOW() - INTERVAL 90 DAY`
  run by a cron job or the orchestrator's background loop.

### 25. Should we add a separate audit/event log table?

**Source:** [database-schema.md](database-schema.md)

Currently lifecycle events and credential operations are logged to stdout. A database
table provides queryable, durable audit history.

- **Recommended: Yes — add an `audit_events` table with `(id, timestamp, event_type, actor, resource, payload)` columns.**
  Audit logs are essential for security posture, especially for credential operations.
  Use append-only writes (no UPDATE/DELETE) and a separate retention policy (e.g., 1 year).
  This is a small schema addition with outsized security value.

### 26. For multi-tenant deployments, do we need tenant_id isolation at the database level?

**Source:** [database-schema.md](database-schema.md)

If multiple organizations share a Synchestra deployment, data isolation must be enforced.
Database-level isolation (tenant_id column + row-level security) is the standard approach.

- **Recommended: Add `tenant_id` to all tables now, enforce at the application layer for v1.**
  Adding the column early is cheap and avoids a painful migration later. Application-layer
  enforcement (middleware that filters by tenant_id) is sufficient for v1. If the project
  grows to handle untrusted multi-tenant workloads, upgrade to PostgreSQL row-level
  security policies.

### 27. Should resource_quota fields be enforced at database layer or application layer?

**Source:** [database-schema.md](database-schema.md)

Resource quotas (CPU, memory, disk) can be enforced via database constraints (CHECK,
triggers) or in application code. Database enforcement is harder to bypass but less
flexible; application enforcement is easier to evolve.

- **Recommended: Application-layer enforcement backed by database-stored configuration.**
  Store quota values in the database; enforce them in the orchestrator when creating or
  resizing containers. Database CHECK constraints are too rigid for complex quota logic
  (e.g., burst allowances, soft vs. hard limits). The orchestrator already manages
  container resources — keep the enforcement logic there.

---

## Testing & CI/CD

Questions about testing strategy, CI pipeline, and test infrastructure.

### 28. Should there be a chaos testing framework for simulating Docker daemon failures?

**Source:** [testing.md](testing.md)

The orchestrator must handle Docker daemon crashes, network partitions, and hung
containers gracefully. Without chaos testing, these code paths are exercised only in
production.

- **Recommended: Yes, but keep it simple — use a mock Docker client that injects failures.**
  Don't adopt a full chaos engineering platform (Chaos Monkey, Litmus) for v1. Instead,
  create a `FaultyDockerClient` wrapper that can be configured to fail specific operations
  (create, start, stop) with specific errors. Use it in integration tests. This covers
  the critical paths without operational overhead.

### 29. Should performance benchmarks be included as part of CI?

**Source:** [testing.md](testing.md)

Benchmarks in CI catch performance regressions early. However, CI environments have
variable performance, leading to flaky benchmark results.

- **Recommended: Run Go benchmarks in CI but don't fail the build on regression.**
  Use `go test -bench` and store results as CI artifacts. Use `benchstat` to compare
  against a baseline and post a comment on PRs with significant changes. Alert on
  regressions but don't block merges — CI hardware variance makes hard thresholds
  unreliable.

### 30. Should there be snapshot/golden-file tests for gRPC protocol messages?

**Source:** [testing.md](testing.md)

Golden-file tests catch unintended changes to the wire format of gRPC messages, which
could break backward compatibility.

- **Recommended: Yes — add golden-file tests for all gRPC response messages.**
  Serialize representative responses to JSON, store them as `.golden` files, and compare
  in tests. Use `go test -update` flag to regenerate. This is lightweight, catches
  accidental breaking changes, and documents the expected wire format.

### 31. Should integration tests use a dedicated Docker network?

**Source:** [testing.md](testing.md)

A dedicated network prevents test containers from interfering with the host or other test
runs, and enables predictable DNS resolution between test containers.

- **Recommended: Yes — create a test-scoped Docker network per test run.**
  Name it `synchestra-test-{run_id}` and clean it up in test teardown. This isolates
  tests from each other (important for parallel CI), prevents port conflicts, and allows
  container-to-container communication via DNS names.

### 32. What is the retention policy for test container images in the CI registry?

**Source:** [testing.md](testing.md)

CI builds produce container images that accumulate in the registry. Without cleanup,
storage costs grow indefinitely.

- **Recommended: Keep tagged releases indefinitely; delete untagged/PR images after 7 days.**
  Use the container registry's built-in lifecycle policies (GitHub Container Registry,
  Docker Hub, etc.) to auto-delete images older than 7 days that don't match a release
  tag pattern. This keeps CI fast (recent images are cached) while bounding storage.

---

## Security & Compliance

Questions about compliance requirements, process isolation, and security hardening.

### 33. Are there compliance requirements (HIPAA, PCI-DSS, SOC 2) affecting credential handling?

**Source:** [README.md](README.md)

Compliance requirements dictate encryption standards, audit log retention, access controls,
and key management procedures. The answer shapes many other design decisions.

- **Recommended: Target SOC 2 Type II readiness as the baseline; defer HIPAA/PCI-DSS unless a concrete need arises.**
  SOC 2 aligns well with the project's existing security posture (AES-256-GCM encryption,
  audit logging, access controls). HIPAA and PCI-DSS add significant operational burden
  (BAAs, network segmentation, annual audits) that isn't justified unless the project
  handles healthcare or payment data. Document the current security controls against
  SOC 2 criteria so the gap analysis is easy when needed.

### 34. Should cgroup v2 be mandatory or fallback to v1/no limits?

**Source:** [agent-implementation-guide.md](agent-implementation-guide.md)

cgroup v2 provides unified resource control and is the default on modern Linux
distributions. Older hosts may only support cgroup v1 or have no cgroup support.

- **Recommended: Require cgroup v2 for production; allow v1 fallback for development with a warning.**
  cgroup v2 is the default on Ubuntu 22.04+, Fedora 31+, Debian 11+, and Alpine 3.16+.
  Requiring it for production simplifies the code (single code path) and ensures
  consistent resource enforcement. Log a warning on v1 and disable resource limits
  entirely if no cgroup support is detected.

### 35. Should we implement per-process memory limits (ulimits) beyond container cgroup limits?

**Source:** [agent-implementation-guide.md](agent-implementation-guide.md)

Container-level cgroup limits cap total memory for the container. Per-process ulimits add
defense-in-depth against a single runaway process consuming the container's entire
allocation.

- Option A (**Recommended**): **Apply per-process ulimits as a fraction of the container limit (e.g., 80%).**
  This prevents a single command from starving the agent process and other system
  processes inside the container. Set `RLIMIT_AS` via `syscall.Setrlimit` in the command
  executor. The 80% threshold leaves headroom for the gRPC agent and OS overhead.
- Option B: Rely on cgroup limits only. Simpler, but a single `malloc` bomb can crash the
  agent process.

---

## Compute Backends

Questions about deployment topologies, cloud serverless architecture, and backend selection.

### 36. Should Synchestra pre-warm cloud containers (min-instances=1) or accept cold-start latency?

**Source:** [compute-backends.md](compute-backends.md), [cloud-serverless.md](cloud-serverless.md)

Cloud serverless platforms have cold starts of 1-5 seconds. Pre-warming keeps at least one
instance running, eliminating latency but incurring idle cost.

- Option A (**Recommended**): **Accept cold starts by default, allow per-project opt-in to pre-warming.**
  Most projects tolerate a few seconds of initial latency. Pre-warming should be a paid
  premium feature to avoid unbounded cloud costs.
- Option B: Always pre-warm. Better UX but adds $5-15/mo per idle project.

### 37. Should the `external` backend support any lifecycle operations beyond `Connect()`?

**Source:** [compute-backends.md](compute-backends.md)

The external backend (user-provided endpoint) currently only defines `Connect()`. Users
might want Synchestra to report health, restart on failure, or manage workspace sync.

- Option A (**Recommended**): **Connect() + HealthCheck() only.** Keep it minimal — the
  user manages their own infra. Synchestra just monitors reachability.
- Option B: Add optional Restart() and WorkspaceSync(). More capable but blurs the
  ownership boundary.

### 38. Should backend selection be immutable per-project, or should projects migrate between backends?

**Source:** [compute-backends.md](compute-backends.md)

A project starts on Single Host but outgrows it. Can it move to Cloud Serverless without
recreating the sandbox?

- Option A (**Recommended**): **Allow migration with a `synchestra sandbox migrate` command.**
  Requires stop → workspace export → re-create on new backend → workspace import.
  Explicit and auditable.
- Option B: Immutable — create a new sandbox on the desired backend. Simpler but loses
  container history and credential bindings.

### 39. For Kubernetes mode, CRD+operator pattern or simpler Deployment-per-project?

**Source:** [compute-backends.md](compute-backends.md)

CRD+operator gives the most Kubernetes-native experience (custom resources, reconciliation
loops, `kubectl` integration) but is significantly more complex to build.

- Option A: **CRD+operator.** Best long-term but 4-6 weeks to implement.
- Option B (**Recommended for MVP**): **Deployment-per-project with a thin client.** Create
  Deployments/Services via the K8s API directly. Faster to ship, upgrade to operator later.

### 40. Should cloud workspace sync be full snapshots or incremental (rsync-style diffs)?

**Source:** [compute-backends.md](compute-backends.md), [cloud-serverless.md](cloud-serverless.md)

Full snapshots are simple but slow for large workspaces. Incremental sync is faster but
adds complexity (tracking changed files, partial failure handling).

- Option A (**Recommended**): **Full snapshots initially, incremental as optimization.**
  Ship the simple version first. Most workspaces are small (< 100 MB). Add rsync-style
  diffs when workspace sizes cause measurable latency.
- Option B: Incremental from day one. Better performance but higher implementation cost.

### 41. For Submode A (Fully Managed), what pricing model works best?

**Source:** [cloud-serverless.md](cloud-serverless.md)

Synchestra pays the cloud bill and charges users. The pricing model must cover costs with
margin while remaining competitive.

- Option A (**Recommended**): **Per-minute execution time + small monthly base fee.**
  Aligns cost with usage. Base fee covers workspace storage. Transparent and predictable.
- Option B: Flat monthly tiers (e.g., $10/mo for 100 min, $25/mo for 500 min). Simpler
  billing but may over/under-charge depending on usage patterns.

### 42. For Submode B (Delegated), support GCP Workload Identity Federation as alternative to service account keys?

**Source:** [cloud-serverless.md](cloud-serverless.md)

Service account JSON keys are a security liability (long-lived credentials). Workload
Identity Federation provides keyless auth via OIDC tokens.

- Option A (**Recommended**): **Support both, recommend WIF.** Service account keys for
  quick setup; WIF for production. WIF is the GCP best practice and eliminates key rotation.
- Option B: Service account keys only. Simpler to implement but less secure.

### 43. What is the maximum acceptable cold start latency before auto-warming is triggered?

**Source:** [cloud-serverless.md](cloud-serverless.md)

Need a threshold to decide when a project's cold starts are unacceptable and should trigger
pre-warming (if opted in).

- Option A (**Recommended**): **5 seconds.** Cloud Run cold starts are typically 1-3s;
  above 5s suggests a heavy container image that should be optimized or pre-warmed.
- Option B: 10 seconds. More lenient, fewer pre-warming costs, but worse UX for some users.

### 44. For Submode C (External), should Synchestra verify agent protocol compatibility on first connection?

**Source:** [cloud-serverless.md](cloud-serverless.md)

The user manages their own agent. If the agent runs an incompatible protocol version,
commands will fail with confusing errors.

- Option A (**Recommended**): **Yes, mandatory version handshake on Connect().** The agent
  reports its protocol version; Synchestra rejects incompatible versions with a clear error
  message. Low implementation cost, high diagnostic value.
- Option B: No check — let failures surface naturally. Simpler but harder to debug.

### 45. Should there be a Submode B setup wizard in the WebUI?

**Source:** [cloud-serverless.md](cloud-serverless.md)

Setting up delegated access to a user's cloud account requires multiple steps (create
service account, grant roles, configure billing). A wizard could generate the
Terraform/gcloud commands.

- Option A (**Recommended**): **Yes, a step-by-step wizard that generates copy-paste commands.**
  Reduces setup friction significantly. Can output Terraform HCL or `gcloud` CLI commands.
- Option B: Documentation only. Lower implementation cost but higher user friction and
  support burden.

---

## Outstanding Questions

None at this time — this document *is* the consolidated questions list.
