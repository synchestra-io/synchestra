# Container Build Automation

## Makefile Targets (in this repository root)

Add these targets to the main `Makefile`:

```makefile
# Build sandbox agent container image
.PHONY: sandbox-image
sandbox-image:
	@echo "Building Synchestra Sandbox Agent image..."
	docker build \
		-t synchestra/sandbox-agent:latest \
		-t synchestra/sandbox-agent:$(VERSION) \
		--label "version=$(VERSION)" \
		--label "commit=$(GIT_COMMIT)" \
		--label "build_date=$(BUILD_DATE)" \
		-f build/Dockerfile.sandbox-agent \
		.
	@echo "✓ Image built: synchestra/sandbox-agent:$(VERSION)"
	@docker images | grep sandbox-agent | head -1

# Scan sandbox image for vulnerabilities
.PHONY: sandbox-image-scan
sandbox-image-scan:
	@echo "Scanning Sandbox Agent image for vulnerabilities..."
	trivy image \
		--severity HIGH,CRITICAL \
		--exit-code 1 \
		synchestra/sandbox-agent:latest
	@echo "✓ Vulnerability scan passed"

# Push sandbox image to registry
.PHONY: sandbox-image-push
sandbox-image-push: sandbox-image
	@echo "Pushing Sandbox Agent image to registry..."
	docker tag synchestra/sandbox-agent:latest $(REGISTRY)/synchestra/sandbox-agent:latest
	docker tag synchestra/sandbox-agent:$(VERSION) $(REGISTRY)/synchestra/sandbox-agent:$(VERSION)
	docker push $(REGISTRY)/synchestra/sandbox-agent:latest
	docker push $(REGISTRY)/synchestra/sandbox-agent:$(VERSION)
	@echo "✓ Image pushed: $(REGISTRY)/synchestra/sandbox-agent:$(VERSION)"

# Build multi-platform images (requires buildx)
.PHONY: sandbox-image-buildx
sandbox-image-buildx:
	@echo "Building multi-platform Sandbox Agent image..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t synchestra/sandbox-agent:latest \
		-t synchestra/sandbox-agent:$(VERSION) \
		--push \
		-f build/Dockerfile.sandbox-agent \
		.
	@echo "✓ Multi-platform image built and pushed"

# Test sandbox image (run health check)
.PHONY: sandbox-image-test
sandbox-image-test: sandbox-image
	@echo "Testing Sandbox Agent image..."
	docker run --rm synchestra/sandbox-agent:latest /usr/local/bin/synchestra-sandbox-agent --version
	@echo "✓ Image test passed"

# Build and test sandbox image (full pipeline)
.PHONY: sandbox-image-build-all
sandbox-image-build-all: sandbox-image sandbox-image-test sandbox-image-scan
	@echo "✓ Sandbox Agent image build complete"

# Show sandbox image info
.PHONY: sandbox-image-info
sandbox-image-info:
	@docker inspect synchestra/sandbox-agent:latest | jq '.[0] | {
		Id: .Id,
		RepoTags: .RepoTags,
		Size: .Size,
		VirtualSize: .VirtualSize,
		Created: .Created,
		Architecture: .Architecture,
		Os: .Os,
		Config: .Config | {
			User: .User,
			Entrypoint: .Entrypoint,
			Cmd: .Cmd,
			Env: .Env,
			Healthcheck: .Healthcheck,
			Labels: .Labels
		}
	}'

# Variables
VERSION ?= 1.0.0
REGISTRY ?= myregistry.azurecr.io
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
```

## Build Scripts

### build/build-sandbox-image.sh

```bash
#!/bin/bash
# Build and optionally push Synchestra Sandbox Agent container image
# Place this script at: build/build-sandbox-image.sh (in repository root)

# Configuration
REGISTRY="${REGISTRY:-synchestra}"
VERSION="${VERSION:-latest}"
BUILD_ARGS="${BUILD_ARGS:-}"
PUSH="${PUSH:-false}"
SCAN="${SCAN:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Validate prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check Dockerfile
    if [[ ! -f "build/Dockerfile.sandbox-agent" ]]; then
        log_error "Dockerfile not found at build/Dockerfile.sandbox-agent"
        exit 1
    fi
    
    # Check entrypoint script
    if [[ ! -f "build/docker-entrypoint.sh" ]]; then
        log_error "Entrypoint script not found at build/docker-entrypoint.sh"
        exit 1
    fi
    
    # Check trivy if scanning
    if [[ "${SCAN}" == "true" ]] && ! command -v trivy &> /dev/null; then
        log_warn "trivy not found; skipping vulnerability scan"
        SCAN="false"
    fi
    
    log_info "Prerequisites OK"
}

# Build image
build_image() {
    log_info "Building Sandbox Agent image: ${REGISTRY}/synchestra/sandbox-agent:${VERSION}"
    
    local build_cmd="docker build"
    
    # Add labels
    build_cmd+=" --label version=${VERSION}"
    build_cmd+=" --label commit=$(git rev-parse HEAD)"
    build_cmd+=" --label build_date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
    
    # Add tags
    build_cmd+=" -t synchestra/sandbox-agent:latest"
    build_cmd+=" -t synchestra/sandbox-agent:${VERSION}"
    if [[ "${REGISTRY}" != "synchestra" ]]; then
        build_cmd+=" -t ${REGISTRY}/synchestra/sandbox-agent:${VERSION}"
    fi
    
    # Add build args
    if [[ -n "${BUILD_ARGS}" ]]; then
        for arg in ${BUILD_ARGS}; do
            build_cmd+=" --build-arg ${arg}"
        done
    fi
    
    # Build
    build_cmd+=" -f build/Dockerfile.sandbox-agent ."
    
    if eval "${build_cmd}"; then
        log_info "✓ Image built successfully"
    else
        log_error "Image build failed"
        exit 1
    fi
}

# Scan image
scan_image() {
    if [[ "${SCAN}" != "true" ]]; then
        return 0
    fi
    
    log_info "Scanning image for vulnerabilities..."
    
    if trivy image --severity HIGH,CRITICAL "synchestra/sandbox-agent:${VERSION}"; then
        log_info "✓ Vulnerability scan passed"
    else
        log_error "Vulnerability scan failed"
        exit 1
    fi
}

# Push image
push_image() {
    if [[ "${PUSH}" != "true" ]]; then
        return 0
    fi
    
    log_info "Pushing image to registry: ${REGISTRY}"
    
    docker tag "synchestra/sandbox-agent:${VERSION}" "${REGISTRY}/synchestra/sandbox-agent:${VERSION}"
    docker tag "synchestra/sandbox-agent:latest" "${REGISTRY}/synchestra/sandbox-agent:latest"
    
    if docker push "${REGISTRY}/synchestra/sandbox-agent:${VERSION}" && \
       docker push "${REGISTRY}/synchestra/sandbox-agent:latest"; then
        log_info "✓ Image pushed successfully"
    else
        log_error "Image push failed"
        exit 1
    fi
}

# Display image info
show_image_info() {
    log_info "Image information:"
    docker images "synchestra/sandbox-agent:${VERSION}" --no-trunc
    
    local size=$(docker inspect "synchestra/sandbox-agent:${VERSION}" | jq -r '.[0].Size')
    log_info "Image size: $(numfmt --to=iec-i --suffix=B ${size})"
}

# Main
main() {
    log_info "Synchestra Sandbox Agent Image Builder"
    log_info "Registry: ${REGISTRY}"
    log_info "Version: ${VERSION}"
    
    check_prerequisites
    build_image
    scan_image
    push_image
    show_image_info
    
    log_info "✓ Build complete"
}

main "$@"
```

## CI/CD Pipeline

### GitHub Actions Workflow

```yaml
# .github/workflows/sandbox-image.yml
name: Build & Push Sandbox Agent Image

on:
  push:
    branches: [main, develop]
    paths:
      - 'internal/sandbox/**'
      - 'build/Dockerfile.sandbox-agent'
      - 'build/docker-entrypoint.sh'
      - '.github/workflows/sandbox-image.yml'
  
  workflow_dispatch:
    inputs:
      registry:
        description: 'Container registry'
        required: false
        default: 'ghcr.io'
      push:
        description: 'Push to registry'
        type: boolean
        default: false

env:
  REGISTRY: ${{ github.event.inputs.registry || 'ghcr.io' }}
  IMAGE_NAME: synchestra/sandbox-agent

jobs:
  build:
    runs-on: ubuntu-latest
    
    permissions:
      contents: read
      packages: write
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3
    
    - name: Log in to Container Registry
      if: github.event.inputs.push == 'true' || github.event_name == 'push'
      uses: docker/login-action@v3
      with:
        registry: ${{ env.REGISTRY }}
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v5
      with:
        images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
        tags: |
          type=ref,event=branch
          type=semver,pattern={{version}}
          type=sha
    
    - name: Build & Scan
      uses: docker/build-push-action@v5
      with:
        context: .
        file: ./build/Dockerfile.sandbox-agent
        push: ${{ github.event.inputs.push == 'true' || github.event_name == 'push' }}
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
        cache-from: type=registry,ref=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:buildcache
        cache-to: type=registry,ref=${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:buildcache,mode=max
    
    - name: Scan with Trivy
      uses: aquasecurity/trivy-action@master
      with:
        image-ref: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:latest
        format: 'sarif'
        output: 'trivy-results.sarif'
    
    - name: Upload Trivy results
      uses: github/codeql-action/upload-sarif@v2
      with:
        sarif_file: 'trivy-results.sarif'
```

## Testing Container Image

### Integration Test

```bash
#!/bin/bash
# Test sandbox container locally

set -euo pipefail

PROJECT_ID="test-project"
WORK_DIR="/tmp/synchestra-test-${PROJECT_ID}"
SOCKET_PATH="/tmp/synchestra-${PROJECT_ID}.sock"

# Create test workspace
mkdir -p "${WORK_DIR}"

# Initialize git repo
cd "${WORK_DIR}"
git init
git config user.email "test@synchestra.local"
git config user.name "Test User"
touch README.md
git add README.md
git commit -m "initial"
cd -

# Run container
docker run --rm \
  --name test-sandbox-${PROJECT_ID} \
  --cap-drop=ALL \
  --read-only \
  --tmpfs /tmp \
  -v ${WORK_DIR}:/workspace/${PROJECT_ID}:rw \
  -v /tmp:/var/run:rw \
  -e SYNCHESTRA_PROJECT_ID=${PROJECT_ID} \
  -e SYNCHESTRA_STATE_REPO_URL=file://${WORK_DIR} \
  -e SYNCHESTRA_LOG_LEVEL=debug \
  synchestra/sandbox-agent:latest &

CONTAINER_PID=$!
sleep 2

# Test gRPC health check
echo "Testing health check..."
docker exec test-sandbox-${PROJECT_ID} \
  synchestra-sandbox-agent health \
  --socket ${SOCKET_PATH} \
  || {
    echo "Health check failed"
    kill ${CONTAINER_PID} 2>/dev/null || true
    exit 1
  }

echo "✓ Container test passed"

# Cleanup
kill ${CONTAINER_PID} 2>/dev/null || true
rm -rf ${WORK_DIR}
```
