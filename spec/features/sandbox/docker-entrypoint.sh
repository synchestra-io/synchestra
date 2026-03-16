#!/bin/bash
# This script should be placed at: build/docker-entrypoint.sh (in repository root)
#
# Container entrypoint that:
# 1. Validates environment
# 2. Initializes workspace
# 3. Clones state repository
# 4. Starts gRPC agent

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

PROJECT_ID="${SYNCHESTRA_PROJECT_ID:?ERROR: SYNCHESTRA_PROJECT_ID not set}"
STATE_REPO_URL="${SYNCHESTRA_STATE_REPO_URL:?ERROR: SYNCHESTRA_STATE_REPO_URL not set}"
WORK_DIR="${SYNCHESTRA_WORK_DIR:-/workspace/${PROJECT_ID}}"
SOCKET_PATH="${SYNCHESTRA_SOCKET_PATH:-/var/run/synchestra-${PROJECT_ID}.sock}"
LOG_LEVEL="${SYNCHESTRA_LOG_LEVEL:-info}"

# Git credentials (optional)
STATE_REPO_TOKEN="${SYNCHESTRA_STATE_REPO_TOKEN:-}"
GIT_USER="${SYNCHESTRA_GIT_USER:-synchestra}"

# Encryption & security
ENCRYPTION_KEY="${SYNCHESTRA_ENCRYPTION_KEY:-}"  # Will be auto-generated if empty

echo "[entrypoint] Initializing Synchestra Sandbox Agent"
echo "[entrypoint] Project ID: ${PROJECT_ID}"
echo "[entrypoint] Work Directory: ${WORK_DIR}"
echo "[entrypoint] State Repo: ${STATE_REPO_URL}"

# ============================================================================
# Validation
# ============================================================================

if [[ ! -d "${WORK_DIR}" ]]; then
    echo "[entrypoint] ERROR: Work directory ${WORK_DIR} does not exist or is not mounted"
    echo "[entrypoint] Ensure host-side volume is properly mounted: -v /path/on/host:${WORK_DIR}"
    exit 1
fi

# Verify directory is writable
if ! touch "${WORK_DIR}/.test" 2>/dev/null; then
    echo "[entrypoint] ERROR: Work directory ${WORK_DIR} is not writable"
    echo "[entrypoint] Ensure volume mount has correct permissions"
    exit 1
fi
rm -f "${WORK_DIR}/.test"

echo "[entrypoint] ✓ Work directory validated"

# ============================================================================
# Directory Structure Setup
# ============================================================================

STATE_REPO_PATH="${WORK_DIR}/.synchestra"
SECURE_PATH="${WORK_DIR}/.secure"
SESSIONS_PATH="${WORK_DIR}/sessions"
REPOS_PATH="${WORK_DIR}/repos"

mkdir -p "${SECURE_PATH}" "${SESSIONS_PATH}" "${REPOS_PATH}"

# Set restrictive permissions on secure directory
chmod 0700 "${SECURE_PATH}"

echo "[entrypoint] ✓ Directory structure created"

# ============================================================================
# Git Configuration
# ============================================================================

# Configure git user (for commits if agent updates state)
git config --global user.email "agent@synchestra.local"
git config --global user.name "Synchestra Agent"

# Store credentials if provided (using git credential store)
if [[ -n "${STATE_REPO_TOKEN}" ]]; then
    echo "[entrypoint] Configuring git credentials..."
    
    # Extract host from URL: https://github.com/user/repo.git -> github.com
    REPO_HOST=$(echo "${STATE_REPO_URL}" | sed -E 's|https?://([^/]+)/.*|\1|')
    
    # Store credential for git to use
    # Format: https://{username}:{token}@{host}
    GIT_CREDENTIAL_STORE="${HOME}/.git-credentials"
    echo "https://${GIT_USER}:${STATE_REPO_TOKEN}@${REPO_HOST}" > "${GIT_CREDENTIAL_STORE}"
    chmod 0600 "${GIT_CREDENTIAL_STORE}"
    
    # Configure git to use stored credentials
    git config --global credential.helper store
    git config --global "credential.${REPO_HOST}.useHttpPath" true
    
    echo "[entrypoint] ✓ Git credentials configured"
fi

# ============================================================================
# State Repository Clone
# ============================================================================

if [[ -d "${STATE_REPO_PATH}/.git" ]]; then
    echo "[entrypoint] State repository already exists, updating..."
    cd "${STATE_REPO_PATH}"
    git fetch origin main 2>&1 || {
        echo "[entrypoint] WARNING: git fetch failed (may be offline or network issue)"
    }
    cd - > /dev/null
else
    echo "[entrypoint] Cloning state repository..."
    
    git clone "${STATE_REPO_URL}" "${STATE_REPO_PATH}" 2>&1 || {
        echo "[entrypoint] ERROR: Failed to clone state repository"
        echo "[entrypoint] URL: ${STATE_REPO_URL}"
        echo "[entrypoint] Verify URL is correct and credentials are provided (if needed)"
        exit 1
    }
    
    # Verify clone succeeded
    if [[ ! -d "${STATE_REPO_PATH}/.git" ]]; then
        echo "[entrypoint] ERROR: State repository clone verification failed"
        exit 1
    fi
    
    echo "[entrypoint] ✓ State repository cloned"
fi

# ============================================================================
# Encryption Key Setup
# ============================================================================

KEY_FILE="${SECURE_PATH}/encryption.key"

if [[ -z "${ENCRYPTION_KEY}" ]]; then
    if [[ -f "${KEY_FILE}" ]]; then
        # Reuse existing key from previous container run
        ENCRYPTION_KEY=$(cat "${KEY_FILE}")
        echo "[entrypoint] Loaded existing encryption key from ${KEY_FILE}"
    else
        # Generate new key for first run
        ENCRYPTION_KEY=$(openssl rand -base64 32)
        echo "${ENCRYPTION_KEY}" > "${KEY_FILE}"
        chmod 0400 "${KEY_FILE}"
        echo "[entrypoint] Generated new encryption key, persisted to ${KEY_FILE}"
    fi
fi

export SYNCHESTRA_ENCRYPTION_KEY="${ENCRYPTION_KEY}"
echo "[entrypoint] ✓ Encryption key ready"

# ============================================================================
# Socket Path Setup
# ============================================================================

# Remove stale socket if it exists
if [[ -S "${SOCKET_PATH}" ]]; then
    echo "[entrypoint] Removing stale socket: ${SOCKET_PATH}"
    rm -f "${SOCKET_PATH}"
fi

# Ensure socket directory exists and has correct permissions
SOCKET_DIR=$(dirname "${SOCKET_PATH}")
mkdir -p "${SOCKET_DIR}"
chmod 0755 "${SOCKET_DIR}"

echo "[entrypoint] ✓ Socket path: ${SOCKET_PATH}"

# ============================================================================
# Start Agent
# ============================================================================

echo "[entrypoint] Starting Synchestra Sandbox Agent..."
echo "[entrypoint] Configuration:"
echo "[entrypoint]   - Project: ${PROJECT_ID}"
echo "[entrypoint]   - Work Dir: ${WORK_DIR}"
echo "[entrypoint]   - State Repo: ${STATE_REPO_PATH}"
echo "[entrypoint]   - Socket: ${SOCKET_PATH}"
echo "[entrypoint]   - Log Level: ${LOG_LEVEL}"
echo "[entrypoint] ---"

# Start agent with proper error handling
exec synchestra-sandbox-agent \
    --project-id="${PROJECT_ID}" \
    --work-dir="${WORK_DIR}" \
    --state-repo-path="${STATE_REPO_PATH}" \
    --socket-path="${SOCKET_PATH}" \
    --log-level="${LOG_LEVEL}" \
    --encryption-key="${ENCRYPTION_KEY}"
