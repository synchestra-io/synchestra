# Sandbox Security Model

## Executive Summary

The Sandbox feature is designed with **security by architecture**: isolating secrets in containers, preventing host compromise from leaking credentials, and enforcing user isolation within shared containers. This document details the threat model, security mechanisms, and validation strategies.

**Key Principles:**
1. **Host has no secrets**: Host cannot access encrypted credentials, decryption keys, or unencrypted tokens
2. **Containers are security boundary**: Each container is isolated from others; compromise of one does not affect others
3. **Encryption at rest**: All secrets stored in containers encrypted with AES256
4. **Defense in depth**: Multiple layers of isolation (filesystem, process, network, resource limits)

## Threat Model

### Adversaries & Attack Vectors

#### 1. Host Compromise

**Threat**: Attacker gains root access to host machine. Can they extract secrets?

**Attack Vectors:**
- Exploit host OS vulnerability
- Compromise host daemon (`synchestra serve --http`)
- Gain access to container metadata database

**Impact if Unmitigated:**
- Host could read container memory, disk, or orchestrate malicious commands
- Host could reconstruct encrypted credentials (if keys not isolated)

**Mitigations:**
- Decryption keys generated inside container and never exported to host
- Encrypted credentials stored in container filesystem (not shared with host)
- Host DB contains NO credentials, only metadata and access mappings
- Host can read unencrypted project artifacts (repos, workspaces) but NOT secrets

**Residual Risk**: Host compromise can execute commands inside container, but cannot directly read secrets without container cooperation. Container must be independently hardened to prevent this.

#### 2. Container Escape

**Threat**: Attacker inside container breaks out and compromises host or sibling containers.

**Attack Vectors:**
- Docker/kernel vulnerability (CVE, race condition)
- Misconfigured Docker security options
- Shared volumes or sockets with dangerous permissions

**Impact if Unmitigated:**
- Attacker gains host root access (Docker daemon runs as root)
- Can access host filesystem, stop other containers, read other projects' workspaces
- Can exfiltrate secrets from sibling containers or encrypted stores

**Mitigations:**
- Run container with restricted capabilities: `--cap-drop=ALL --cap-add=NET_BIND_SERVICE`
- Drop: `CAP_SYS_ADMIN`, `CAP_SYS_PTRACE`, `CAP_NET_ADMIN`, etc.
- Read-only root filesystem (where safe): `/` read-only, `/workspace` read-write
- No privileged flag, no host network access
- Seccomp profile: restrict dangerous syscalls
- Cgroups limits: prevent fork bombs, memory exhaustion
- No Docker socket access inside container (unless explicitly needed for DinD, then isolated)
- Regular security updates to base image

**Residual Risk**: Kernel/Docker vulnerabilities can still escape containers (unknown 0-days). Mitigated by:
- Running containers as unprivileged user
- Namespace isolation (containers cannot see host processes)
- Device restrictions (no `/dev/mem`, limited device access)

#### 3. Credential Theft from Container

**Threat**: Attacker with code execution inside a user's session (or container-wide) steals stored credentials.

**Attack Vectors:**
- Compromise of user command execution (e.g., malicious script in cloned repo)
- Compromise of container agent process
- Sidecar container in same Kubernetes cluster (not applicable here, but for future)
- Memory dump of gRPC agent or decrypted credential cache

**Impact if Unmitigated:**
- Attacker extracts git tokens, SSH keys, API keys
- Can clone/push to private repos, access external services (AWS, etc.)

**Mitigations:**
- Credentials stored encrypted on disk (AES256)
- Decryption key never serialized or logged (in-memory only)
- Command execution: decrypt on-demand, pass to subprocess via environment variable or stdin
- Decrypted value cleared from memory after command completes
- No credential caching in logs or persistent state
- Credentials marked with expiry/rotation policy (future: automatic rotation)
- Audit log: log credential access attempts (not the values)
- Secrets not echoed in command output (gRPC layer sanitizes)

**Residual Risk**: 
- Memory dump of running process can retrieve decrypted credentials temporarily in memory
- Malicious subprocess with access to environment variables can read tokens
- Mitigated by: restricted subprocess permissions, in-process credential cleanup, memory-locking (future)

#### 4. Cross-User Credential Leakage

**Threat**: User A's stored credentials become visible to User B in the same container.

**Attack Vectors:**
- Filesystem race condition in session directory cleanup
- Shared environment variables or process state
- Credential store world-readable (permissioning bug)
- Logs containing credentials (unintended logging)

**Impact if Unmitigated:**
- User B can use User A's git tokens to access their repos
- User B can invoke commands as User A

**Mitigations:**
- Credential store `/workspace/{project}/.secure/credentials.enc` has restrictive permissions (0600)
- Session directories created with user-specific umask: 0700 (owner read-write-execute only)
- Environment variables cleared after command execution
- No credential values in logs; log only: `Used credential type=git_token identifier=github-prod`
- Each request validated for user_id at gRPC layer

**Residual Risk**: Low. File permissions and proper cleanup enforce isolation.

#### 5. Denial of Service (DoS)

**Threat**: Attacker exhausts container resources preventing legitimate commands.

**Attack Vectors:**
- Infinite loops consuming CPU
- Memory leak or allocation bomb
- Disk space exhaustion (clone huge repo repeatedly)
- Network exhaustion (parallel requests, connection limits)
- Process fork bomb

**Impact if Unmitigated:**
- Container becomes unresponsive
- Legitimate users' commands blocked
- Host disk/memory pressure

**Mitigations:**
- Per-session resource limits (cgroups):
  - Memory: ~100MB (configurable, kills process if exceeded)
  - CPU: 0.5 cores (configurable)
  - PIDs: 256 (limits fork bombs)
  - Disk: quota on shared `/workspace` volume (50GB default)
- Command timeout: default 30 minutes (configurable)
- Max concurrent sessions per user: configurable (default unlimited, but could be added)
- Rate limiting at HTTP API layer (host-side)
- Connection pooling limits (host daemon manages gRPC connections)

**Residual Risk**: Sophisticated attack might evade cgroup limits, or attack host via rate-limiting. Mitigated by host-level rate limiting and monitoring.

#### 6. State Repo Tampering

**Threat**: Attacker modifies `.synchestra/` state repo inside container to corrupt task state or escalate privileges.

**Attack Vectors:**
- Malicious command that modifies task state
- Direct git operations: `git reset`, rebase, force-push
- Write to `.synchestra/state.json` directly

**Impact if Unmitigated:**
- Task state inconsistency across team
- False completion reports (task marked done when not)
- Potential for privilege escalation if state repo used for authorization

**Mitigations:**
- `.synchestra/` is git-backed; all mutations are auditable (commit history)
- If host maintains copy of state repo, conflicts can be detected during sync (future)
- gRPC service validates task operations (e.g., cannot transition invalid states)
- Signed commits (future): require GPG signing for task state mutations
- Read-only `.synchestra/` for most operations; only gRPC `UpdateTask` modifies state

**Residual Risk**: Attacker inside container can modify state repo. Mitigated by git audit trail and read-only defaults.

#### 7. Network Interception

**Threat**: Attacker on network intercepts gRPC messages between host and container.

**Attack Vectors:**
- Man-in-the-middle (MITM) on Docker bridge network
- Network sniffing if gRPC uses TCP instead of Unix socket

**Impact if Unmitigated:**
- Credential values, commands, or task state leaked in plaintext
- Attacker can inject malicious commands

**Mitigations:**
- **Recommended**: Unix socket communication (`/var/run/synchestra-{project_id}.sock`)
  - No network exposure; filesystem permissions only
  - Attacker must have local file access (already inside container or host)
- **Fallback**: TCP with TLS/mTLS encryption (if needed for remote orchestration)
- All credential RPC calls encrypted at transport layer

**Residual Risk**: Minimal if using Unix socket. If using TCP, TLS mitigates plaintext leakage.

### Threat Matrix (Severity vs. Likelihood)

| Threat | Severity | Likelihood | Status |
|--------|----------|-----------|--------|
| Host compromise leaks secrets | Critical | Low | Mitigated (keys in container) |
| Container escape | Critical | Low | Mitigated (hardening, capabilities) |
| Credential theft from container | High | Medium | Mitigated (encryption, cleanup) |
| Cross-user credential leakage | High | Low | Mitigated (permissions) |
| Denial of Service | Medium | Medium | Mitigated (resource limits) |
| State repo tampering | Medium | Low | Mitigated (git audit trail) |
| Network interception | High | Low | Mitigated (Unix socket) |

## Security Mechanisms

### 1. Credential Encryption

**Algorithm**: AES256-GCM (authenticated encryption)

**Key Management:**
- Generated per container at startup: `openssl rand 32 > /secure/encryption.key`
- Stored in container memory (encrypted at rest using container secrets if available)
- Never transmitted to host or other containers
- Optional: Store in container image secrets (Docker Secrets, Kubernetes Secrets) for cluster deployments

**Encryption Flow:**
```
User provides token → gRPC StoreCredential() → 
  Container: AES256-GCM(token, key, nonce) → 
  Store ciphertext in /workspace/{project}/.secure/credentials.enc →
  Response: { success: true }
```

**Decryption Flow (On-Demand):**
```
Command: git clone https://github.com/private/repo →
  Container: Load encrypted credentials → 
  AES256-GCM-decrypt(ciphertext, key, nonce) → 
  Inject into git credential helper or environment →
  Execute command →
  Clear decrypted value from memory
```

**Credential Storage Format** (encrypted JSON):
```json
{
  "credentials": [
    {
      "type": "git_token",
      "identifier": "github-prod",
      "encrypted_value": "base64(AES256-GCM(token, key))",
      "nonce": "hex(nonce)",
      "created_at": "2026-03-15T11:00:00Z",
      "expires_at": null
    }
  ]
}
```

### 2. Container Hardening

**Docker Run Flags:**
```bash
docker run \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  --read-only \
  --tmpfs /run:noexec,nosuid \
  --tmpfs /tmp:noexec,nosuid \
  --volume /workspace:/workspace:rw \
  --user 1000:1000 \
  --security-opt=seccomp=profile.json \
  --memory=512mb \
  --cpus=2 \
  --pids-limit=256 \
  sandbox-image:latest
```

**Seccomp Profile** (`profile.json`):
- Allow: `read`, `write`, `open`, `close`, `stat`, `clone`, `fork`, `exec*`, `socket`, `connect`, `bind`
- Deny: `ptrace`, `process_vm_*`, `mount`, `umount`, `sysctl`, `reboot`, `load_modules`
- Full profile maintained in implementation repo

**Filesystem Permissions:**
```
/workspace/{project}/.secure/ — mode 0700 (root only, or container uid)
/workspace/{project}/.secure/credentials.enc — mode 0600
/workspace/{project}/sessions/{session_id}/ — mode 0700 (per-session cleanup)
/workspace/{project}/.synchestra/ — mode 0755 (readable by processes)
```

**Unprivileged User:**
- Container runs as UID 1000 (unprivileged)
- Cannot modify system files or host
- Workspace directories owned by UID 1000

### 3. User Isolation

**Session Directory Isolation:**
- Path: `/workspace/{project_id}/sessions/{session_id}/`
- Permissions: `0700` (owner only)
- Owner: Derived from `user_id` in gRPC request (optional: create OS user per user)
- Cleanup: Delete on session completion or timeout

**Credential Isolation:**
- Stored in shared `/workspace/{project}/.secure/credentials.enc`
- gRPC layer validates `user_id` before granting credential access
- Future: Per-user credential encryption with separate keys

**Process Isolation:**
- Commands run in separate processes
- No shared environment between sessions
- cgroups enforce memory/CPU per session (if possible via isolation)

**Log Isolation:**
- Stdout/stderr captured per session: `/workspace/{project}/sessions/{session_id}/logs/`
- No cross-session log access
- Logs cleaned up with session

### 4. Resource Limits (cgroups)

**Per-Session Limits:**
```
Memory:    100 MB (kill process if exceeded)
CPU:       0.5 cores (throttle)
PIDs:      256 (fork limit)
Disk:      (enforced at volume level, not per-session)
```

**Per-Project Volume Quota:**
```
Disk:      50 GB (default)
         (configurable; prevents one project starving others)
```

### 5. gRPC Security

**Authentication:**
- Each request includes `user_id` and `project_id`
- Host validates user has access to project (from `sandbox_user_project_access` table)
- Container can optionally validate `user_id` in request headers

**Transport:**
- Unix socket: No TLS needed (filesystem permissions)
- TCP (if used): Mandatory TLS 1.3 with client certificates (mTLS)

**Secrets in Messages:**
- Credential values sent only in `StoreCredential` RPC
- Never logged or returned in response
- Marked sensitive in protobuf schema (future: code generation to enforce no-logging)

## Validation & Testing

### Security Testing Checklist

- [ ] **Credential Encryption**: Verify encrypted credentials cannot be read without decryption key
- [ ] **Key Isolation**: Verify decryption key never exported from container
- [ ] **Container Hardening**: Run container and verify capabilities drop, seccomp active, UID non-root
- [ ] **User Isolation**: Two users execute commands; verify session directories separate, credentials not shared
- [ ] **Resource Limits**: Execute memory bomb; verify process killed at 100MB limit
- [ ] **Command Timeout**: Execute infinite loop; verify killed after timeout
- [ ] **State Repo**: Verify `.synchestra/` is read-only for non-gRPC operations
- [ ] **Cross-Project Isolation**: Two projects in separate containers; verify no file/network access between them
- [ ] **Audit Logging**: Credential access attempt logged (not value); verify in logs
- [ ] **Permissions**: Verify `/.secure/credentials.enc` is 0600, sessions/ are 0700
- [ ] **Network**: Verify no external network access from container (except authorized egress)

### Penetration Testing

**In-Scope:**
- Attempt to read credentials from container filesystem (should be encrypted)
- Attempt to extract decryption key from memory (should fail; key is in-process only)
- Attempt cross-user credential access (should be denied)
- Attempt to escape container and access host (should fail)
- Attempt DoS via resource exhaustion (should be limited)

**Out-of-Scope (Host Hardening):**
- Host OS vulnerabilities
- Docker daemon vulnerabilities (zero-day)
- Kubernetes cluster security (if deployed on K8s)

## Future Enhancements

### Phase 2: Advanced Security

1. **Hardware Security Module (HSM)**
   - Store decryption keys in HSM, not container memory
   - Requires HSM client library in container image
   - Benefits: Keys never in software, resistant to memory dumps

2. **Signed Commits**
   - State repo mutations require GPG signature
   - Prevents unauthorized state changes
   - Requires GPG key storage (similar to credentials)

3. **Audit Trail with Signing**
   - Log all credential access, command execution
   - Sign audit logs to prevent tampering
   - Immutable audit store (e.g., append-only S3)

4. **Automated Credential Rotation**
   - Periodic expiry: credentials become invalid after N days
   - Automatic re-issue from vault
   - User notified of expiry, must renew

5. **Per-User Credential Encryption**
   - Each user has own encryption key for their credentials
   - Container acts as key distributor (authenticated per-user)
   - User A cannot access User B's credentials even with container escape

6. **Network Encryption (if Remote)**
   - mTLS for TCP-based gRPC (if not using Unix socket)
   - Certificate rotation
   - Certificate pinning

7. **Secrets Management Integration**
   - HashiCorp Vault: Container queries vault for credentials on-demand
   - AWS Secrets Manager: Container uses IAM role to fetch secrets
   - Benefits: Centralized rotation, audit trail, no local storage

## Compliance & Standards

### Standards Applicability

- **OWASP Top 10**: Addressed secret management (#2 Broken Auth), sensitive data exposure (#3)
- **CWE-798 (Hardcoded Credentials)**: Mitigated via encrypted storage and gRPC injection
- **CWE-434 (Unrestricted File Upload)**: N/A (not accepting uploads)
- **CWE-95 (Code Injection)**: Mitigated via subprocess isolation, not shell=True

### SOC 2 / ISO 27001 Readiness

- [ ] Encryption at rest: AES256
- [ ] Encryption in transit: TLS/Unix socket
- [ ] Access control: User↔project validation
- [ ] Audit logging: Credential access, command execution
- [ ] Incident response: Container isolation limits blast radius
- [ ] Penetration testing: Security testing checklist above
- [ ] Vulnerability scanning: Container image scans (Trivy, Snyk)

## Outstanding Questions

1. **HSM integration**: Should we support external HSM for key storage? Which platforms (AWS CloudHSM, Azure Key Vault, etc.)?
2. **Key rotation**: How often should decryption keys rotate? Should old keys be retained for decrypting old credentials?
3. **Audit retention**: How long should audit logs be retained? Where should they be stored (local, remote)?
4. **Compliance scope**: Are there specific compliance requirements (HIPAA, PCI-DSS, SOC 2)? These affect credential handling and audit trails.
5. **Secrets sharing across projects**: Should User A in Project A be able to store a secret that User B in Project B can access? Requires cross-project credential store.
6. **Credential revocation**: If a credential is compromised, can it be revoked immediately? Or only on next container restart?
7. **Supply chain security**: Should container image be signed (Docker Content Trust)? How to verify authenticity?
8. **Ephemeral containers**: Should containers be destroyed after idle timeout to reduce credential exposure window?
