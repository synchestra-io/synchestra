# This Dockerfile should be placed at: build/Dockerfile.sandbox-agent (in this repository root)
#
# Multi-stage build for Synchestra Sandbox Agent container
# Produces minimal, secure image optimized for execution isolation

# ============================================================================
# Stage 1: Builder (Go compilation)
# ============================================================================

ARG GO_VERSION=1.21-alpine
FROM golang:${GO_VERSION} AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    make

# Copy go mod/sum
COPY go.mod go.sum ./

# Download dependencies (cacheable layer)
RUN go mod download

# Copy source code
COPY . .

# Build gRPC agent binary
# Features: static linking, strip symbols for smaller binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /tmp/synchestra-sandbox-agent \
    ./internal/sandbox/agent/cmd

# Verify binary was created
RUN ls -lh /tmp/synchestra-sandbox-agent && file /tmp/synchestra-sandbox-agent

# ============================================================================
# Stage 2: Runtime (minimal Alpine)
# ============================================================================

ARG ALPINE_VERSION=3.19
FROM alpine:${ALPINE_VERSION}

LABEL maintainer="Synchestra Team"
LABEL description="Synchestra Sandbox Agent - Isolated command execution environment"

# Install runtime dependencies (minimal set)
RUN apk add --no-cache \
    bash \
    git \
    ca-certificates \
    tini \
    coreutils

# Verify critical binaries
RUN which bash git ca-certificates tini

# Create unprivileged user (UID 1000 standard)
RUN addgroup -g 1000 synchestra && \
    adduser -D -u 1000 -G synchestra synchestra

# Create workspace directory (will be bind-mounted from host at runtime)
RUN mkdir -p /workspace && chown synchestra:synchestra /workspace

# Create runtime directories (with correct permissions)
RUN mkdir -p /run && chown synchestra:synchestra /run && \
    mkdir -p /tmp && chmod 1777 /tmp

# Copy binary from builder
COPY --from=builder /tmp/synchestra-sandbox-agent /usr/local/bin/

# Verify binary
RUN ls -lh /usr/local/bin/synchestra-sandbox-agent && \
    file /usr/local/bin/synchestra-sandbox-agent

# Copy entrypoint script
COPY build/docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# ============================================================================
# Security Hardening
# ============================================================================

# Switch to unprivileged user
USER synchestra:synchestra

# Set read-only root filesystem (except /tmp, /run, /workspace)
# This is enforced at runtime with --read-only flag

# ============================================================================
# Runtime Configuration
# ============================================================================

# Health check (optional, can be overridden by orchestrator)
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD synchestra-sandbox-agent health || exit 1

# Use tini as init process (reap zombies)
ENTRYPOINT ["/sbin/tini", "--"]

# Default command (start gRPC agent)
CMD ["/entrypoint.sh"]

# ============================================================================
# Image Metadata
# ============================================================================

# These are set at build time:
# docker build --label="version=1.0.0" --label="commit=abc1234" ...

EXPOSE 50051

# Image should be built with: docker build -t synchestra/sandbox-agent:latest -f build/Dockerfile.sandbox-agent .
