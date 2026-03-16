# Credential Management

## Overview

Container-managed encrypted credential vault for the Synchestra sandbox. The host is **stateless** — it never sees, stores, or decrypts credentials. Each container autonomously manages its own vault using AES256-GCM encryption with a per-container persistent key.

This document is the **authoritative, consolidated** credential management reference. For gRPC message definitions and RPC behavior, see [protocol.md](protocol.md). For Go implementation patterns (structs, functions, tests), see [agent-implementation-guide.md](agent-implementation-guide.md).

**Related specs:**

- [protocol.md](protocol.md) — `StoreCredential` / `GetCredential` RPCs, message definitions, behavior
- [agent-implementation-guide.md](agent-implementation-guide.md) — `CredentialVault` interface, AES256-GCM Go implementation
- [agent.proto](agent.proto) — Protobuf message schemas
- [orchestrator.md](orchestrator.md) — Host statelessness design principle

## Design Principles

1. **Host isolation** — Credentials flow through the host as opaque gRPC messages. The host has no decryption capability and never persists credential values. Unix socket transport means credentials never traverse a network.
2. **Container autonomy** — Each container manages its own vault. No shared credential store across projects. A compromised container exposes only that project's credentials.
3. **Encrypt at rest** — All credentials are encrypted with AES256-GCM before writing to disk. Plaintext never touches the filesystem.
4. **Decrypt on demand** — Credentials are decrypted only when needed for command execution, then cleared from memory immediately after use.
5. **Audit everything** — Every credential operation (store, retrieve, delete, use) is logged with actor and timestamp. Credential values are **never** logged.

## Credential Types

| Type | Identifier Pattern | Usage | Example |
|------|-------------------|-------|---------|
| `git_token` | `github-{name}` | Git HTTPS authentication | GitHub PAT for cloning private repos |
| `ssh_key` | `ssh-{name}` | SSH authentication | Deploy key for git over SSH |
| `api_key` | `api-{service}-{name}` | External service authentication | OpenAI API key, AWS access key |
| `env_secret` | `env-{name}` | Injected as environment variable during command execution | Database connection string |
| `custom` | `{name}` | User-defined | Any secret value |

Credential types are **extensible**. The `credential_type` field in the protobuf schema (see [agent.proto](agent.proto)) is a free-form string. The types above are conventions, not an exhaustive enum.

## Encryption Architecture

### Key Management

| Property | Value |
|----------|-------|
| **Algorithm** | AES-256 (256-bit key) |
| **Mode** | GCM (Galois/Counter Mode) — authenticated encryption |
| **Key size** | 32 bytes |
| **Nonce size** | 12 bytes (GCM standard) |
| **Key generation** | `crypto/rand.Read(key)` or equivalently `openssl rand -base64 32` |
| **Key path** | `/workspace/{project_id}/.secure/encryption.key` |
| **Key permissions** | `0400` (owner-read only) |
| **Key isolation** | Exists only inside the container volume. Never exported to host. |

**Lifecycle:**

- **First start**: Key is generated and persisted to disk.
- **Restart**: Key is loaded from disk, enabling decryption of previously stored credentials.
- **Rotation**: Manual trigger only — see [Key Rotation](#key-rotation).

### Encryption Process (Store)

1. Serialize credential to JSON:
   ```json
   {"type":"git_token","value":"ghp_xxx...","description":"...","expires_at":1234567890}
   ```
2. Generate 12-byte random nonce: `crypto/rand.Read(nonce)`
3. Create AES-256 block cipher: `aes.NewCipher(key)`
4. Create GCM: `cipher.NewGCM(block)`
5. Encrypt with AAD: `gcm.Seal(nil, nonce, plaintext, []byte(identifier))`
   — the credential identifier is passed as additional authenticated data
6. Store: `base64(nonce || ciphertext)` as a single string in the vault file
7. Zero plaintext from memory

### Decryption Process (Get)

1. Load encrypted entry from vault file by identifier
2. Decode base64 → extract nonce (first 12 bytes) and ciphertext (remainder)
3. Create AES-256 block cipher and GCM
4. Decrypt with AAD: `gcm.Open(nil, nonce, ciphertext, []byte(identifier))`
5. Deserialize JSON → return credential
6. Zero plaintext from memory after caller is done (`defer zeroize()`)

### Why AAD (Additional Authenticated Data)?

Using the credential identifier as GCM authenticated additional data prevents:

- **Credential swapping** — An attacker cannot copy encrypted data from one identifier to another. Decryption will fail because the AAD won't match.
- **Tampering** — Any modification to the ciphertext or identifier causes authenticated decryption failure.
- **Replay** — Encrypted data is cryptographically bound to its identifier.

## Vault File Format

The vault is a single JSON file with per-entry encryption:

```json
{
  "version": 1,
  "entries": {
    "github-prod": {
      "ciphertext": "base64(nonce || encrypted_json)",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    },
    "ssh-deploy": {
      "ciphertext": "base64(nonce || encrypted_json)",
      "created_at": "2024-01-15T11:00:00Z",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  }
}
```

| Property | Value |
|----------|-------|
| **File path** | `/workspace/{project_id}/.secure/credentials.enc` |
| **File permissions** | `0600` (owner read/write) |
| **Directory permissions** | `0700` (owner only) |
| **Concurrency** | `flock()` for file-level locking across parallel sessions |

Each entry is independently encrypted — updating one credential does not require re-encrypting the others.

## Credential Lifecycle

### Store (Create/Update)

1. HTTP API receives credential from user (HTTPS).
2. Host forwards via gRPC `StoreCredential` to container (Unix socket — no network).
3. Container agent encrypts credential value with AES256-GCM.
4. Writes to vault file. Upsert semantics — overwrites if identifier exists.
5. Audit log: `credential.stored {identifier} {type} {user_id}` (never the value).
6. Returns `StoreCredentialResponse` with success confirmation.

> See [protocol.md](protocol.md) for `StoreCredentialRequest` / `StoreCredentialResponse` message definitions and RPC behavior.

### Retrieve (for Command Execution)

1. Command needs credential (e.g., `git clone` with auth).
2. Agent looks up credential by identifier in vault.
3. Decrypts on demand.
4. Injects into command environment (env var) or writes to temp file (SSH key).
5. Command executes.
6. Agent clears decrypted value from memory.
7. Temp files (e.g., SSH key file) are removed immediately after command.
8. Audit log: `credential.accessed {identifier} {type} {session_id}`.

> See [protocol.md](protocol.md) for `GetCredentialRequest` / `GetCredentialResponse` message definitions.

### Delete

1. User requests credential deletion via API.
2. Container removes entry from vault file.
3. File rewritten without the entry.
4. Audit log: `credential.deleted {identifier} {type} {user_id}`.

### Expiry

- Optional `expires_at` timestamp per credential (Unix timestamp, `0` = no expiry).
- On `GetCredential`: if expired, return `found=true, expired=true, credential_value=""`.
- On command execution: check expiry before injection. If expired, fail the command with a clear error message.
- No automatic cleanup — expired credentials remain in the vault until explicitly deleted or overwritten.

## Credential Injection for Commands

When a command needs credentials, the agent injects them securely based on credential type.

### Git HTTPS Tokens (`git_token`)

```bash
# Injected via GIT_ASKPASS helper script
export GIT_ASKPASS=/tmp/synchestra-git-askpass-{session_id}
# The askpass script echoes the decrypted token, then self-deletes
```

The askpass script is created with mode `0700`, writes the token to stdout on invocation, and removes itself after first use.

### SSH Keys (`ssh_key`)

```bash
# Written to temp file with restrictive permissions
SSH_KEY_FILE=/tmp/synchestra-ssh-{session_id}-{identifier}
chmod 0600 ${SSH_KEY_FILE}
export GIT_SSH_COMMAND="ssh -i ${SSH_KEY_FILE} -o StrictHostKeyChecking=accept-new"
# Removed immediately after command completes
```

### Environment Variables (`env_secret`)

```go
// Injected directly into the command environment
cmd.Env = append(cmd.Env, "DATABASE_URL=decrypted_value")
// Value cleared from memory after cmd.Start()
```

### API Keys (`api_key`)

```bash
# Injected as environment variable with conventional name
export OPENAI_API_KEY=decrypted_value
# Or written to config file if the service requires it
```

The environment variable name is derived from the credential identifier by convention. For example, `api-openai-prod` maps to `OPENAI_API_KEY`. Custom mappings can be specified at store time via the `description` field.

## Key Rotation

### Trigger

- Manual only: admin endpoint or CLI command.
- Future: automatic rotation on schedule.

### Process

1. Generate new encryption key: `crypto/rand.Read(newKey)` (32 bytes).
2. Load all credentials from vault (decrypt with old key).
3. Re-encrypt all credentials with new key (new nonces generated per entry).
4. Atomic write: write new vault file to temp path, then `os.Rename()` to final path.
5. Replace old key file with new key (atomic rename).
6. Zero old key from memory.
7. Audit log: `credential.key_rotated {project_id} {credential_count}`.

### Failure Handling

- If rotation fails mid-way: old vault file is unchanged (atomic write protects).
- If new key file write fails: old key still valid, retry rotation.
- The old vault + old key remain consistent at every point in the process.

## Credential Scoping

### Project-Wide Credentials (Current)

All credentials are project-wide by default. Any user with access to the project can use any credential stored in that project's vault.

Use cases: shared deploy keys, CI tokens, team API keys.

### Per-User Credentials (Future Enhancement)

- Optional `owner_user_id` field on credentials.
- Only the owner can retrieve/use the credential.
- Other users can see the credential identifier (for UI display) but not the value.

> **Note**: Per-user credential isolation is a future enhancement. The initial implementation treats all credentials as project-wide. The audit log captures which user accessed which credential for accountability.

## Security Properties

### What the host CANNOT do

- Read credential values (encrypted at rest, key inside container).
- Decrypt credentials (no access to encryption key).
- Access the credential vault file (container filesystem boundary).
- Intercept credentials in transit (Unix socket, not network).

### What a container compromise exposes

- All credentials for **that one project** (encryption key + vault are co-located).
- **NOT** credentials from other projects (separate containers, separate keys).
- Mitigation: credential expiry limits the exposure window.

### What a volume compromise exposes

- Encrypted vault file + encryption key file (if attacker gains full volume access).
- Mitigation: volume access restricted to container UID 1000 only.
- Future mitigation: external key management (HashiCorp Vault, cloud KMS) separates key from data.

### Transport security

- gRPC over Unix socket — credentials never traverse a network.
- User → host API is HTTPS — encrypted in transit.
- See [protocol.md](protocol.md) for transport details.

## Audit Log

All credential operations are logged without revealing secret values:

```json
{"timestamp":"2024-01-15T10:30:00Z","event":"credential.stored","identifier":"github-prod","type":"git_token","user_id":"user-123","project_id":"proj-456"}
{"timestamp":"2024-01-15T10:31:00Z","event":"credential.accessed","identifier":"github-prod","type":"git_token","session_id":"sess-789","project_id":"proj-456"}
{"timestamp":"2024-01-15T10:35:00Z","event":"credential.deleted","identifier":"github-prod","type":"git_token","user_id":"user-123","project_id":"proj-456"}
{"timestamp":"2024-01-15T12:00:00Z","event":"credential.key_rotated","project_id":"proj-456","credential_count":5}
```

| Property | Value |
|----------|-------|
| **Log path** | `/workspace/{project_id}/.secure/audit.log` |
| **Log permissions** | `0600` (owner read/write) |
| **Log mode** | Append-only |
| **Log rotation** | Rotated when file exceeds 10 MB |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SYNCHESTRA_ENCRYPTION_KEY` | *(auto-generated)* | AES256 encryption key (base64). If empty, generated on first start and persisted. |
| `SYNCHESTRA_CREDENTIAL_MAX_SIZE` | `64KB` | Maximum size of a single credential value |
| `SYNCHESTRA_CREDENTIAL_MAX_COUNT` | `100` | Maximum number of credentials per project |
| `SYNCHESTRA_CREDENTIAL_AUDIT_MAX_SIZE` | `10MB` | Audit log rotation threshold |

## Outstanding Questions

1. Should there be a `ListCredentials` RPC that returns identifiers (not values) for UI display? The `CredentialVault` interface in the implementation guide already defines `List(userID string) ([]CredentialMetadata, error)` but there is no corresponding RPC in [agent.proto](agent.proto).
2. Should credential expiry trigger a notification/event via the event bus?
3. Should the encryption key be derivable from a user-provided passphrase (PBKDF2/Argon2) for additional security?
4. Should `DeleteCredential` be added as an explicit RPC? Currently deletion is described in lifecycle but has no protobuf definition.
5. What is the convention for mapping `api_key` identifiers to environment variable names? Should this be explicit metadata on the credential, or derived from the identifier pattern?
