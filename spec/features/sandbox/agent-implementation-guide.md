# Sandbox gRPC Agent Implementation Guide

## Overview

The gRPC Agent is a Go service that runs inside each Synchestra sandbox container. It listens on a Unix socket and implements the `SandboxAgent` service defined in `agent.proto`.

**Language**: Go  
**Location in repo**: `internal/sandbox/agent/`  
**Container entry point**: `/app/synchestra-sandbox-agent`

## Architecture

### Components

```
SandboxAgent Service
├── ExecuteCommandHandler
│   ├── SessionManager (create, clean up isolation)
│   ├── SubprocessExecutor (exec with timeouts, resource limits)
│   └── OutputStreamer (stdout/stderr capture, real-time streaming)
├── CredentialManager
│   ├── Encryption (AES256-GCM)
│   ├── LocalVault (file-based encrypted store)
│   └── DecryptionCache (short-lived cache for performance)
├── TaskStateManager
│   ├── GitRepo (local .synchestra/ operations)
│   └── StateReader (JSON parsing)
├── HealthChecker
│   ├── ResourceMonitor (memory, CPU, disk)
│   └── CircuitBreaker (graceful degradation)
└── GRPCServer
    ├── Unix socket listener
    ├── ConnectionPool
    └── ErrorHandler
```

### Key Interfaces

```go
// SessionManager creates and tracks isolated execution environments
type SessionManager interface {
    CreateSession(ctx context.Context, sessionID, userID string) (*Session, error)
    GetSession(sessionID string) (*Session, error)
    ListSessions(userID string) ([]*Session, error)
    CleanupSession(sessionID string, retentionHours int) error
    GetSessionResources(sessionID string) (*SessionResources, error)
}

// CredentialVault stores and retrieves encrypted credentials
type CredentialVault interface {
    Store(ctx context.Context, identifier string, credType string, value string, expiresAt time.Time) error
    Get(ctx context.Context, userID string, identifier string) (string, error)
    Delete(identifier string) error
    List(userID string) ([]CredentialMetadata, error)
}

// SubprocessExecutor runs commands with isolation and resource limits
type SubprocessExecutor interface {
    Execute(ctx context.Context, req *CommandRequest, session *Session) (*CommandExecution, error)
}

// TaskStateReader queries the local .synchestra/ git repo
type TaskStateReader interface {
    GetState(ctx context.Context, queryPath, gitRef string) ([]byte, string, error)
}
```

## Implementation Details

### 1. Session Isolation

**Directory Structure:**
```
/workspace/{project_id}/sessions/{session_id}/
├── working/                    # Command execution directory
├── logs/
│   ├── stdout.log
│   ├── stderr.log
│   └── metadata.json           # Session metadata
├── .cgroup                     # cgroup v2 configuration (optional)
└── .env                        # Session environment
```

**Session Creation:**
```go
func (s *SessionManager) CreateSession(ctx context.Context, sessionID, userID string) (*Session, error) {
    sessionPath := filepath.Join(workspaceRoot, sessionID)
    
    // Create directory structure with secure permissions
    if err := os.MkdirAll(filepath.Join(sessionPath, "working"), 0700); err != nil {
        return nil, fmt.Errorf("create session dir: %w", err)
    }
    
    // Create cgroup for resource limits (if cgroup v2 available)
    cgroup, err := s.createCgroup(sessionID, memoryLimitMB, cpuLimit)
    if err != nil {
        log.Warnf("cgroup creation failed (non-fatal): %v", err)
    }
    
    // Record session metadata
    metadata := &SessionMetadata{
        SessionID:  sessionID,
        UserID:     userID,
        CreatedAt:  time.Now(),
        Status:     "pending",
        MemoryUsed: 0,
        PID:        0,
    }
    
    if err := recordMetadata(sessionPath, metadata); err != nil {
        return nil, fmt.Errorf("record metadata: %w", err)
    }
    
    return &Session{
        ID:        sessionID,
        UserID:    userID,
        Path:      sessionPath,
        Cgroup:    cgroup,
        Metadata:  metadata,
        CreatedAt: time.Now(),
    }, nil
}
```

### 2. Command Execution

**Subprocess Isolation:**
```go
func (e *SubprocessExecutor) Execute(ctx context.Context, req *CommandRequest, session *Session) (*CommandExecution, error) {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
    defer cancel()
    
    // Prepare command
    cmd := exec.CommandContext(ctx, req.Command[0], req.Command[1:]...)
    
    // Set working directory (validate against path traversal)
    workingDir := filepath.Join(session.Path, "working", req.WorkingDir)
    workingDir, err := filepath.EvalSymlinks(workingDir)
    if err != nil || !strings.HasPrefix(workingDir, session.Path) {
        return nil, fmt.Errorf("path traversal detected: %s", req.WorkingDir)
    }
    cmd.Dir = workingDir
    
    // Set environment (merge inherited + provided)
    cmd.Env = mergeEnvironment(os.Environ(), req.EnvVars, session.ID, session.UserID)
    
    // Capture output (with buffering for streaming)
    stdout := NewTeeWriter(e.newStreamWriter("stdout", session.ID), 10*1024*1024) // 10MB buffer
    stderr := NewTeeWriter(e.newStreamWriter("stderr", session.ID), 10*1024*1024)
    cmd.Stdout = stdout
    cmd.Stderr = stderr
    
    // Set process credentials (run as container unprivileged user)
    if e.unprivilegedUID > 0 {
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Credential: &syscall.Credential{Uid: e.unprivilegedUID, Gid: e.unprivilegedGID},
        }
    }
    
    // Add to cgroup for resource limits
    if session.Cgroup != nil {
        if err := session.Cgroup.AddProcess(cmd.Process); err != nil {
            log.Warnf("cgroup add process: %v", err)
        }
    }
    
    // Start process
    startTime := time.Now()
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("start process: %w", err)
    }
    
    // Wait for completion (with context timeout)
    go func() {
        <-ctx.Done()
        if cmd.Process != nil {
            log.Debugf("session %s timeout, sending SIGTERM", session.ID)
            cmd.Process.Signal(syscall.SIGTERM)
            
            // Force SIGKILL after 5 seconds
            time.Sleep(5 * time.Second)
            if cmd.ProcessState == nil {
                log.Debugf("session %s still running, sending SIGKILL", session.ID)
                cmd.Process.Signal(syscall.SIGKILL)
            }
        }
    }()
    
    // Record start in metadata
    session.Metadata.PID = cmd.Process.Pid
    session.Metadata.Status = "running"
    recordMetadata(session.Path, session.Metadata)
    
    // Wait for process
    err = cmd.Wait()
    exitCode := getExitCode(cmd, err)
    runtimeSeconds := int64(time.Since(startTime).Seconds())
    
    // Record completion
    session.Metadata.Status = "completed"
    session.Metadata.ExitCode = int32(exitCode)
    session.Metadata.RuntimeSeconds = runtimeSeconds
    recordMetadata(session.Path, session.Metadata)
    
    return &CommandExecution{
        Session:    session,
        ExitCode:   exitCode,
        Runtime:    runtimeSeconds,
        Stdout:     stdout.Bytes(),
        Stderr:     stderr.Bytes(),
        CompletedAt: time.Now(),
    }, nil
}

func getExitCode(cmd *exec.Cmd, err error) int {
    if cmd.ProcessState == nil {
        return -1
    }
    return cmd.ProcessState.ExitCode()
}
```

### 3. Credential Encryption & Storage

**AES256-GCM Encryption:**
```go
type CredentialVault struct {
    encryptionKey []byte          // 32 bytes for AES256
    storagePath   string          // /workspace/{project}/.secure/credentials.enc
    auditLog      *AuditLog
}

func (v *CredentialVault) Store(ctx context.Context, identifier, credType, value string, expiresAt time.Time) error {
    // Load or initialize vault
    vault, err := v.loadVault()
    if err != nil && !errors.Is(err, os.ErrNotExist) {
        return fmt.Errorf("load vault: %w", err)
    }
    if vault == nil {
        vault = &CredentialStore{Credentials: map[string]*Credential{}}
    }
    
    // Check if identifier already exists
    if _, exists := vault.Credentials[identifier]; exists {
        return fmt.Errorf("credential %q already exists", identifier)
    }
    
    // Generate nonce for this credential
    nonce := make([]byte, 12) // GCM standard nonce size
    if _, err := rand.Read(nonce); err != nil {
        return fmt.Errorf("generate nonce: %w", err)
    }
    
    // Encrypt credential value
    cipher, err := aes.NewCipher(v.encryptionKey)
    if err != nil {
        return fmt.Errorf("create cipher: %w", err)
    }
    
    gcm, err := cipher.NewGCM()
    if err != nil {
        return fmt.Errorf("create GCM: %w", err)
    }
    
    ciphertext := gcm.Seal(nil, nonce, []byte(value), []byte(identifier))
    
    // Store in vault (in-memory)
    vault.Credentials[identifier] = &Credential{
        Type:        credType,
        Identifier:  identifier,
        Ciphertext:  base64.StdEncoding.EncodeToString(ciphertext),
        Nonce:       base64.StdEncoding.EncodeToString(nonce),
        CreatedAt:   time.Now().Unix(),
        ExpiresAt:   expiresAt.Unix(),
        AccessLog:   []AccessRecord{},
    }
    
    // Persist vault to disk
    if err := v.saveVault(vault); err != nil {
        return fmt.Errorf("save vault: %w", err)
    }
    
    // Audit log
    v.auditLog.Log(AuditEvent{
        Event:      "credential_stored",
        Identifier: identifier,
        Type:       credType,
        Timestamp:  time.Now(),
    })
    
    return nil
}

func (v *CredentialVault) Get(ctx context.Context, userID, identifier string) (string, error) {
    // Load vault
    vault, err := v.loadVault()
    if err != nil {
        return "", fmt.Errorf("load vault: %w", err)
    }
    
    // Find credential
    cred, exists := vault.Credentials[identifier]
    if !exists {
        return "", fmt.Errorf("credential %q not found", identifier)
    }
    
    // Check expiry
    if cred.ExpiresAt > 0 && time.Now().Unix() > cred.ExpiresAt {
        return "", fmt.Errorf("credential %q expired", identifier)
    }
    
    // Decrypt
    ciphertext, err := base64.StdEncoding.DecodeString(cred.Ciphertext)
    if err != nil {
        return "", fmt.Errorf("decode ciphertext: %w", err)
    }
    
    nonce, err := base64.StdEncoding.DecodeString(cred.Nonce)
    if err != nil {
        return "", fmt.Errorf("decode nonce: %w", err)
    }
    
    cipher, err := aes.NewCipher(v.encryptionKey)
    if err != nil {
        return "", fmt.Errorf("create cipher: %w", err)
    }
    
    gcm, err := cipher.NewGCM()
    if err != nil {
        return "", fmt.Errorf("create GCM: %w", err)
    }
    
    plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(identifier))
    if err != nil {
        return "", fmt.Errorf("decrypt: %w", err)
    }
    
    // Record access (not the value)
    cred.AccessLog = append(cred.AccessLog, AccessRecord{
        UserID:    userID,
        Timestamp: time.Now(),
    })
    
    // Audit log (not the decrypted value)
    v.auditLog.Log(AuditEvent{
        Event:      "credential_accessed",
        UserID:     userID,
        Identifier: identifier,
        Timestamp:  time.Now(),
    })
    
    // Clear plaintext from memory after return
    defer func() {
        for i := range plaintext {
            plaintext[i] = 0
        }
    }()
    
    return string(plaintext), nil
}
```

### 4. Output Streaming

**Real-Time Output with Buffering:**
```go
type StreamWriter struct {
    sessionID     string
    stream        string  // "stdout" or "stderr"
    ch            chan []byte
    sendFn        func([]byte) error
    buffer        []byte
    bufferMaxSize int
}

func (w *StreamWriter) Write(p []byte) (int, error) {
    w.buffer = append(w.buffer, p...)
    
    // Flush buffer if too large or contains complete line
    if len(w.buffer) >= w.bufferMaxSize || bytes.Contains(w.buffer, []byte{'\n'}) {
        if err := w.flush(); err != nil {
            return 0, err
        }
    }
    
    return len(p), nil
}

func (w *StreamWriter) flush() error {
    if len(w.buffer) == 0 {
        return nil
    }
    
    select {
    case w.ch <- w.buffer:
        w.buffer = nil
        return nil
    default:
        // Channel full; send to gRPC stream directly
        if err := w.sendFn(w.buffer); err != nil {
            return err
        }
        w.buffer = nil
        return nil
    }
}

// In ExecuteCommandHandler, stream output to client
func (s *SandboxAgentServer) ExecuteCommand(req *CommandRequest, stream SandboxAgent_ExecuteCommandServer) error {
    session, err := s.sessionMgr.CreateSession(stream.Context(), req.SessionId, req.UserId)
    if err != nil {
        return status.Error(codes.Internal, err.Error())
    }
    defer s.sessionMgr.CleanupSession(req.SessionId, retentionHours)
    
    exec, err := s.executor.Execute(stream.Context(), req, session)
    if err != nil {
        return status.Error(codes.Internal, err.Error())
    }
    
    // Stream output chunks
    if req.StreamOutput {
        for chunk := range exec.OutputChannel {
            if err := stream.Send(&CommandOutput{
                SessionId: req.SessionId,
                Stdout:    chunk.Stdout,
                Stderr:    chunk.Stderr,
                Completed: false,
            }); err != nil {
                return err
            }
        }
    }
    
    // Final message with exit code
    return stream.Send(&CommandOutput{
        SessionId:    req.SessionId,
        ExitCode:     int32(exec.ExitCode),
        Completed:    true,
        ErrorMessage: exec.ErrorMessage,
    })
}
```

### 5. Container Startup & Initialization

**Main Entry Point:**
```go
func main() {
    // Parse environment
    projectID := os.Getenv("SYNCHESTRA_PROJECT_ID")
    stateRepoURL := os.Getenv("SYNCHESTRA_STATE_REPO_URL")
    socketPath := fmt.Sprintf("/var/run/synchestra-%s.sock", projectID)
    
    // Initialize encryption key (in production, use secrets manager)
    encryptionKey := make([]byte, 32)
    if _, err := rand.Read(encryptionKey); err != nil {
        log.Fatalf("generate encryption key: %v", err)
    }
    
    // Initialize components
    sessionMgr := NewSessionManager("/workspace/" + projectID)
    credentialVault := NewCredentialVault(encryptionKey, "/workspace/"+projectID+"/.secure/credentials.enc")
    taskStateReader := NewTaskStateReader("/workspace/"+projectID+"/.synchestra")
    executor := NewSubprocessExecutor()
    
    // Clone state repository
    if err := cloneStateRepo(stateRepoURL, "/workspace/"+projectID+"/.synchestra"); err != nil {
        log.Fatalf("clone state repo: %v", err)
    }
    
    // Create gRPC server
    server := grpc.NewServer()
    agent := &SandboxAgentServer{
        projectID:        projectID,
        sessionMgr:       sessionMgr,
        credentialVault:  credentialVault,
        taskStateReader:  taskStateReader,
        executor:         executor,
    }
    pb.RegisterSandboxAgentServer(server, agent)
    
    // Listen on Unix socket
    lis, err := net.Listen("unix", socketPath)
    if err != nil {
        log.Fatalf("listen on socket: %v", err)
    }
    defer lis.Close()
    
    // Set socket permissions (owner read-write only)
    if err := os.Chmod(socketPath, 0600); err != nil {
        log.Fatalf("chmod socket: %v", err)
    }
    
    log.Infof("SandboxAgent listening on %s", socketPath)
    if err := server.Serve(lis); err != nil {
        log.Fatalf("server error: %v", err)
    }
}

func cloneStateRepo(repoURL, targetPath string) error {
    cmd := exec.Command("git", "clone", repoURL, targetPath)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("clone failed: %w", err)
    }
    
    // Verify clone succeeded
    if _, err := os.Stat(filepath.Join(targetPath, ".git")); err != nil {
        return fmt.Errorf("verify clone: %w", err)
    }
    
    return nil
}
```

### 6. Error Handling & Logging

**Comprehensive Error Handling:**
```go
func toGRPCError(err error) error {
    switch {
    case errors.Is(err, ErrPathTraversal):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, ErrCredentialNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, ErrCredentialExpired):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, ErrSessionLimitReached):
        return status.Error(codes.ResourceExhausted, err.Error())
    case errors.Is(err, ErrDiskQuotaExceeded):
        return status.Error(codes.ResourceExhausted, err.Error())
    case errors.Is(err, context.DeadlineExceeded):
        return status.Error(codes.DeadlineExceeded, "command timeout")
    default:
        return status.Error(codes.Internal, "internal server error")
    }
}
```

## Testing

### Unit Tests

```go
// TestSessionIsolation verifies each session gets unique directory
func TestSessionIsolation(t *testing.T) {
    mgr := NewSessionManager(t.TempDir())
    
    s1, err := mgr.CreateSession(context.Background(), "sess1", "user1")
    assert.NoError(t, err)
    
    s2, err := mgr.CreateSession(context.Background(), "sess2", "user2")
    assert.NoError(t, err)
    
    // Verify directories are different
    assert.NotEqual(t, s1.Path, s2.Path)
    
    // Verify directories have correct permissions
    info1, _ := os.Stat(s1.Path)
    assert.Equal(t, os.FileMode(0700), info1.Mode().Perm())
}

// TestCredentialEncryption verifies encryption/decryption
func TestCredentialEncryption(t *testing.T) {
    key := make([]byte, 32)
    vault := NewCredentialVault(key, t.TempDir())
    
    err := vault.Store(context.Background(), "github-token", "git_token", "ghp_secrettoken123", time.Time{})
    assert.NoError(t, err)
    
    value, err := vault.Get(context.Background(), "user1", "github-token")
    assert.NoError(t, err)
    assert.Equal(t, "ghp_secrettoken123", value)
}

// TestPathTraversalDetection verifies working_dir validation
func TestPathTraversalDetection(t *testing.T) {
    executor := NewSubprocessExecutor()
    session := &Session{Path: "/workspace/project/sessions/s1"}
    
    req := &CommandRequest{
        WorkingDir: "../../../etc/passwd",
        Command:    []string{"cat", "/etc/passwd"},
    }
    
    _, err := executor.Execute(context.Background(), req, session)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "path traversal")
}
```

## Outstanding Questions

1. Should credential decryption cache temporarily decrypted values for repeated use in same session?
2. Should we support credential rotation (re-encrypt all with new key)?
3. Should cgroup v2 be mandatory or fallback to v1/no limits?
4. How long should session logs be retained after completion?
5. Should we implement process limits (memory per command, not just cgroup limits)?
