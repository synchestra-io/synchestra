package project

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// DeriveTitle returns the project title using the fallback chain:
// 1. Explicit title (if non-empty)
// 2. First # heading in specRepoDir/README.md
// 3. repoIdentifier (e.g., "acme-spec")
func DeriveTitle(explicit, specRepoDir, repoIdentifier string) string {
	if explicit != "" {
		return explicit
	}
	data, err := os.ReadFile(filepath.Join(specRepoDir, "README.md"))
	if err == nil {
		if h := extractFirstHeading(data); h != "" {
			return h
		}
	}
	return repoIdentifier
}

// extractFirstHeading returns the text of the first # heading in markdown content.
func extractFirstHeading(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
