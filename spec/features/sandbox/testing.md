# Integration Testing

Testing strategy for the sandbox feature covering unit tests, integration tests, and end-to-end tests. Tests validate the stateless-host / autonomous-container architecture, session reconnection, credential encryption, and container lifecycle management.

**Location in repo**: `internal/sandbox/` (tests co-located with source)

## Test Categories

### Unit Tests

Unit tests run without Docker. They use interface-based mocks for all external dependencies (Docker client, gRPC connections, credential storage).

#### Orchestrator (`internal/sandbox/orchestrator/`)

**State Machine** (`state_machine_test.go`):
- All valid transitions succeed: `STOPPED→STARTING`, `STARTING→RUNNING`, `RUNNING→PAUSING`, `PAUSING→PAUSED`, `PAUSED→RESUMING`, `RESUMING→RUNNING`, `RUNNING→STOPPING`, `STOPPING→STOPPED`, `*→FAILED`.
- All invalid transitions return `ErrInvalidTransition`: e.g., `PAUSED→STARTING`, `STOPPED→RUNNING`.
- Concurrent transition attempts on the same `{project_id}` are serialized — only one succeeds.
- Transition callbacks fire in order (pre-transition, state change, post-transition).

**Circuit Breaker** (`circuit_breaker_test.go`):
- Starts in CLOSED state; requests pass through.
- After `N` consecutive failures, transitions to OPEN; requests fail immediately with `UNAVAILABLE`.
- After the cooldown window, transitions to HALF-OPEN; one probe request allowed.
- Probe success → CLOSED; probe failure → OPEN.
- Circuit breakers are per-`{project_id}` — one project's failures don't affect another.
- Timing: verify cooldown duration from config is respected (use a fake clock).

**Connection Pool** (`pool_test.go`):
- `Add` creates a connection entry keyed by `{project_id}`.
- `Get` returns an active connection; returns error for unknown `{project_id}`.
- `Remove` closes the connection and deletes the entry.
- Concurrent `Get`/`Add`/`Remove` from multiple goroutines doesn't race (run with `-race`).
- Stale connection detection: connection that fails health check is evicted on next `Get`.
- Pool size metric is updated on add and remove.

**Idle Manager** (`idle_manager_test.go`):
- Container with no activity for `idle_timeout` triggers a pause callback.
- Any session activity (command execution, log stream) resets the idle timer.
- Removing a container from tracking cancels its idle timer.
- Multiple containers tracked independently — one's activity doesn't reset another's timer.
- Timer precision: uses a fake clock to avoid flaky time-based tests.

**Eviction** (`eviction_test.go`):
- LRU ordering: least-recently-used container is selected first.
- Only containers in `PAUSED` or `STOPPED` state are eviction candidates.
- Containers in `RUNNING` state with active sessions are never evicted.
- When at `max_containers` capacity, new provision request triggers eviction of LRU candidate.
- When all containers are `RUNNING` with active sessions, returns `RESOURCE_EXHAUSTED`.
- After eviction, the container's workspace cache is preserved (only container removed).

**Config** (`config_test.go`):
- Env var parsing: `SYNCHESTRA_SANDBOX_MAX_CONTAINERS`, `SYNCHESTRA_SANDBOX_IDLE_TIMEOUT`, etc.
- Defaults applied when env vars are unset.
- Validation: negative timeout rejected, max_containers < 1 rejected, invalid duration format rejected.
- Config is immutable after initialization.

#### gRPC Agent (`internal/sandbox/agent/`)

**Credential Vault** (`vault_test.go`):
- `Store` encrypts and persists a credential; `Get` decrypts and returns it.
- `Delete` removes a credential; subsequent `Get` returns `NOT_FOUND`.
- `Store` with existing key overwrites (upsert behavior).
- Expired credential: `Get` returns `NOT_FOUND` after `expires_at` has passed.
- Key rotation: re-encrypt all credentials with a new master key; old key no longer works.
- AES-256-GCM round-trip: encrypt → persist → load → decrypt yields original plaintext.
- Tampered ciphertext detected (GCM authentication failure).

**Session Manager** (`session_manager_test.go`):
- `CreateSession` returns a unique `session_id` and creates the session directory.
- `ListSessions` returns all active sessions for the container.
- Sessions are isolated: files created in session A's working directory are not visible in session B's.
- Concurrent session creation from multiple goroutines doesn't race.
- `CleanupSession` removes session state; subsequent operations on that `session_id` return `NOT_FOUND`.
- Session directory is created at `/workspace/{project_id}/sessions/{session_id}/`.

**Command Executor** (`executor_test.go`):
- Basic execution: `echo hello` returns exit code 0 and stdout `hello\n`.
- Non-zero exit code: `exit 42` returns exit code 42.
- Timeout: command exceeding `timeout_seconds` is killed and returns `DEADLINE_EXCEEDED`.
- Working directory: command runs in the specified `working_dir`.
- Environment injection: supplied env vars are available to the command process.
- Stderr captured separately from stdout.
- Large output: output exceeding buffer size is streamed without truncation.

**Credential Injection** (`injection_test.go`):
- Git HTTPS token: `GIT_ASKPASS` helper script is created; `git clone` receives the token.
- SSH key: temporary key file is written with `0600` permissions; `GIT_SSH_COMMAND` points to it.
- Env var injection: credential with type `ENV_VAR` is set in the command's environment.
- Temp files are cleaned up after command execution completes.
- Missing credential: injection for unknown `credential_id` returns `NOT_FOUND`.

#### HTTP API (`internal/api/sandbox/`)

**Handlers** (`handler_test.go`):
- Each endpoint tested with a mock orchestrator (interface-based).
- `POST /api/v1/projects/{project_id}/sandbox/exec` → calls orchestrator `Execute`, returns command output.
- `POST /api/v1/projects/{project_id}/sandbox/credentials` → calls orchestrator `StoreCredential`, returns 201.
- `GET /api/v1/projects/{project_id}/sandbox/sessions` → returns session list.
- `GET /api/v1/projects/{project_id}/sandbox/sessions/{session_id}/logs` → upgrades to WebSocket, streams logs.
- Error from orchestrator is mapped to appropriate HTTP status (see Error Mapping below).

**Auth Middleware** (`auth_test.go`):
- Valid bearer token: request passes through to handler.
- Invalid token: returns 401 with `{"error": "invalid_token"}`.
- Missing `Authorization` header: returns 401 with `{"error": "missing_token"}`.
- Expired token: returns 401 with `{"error": "token_expired"}`.

**Authorization** (`authz_test.go`):
- Project member with `execute` permission can call exec endpoints.
- Project member with `read` permission can list sessions and stream logs but cannot execute.
- Non-member returns 403 for all project-scoped endpoints.
- Admin can access admin endpoints (`/api/v1/admin/sandbox/{project_id}/*`).

**Error Mapping** (`errors_test.go`):

| gRPC Code | HTTP Status | Scenario |
|---|---|---|
| `NOT_FOUND` | 404 | Container not provisioned for `{project_id}` |
| `UNAVAILABLE` | 503 | Circuit breaker open, container unhealthy |
| `RESOURCE_EXHAUSTED` | 429 | Max containers reached, rate limit exceeded |
| `DEADLINE_EXCEEDED` | 504 | Command timeout |
| `PERMISSION_DENIED` | 403 | Insufficient access level |
| `INVALID_ARGUMENT` | 400 | Malformed request |
| `INTERNAL` | 500 | Unexpected server error |

**Rate Limiting** (`rate_limit_test.go`):
- Requests within limit succeed (200).
- Request exceeding limit returns 429 with `Retry-After` header.
- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` headers present on every response.
- Rate limits are per-`{project_id}`, not global.

**Request Validation** (`validation_test.go`):
- Missing required field (`command` in exec request) → 400.
- Invalid type (string where int expected) → 400.
- Out-of-range value (`timeout_seconds: -1`) → 400.
- Overly long command string (exceeds max length) → 400.

---

### Integration Tests

Integration tests require Docker and use real gRPC connections. They are gated behind the `integration` build tag.

#### Container Lifecycle (`internal/sandbox/orchestrator/lifecycle_integration_test.go`)

- **Provision**: Create a new container for `{project_id}` from scratch — image pull, container start, agent readiness check via gRPC `Ping`. Verify container state is `RUNNING`.
- **Auto-pause**: Set idle timeout to 5s. Wait with no activity. Verify container transitions to `PAUSED` state. Verify Docker container is paused.
- **Auto-resume**: Send a request to a paused container. Verify container resumes and response arrives within the 3s latency target. Verify state returns to `RUNNING`.
- **Graceful shutdown**: Start a long-running command, then trigger shutdown. Verify active session drains (command completes or receives cancellation) before container stops.
- **Failure detection**: Kill the agent process inside the container. Verify orchestrator detects failure via health check and transitions state to `FAILED`. Verify auto-restart creates a new container.
- **Terminate without cache**: `DELETE /api/v1/admin/sandbox/{project_id}` without `clear_cache`. Verify container removed but workspace directory at `{WORKSPACE_ROOT}/{project_id}/` preserved.
- **Terminate with cache**: `DELETE /api/v1/admin/sandbox/{project_id}?clear_cache=true`. Verify both container and workspace directory removed.
- **Eviction**: Provision `max_containers` containers. Pause the oldest. Request a new `{project_id}`. Verify the oldest paused container is evicted and the new one is created.

#### gRPC Communication (`internal/sandbox/agent/grpc_integration_test.go`)

- **Execute and stream**: Send `ExecuteCommand` RPC with `echo hello`. Receive streamed `CommandOutput` messages. Verify stdout contains `hello`.
- **Concurrent execution**: Open two sessions on the same container. Execute commands concurrently. Verify outputs are not interleaved (each stream receives only its own output).
- **Health check**: Call `Ping` on a healthy container — returns success. Pause the container — `Ping` fails with `UNAVAILABLE`. Resume — `Ping` succeeds again.
- **Credential round-trip**: `StoreCredential` via gRPC → `GetCredential` via gRPC → verify plaintext matches. Delete → verify `NOT_FOUND`.
- **Connection recovery**: Execute a command. Pause the container (gRPC connection drops). Resume the container. Execute another command on the same `{project_id}`. Verify the pool reconnects transparently.

#### Session Reconnection (`internal/sandbox/orchestrator/reconnect_integration_test.go`)

- **Disconnect during execution**: Start a long-running command (`sleep 5 && echo done`). Close the client gRPC stream mid-execution. Verify the command continues running inside the container (check session log file).
- **Reconnect via StreamLogs**: After disconnect, call `StreamLogs` with the same `session_id`. Verify buffered output is received (including output generated while disconnected).
- **Reconnect from different connection**: Establish a new gRPC connection (simulating a different device). Call `StreamLogs` with the original `session_id`. Verify logs are streamed from the beginning.
- **Survive pause/resume**: Create a session, execute a command, pause the container, resume. Verify session state (working directory, env vars) is restored.
- **Survive restart**: Create a session, execute a command that writes to a log file. Restart the container. Verify session logs at `/workspace/{project_id}/sessions/{session_id}/logs/` are intact on the volume.

#### Credential Flow (`internal/sandbox/agent/credential_integration_test.go`)

- **HTTP → container vault**: Store a credential via `POST /api/v1/projects/{project_id}/sandbox/credentials`. Verify it is encrypted in the container's vault (inspect vault file — ciphertext, not plaintext).
- **Credential injection in command**: Store a git HTTPS token. Execute `git clone` with credential injection. Verify clone succeeds (the credential was injected via `GIT_ASKPASS`).
- **Persist across restart**: Store a credential. Restart the container. Retrieve the credential. Verify it decrypts correctly (master key and vault are on the volume).
- **Expired credential rejected**: Store a credential with a short `expires_at`. Wait for expiry. Execute a command with credential injection. Verify the command fails with a clear error indicating credential expiry.

---

### End-to-End Tests

Full-stack tests exercise the entire path: HTTP → Orchestrator → gRPC → Container → Response. Gated behind the `e2e` build tag.

#### `internal/sandbox/e2e_test.go`

**Credential + Git Clone**:
1. `POST /api/v1/projects/{project_id}/sandbox/credentials` — store a git HTTPS token.
2. `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `git clone https://github.com/example/repo.git`.
3. Verify 200 response with clone output. Verify repo exists in the container's working directory.

**Long-Running Command + Reconnect**:
1. `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `for i in $(seq 1 10); do echo $i; sleep 1; done`.
2. Open WebSocket to `GET /api/v1/projects/{project_id}/sandbox/sessions/{session_id}/logs`.
3. Receive first few lines, then close the WebSocket (simulating disconnect).
4. Wait 5 seconds.
5. Reopen WebSocket to the same `{session_id}`.
6. Verify all 10 lines are received (including those generated during disconnect).

**Concurrent Users, Isolated Sessions**:
1. User A: `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `echo user_a > /tmp/test.txt`.
2. User B: `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `cat /tmp/test.txt` (in a different session).
3. Verify User B cannot see User A's file (sessions have isolated working directories).

**Idle → Pause → Resume**:
1. `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `echo warm`.
2. Wait for idle timeout (configured short for test, e.g., 5s).
3. Verify container state is `PAUSED` via `GET /api/v1/admin/sandbox/{project_id}`.
4. `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `echo resumed`.
5. Verify response arrives within 3s (resume latency target). Verify output is `resumed`.

**Crash → Restart → Retry**:
1. `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `echo before_crash`.
2. Kill the agent process inside the container (simulate crash via a special test command or Docker exec).
3. Wait for orchestrator health check to detect failure and auto-restart.
4. `POST /api/v1/projects/{project_id}/sandbox/exec` — execute `echo after_restart`.
5. Verify response succeeds with output `after_restart`.

**Eviction Under Pressure**:
1. Provision containers for `{project_id}` A, B, C (where `max_containers=3`).
2. Allow container A to idle and get paused.
3. `POST /api/v1/projects/{project_id_d}/sandbox/exec` — request a 4th project's sandbox.
4. Verify container A is evicted (state removed from orchestrator).
5. Verify the new container for project D is `RUNNING` and the command succeeds.

---

## Test Infrastructure

### Mock Docker Client

An interface-based Docker client (`internal/sandbox/orchestrator/docker.go`) allows unit testing without a real Docker daemon.

```go
// DockerClient defines the interface for container operations.
type DockerClient interface {
    CreateContainer(ctx context.Context, opts CreateOpts) (string, error)
    StartContainer(ctx context.Context, containerID string) error
    PauseContainer(ctx context.Context, containerID string) error
    UnpauseContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
    RemoveContainer(ctx context.Context, containerID string) error
    InspectContainer(ctx context.Context, containerID string) (ContainerInfo, error)
}
```

The mock implementation (`internal/sandbox/orchestrator/docker_mock_test.go`):
- Returns configurable responses per method call (success, error, delay).
- Simulates failure modes: image not found, create timeout, OOM kill, daemon unavailable.
- Records call history for assertion (e.g., verify `PauseContainer` was called with correct ID).
- Thread-safe for concurrent test execution.

### Test Container Image

A lightweight test image for integration and e2e tests:

- **Base**: Alpine Linux with a minimal gRPC agent binary.
- **Startup time**: <2s (production images may be larger).
- **Configurable behaviors** via environment variables:
  - `TEST_AGENT_STARTUP_DELAY` — simulate slow agent startup.
  - `TEST_AGENT_FAIL_AFTER` — agent crashes after N requests (for failure testing).
  - `TEST_AGENT_PING_FAIL` — health check returns error (for circuit breaker testing).
- **Build**: `docker build -f Dockerfile.test -t synchestra-sandbox-test:latest .`

### Test Fixtures

**Sample State Repositories**:
- `testdata/repos/minimal/` — minimal `.synchestra/` directory with a single task definition.
- `testdata/repos/with-credentials/` — repository requiring git credentials for submodule fetch.
- `testdata/repos/large-output/` — task that produces >1MB of stdout (for streaming tests).

**Sample Credentials**:
- `testdata/credentials/` — test encryption keys and pre-encrypted vault files for round-trip testing.
- Never contains real secrets; uses deterministic test values (`test-token-12345`).

**Docker Compose** (`testdata/docker-compose.test.yml`):
- Starts the test container image with pre-configured volume mounts.
- Exposes the gRPC socket for integration tests.
- Configurable via `.env.test` for CI environments.

---

## Test Execution

### Running Tests

```bash
# Unit tests (no Docker required)
go test ./internal/sandbox/... -short

# Integration tests (requires Docker)
go test ./internal/sandbox/... -tags=integration

# End-to-end tests (requires Docker + HTTP server)
go test ./internal/sandbox/... -tags=e2e

# All tests with race detection
go test -race ./internal/sandbox/...

# Coverage report
go test -coverprofile=coverage.out ./internal/sandbox/...
go tool cover -html=coverage.out -o coverage.html

# Run a specific test by name
go test ./internal/sandbox/orchestrator/ -run TestStateMachine_InvalidTransition -v
```

### CI/CD Integration

| Stage | Trigger | Tests | Docker Required |
|---|---|---|---|
| PR check | Every pull request | Unit tests (`-short`) | No |
| Merge to main | Push to `main` | Unit + Integration (`-tags=integration`) | Yes |
| Nightly | Scheduled (daily) | Unit + Integration + E2E (`-tags=e2e`) | Yes |
| Release | Tag push (`v*`) | Full suite + race detection | Yes |

**Coverage Thresholds** (enforced in CI):

| Package | Minimum Coverage |
|---|---|
| `internal/sandbox/orchestrator/` | 80% |
| `internal/sandbox/agent/` (credential vault) | 90% |
| `internal/sandbox/agent/` (other) | 80% |
| `internal/api/sandbox/` | 80% |

**CI Configuration Notes**:
- Docker-in-Docker or a mounted Docker socket is required for integration/e2e stages.
- Test timeout: 5 minutes for unit, 15 minutes for integration, 30 minutes for e2e.
- Tests produce JUnit XML output via `gotestsum` for CI reporting.
- Coverage reports uploaded as artifacts for PR review.

---

## Security Test Cases

Security-specific tests are included in the relevant test files but are also enumerated here for audit purposes.

**Credential Secrecy**:
- After running all credential tests, grep the test output and log files for known test secret values (`test-token-12345`, etc.). Fail if any plaintext secret is found in logs.
- Inspect the host database after `StoreCredential` — verify no credential values are stored on the host (credentials live only inside the container vault).

**Session Isolation**:
- User A creates a session and writes a file. User B creates a separate session on the same `{project_id}`. User B attempts to read User A's file via path traversal (`../../sessions/{session_a_id}/`). Verify access is denied.

**Credential Expiry Enforcement**:
- Store a credential with `expires_at` set to 1 second in the future. Wait 2 seconds. Attempt credential injection. Verify the command receives an error, not the credential.

**Socket Isolation**:
- Container for `{project_id}` A must not be able to access the Unix socket for `{project_id}` B. Verify by attempting to connect to `/var/run/synchestra-{project_id_b}.sock` from inside container A — connection should be refused (socket is not mounted).

**Resource Limits**:
- Execute a command that allocates memory beyond the container's limit. Verify the process is OOM-killed and the orchestrator reports the failure (not a hang).
- Execute a CPU-intensive infinite loop. Verify the command times out at `timeout_seconds` and is killed.

**Input Sanitization**:
- Execute a command containing shell metacharacters (`; rm -rf /`). Verify the command is executed literally (no shell injection) or rejected by validation.

---

## Outstanding Questions

1. Should there be a chaos testing framework for simulating Docker daemon failures (e.g., random pause/kill of the Docker socket during test runs)?
2. Should performance benchmarks be included as part of CI (e.g., max concurrent sessions, resume latency P99, provision time P95)?
3. Should there be snapshot/golden-file tests for the gRPC protocol messages to catch unintended schema changes?
4. Should integration tests use a dedicated Docker network to avoid conflicts with other services on the CI host?
5. What is the retention policy for test container images in the CI registry?
