package sourceref

// Features implemented: cli/code/deps
// https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/code/deps

import (
	"fmt"
	"regexp"
	"strings"
)

// Reference represents a parsed Synchestra resource reference found in source code.
type Reference struct {
	// ResolvedPath is the repo-root-relative path after type-prefix expansion (e.g., "spec/features/cli/task/claim")
	ResolvedPath string
	// CrossRepoSuffix is the optional @host/org/repo (empty string if same-repo reference)
	CrossRepoSuffix string
	// Type is the inferred resource type: "feature", "plan", "doc", or "" if unknown
	Type string
}

// DetectionRegex matches source references preceded by recognized comment prefixes.
// Regex: ^\s*(//|#|--|/\*|\*|%|;)\s*(synchestra:|https://synchestra\.io/)
var DetectionRegex = regexp.MustCompile(`^\s*(//|#|--|/\*|\*|%|;)\s*(synchestra:|https://synchestra\.io/)`)

// DetectReference checks if a line contains a Synchestra source reference.
// Returns true if the line has a recognized comment prefix followed by a reference marker.
func DetectReference(line string) bool {
	return DetectionRegex.MatchString(line)
}

// ExtractReference extracts the reference string from a line.
// Assumes the line has already been validated by DetectReference.
// Handles both short notation (synchestra:...) and expanded URLs (https://synchestra.io/...).
// Returns the extracted reference string, or empty string if not found.
func ExtractReference(line string) string {
	// Find the position of synchestra: or https://synchestra.io/
	idx := strings.Index(line, "synchestra:")
	if idx == -1 {
		idx = strings.Index(line, "https://synchestra.io/")
	}
	if idx == -1 {
		return ""
	}

	// Extract from the marker onwards and trim whitespace/quotes
	extracted := line[idx:]

	// Handle inline comments with trailing content
	// For expanded URLs, stop at whitespace or end of line
	if strings.HasPrefix(extracted, "https://") {
		if endIdx := strings.IndexAny(extracted, " \t\n\r"); endIdx != -1 {
			extracted = extracted[:endIdx]
		}
	} else if strings.HasPrefix(extracted, "synchestra:") {
		// For short notation, also stop at whitespace
		if endIdx := strings.IndexAny(extracted, " \t\n\r"); endIdx != -1 {
			extracted = extracted[:endIdx]
		}
	}

	return extracted
}

// ParseReference parses an extracted reference string and returns a Reference.
// Handles:
// - Short notation: synchestra:reference[@host/org/repo]
// - Expanded URLs: https://synchestra.io/host/org/repo/resolved/path
func ParseReference(extracted string) (*Reference, error) {
	if extracted == "" {
		return nil, fmt.Errorf("empty reference")
	}

	// Check if it's an expanded URL (https://synchestra.io/)
	if strings.HasPrefix(extracted, "https://synchestra.io/") {
		return parseExpandedURL(extracted)
	}

	// Otherwise, it's short notation (synchestra:...)
	if strings.HasPrefix(extracted, "synchestra:") {
		return parseShortNotation(extracted)
	}

	return nil, fmt.Errorf("unrecognized reference format: %s", extracted)
}

// parseExpandedURL parses an expanded URL in the form:
// https://synchestra.io/host/org/repo/resolved/path
func parseExpandedURL(url string) (*Reference, error) {
	// Remove the prefix
	url = strings.TrimPrefix(url, "https://synchestra.io/")

	// Split into segments
	parts := strings.Split(url, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid expanded URL format: too few path segments")
	}

	// First three segments are host/org/repo
	host := parts[0]
	org := parts[1]
	repo := parts[2]
	resolvedPath := strings.Join(parts[3:], "/")

	// Determine if this is a cross-repo reference
	currentHost, currentOrg, currentRepo := "github.com", "synchestra-io", "synchestra" // defaults (should be inferred)
	crossRepoSuffix := ""
	if host != currentHost || org != currentOrg || repo != currentRepo {
		crossRepoSuffix = fmt.Sprintf("@%s/%s/%s", host, org, repo)
	}

	// Infer type from resolved path
	refType := inferType(resolvedPath)

	return &Reference{
		ResolvedPath:    resolvedPath,
		CrossRepoSuffix: crossRepoSuffix,
		Type:            refType,
	}, nil
}

// parseShortNotation parses short notation in the form:
// synchestra:reference[@host/org/repo]
func parseShortNotation(notation string) (*Reference, error) {
	// Remove the synchestra: prefix
	notation = strings.TrimPrefix(notation, "synchestra:")

	// Check for cross-repo suffix
	crossRepoSuffix := ""
	reference := notation
	if idx := strings.LastIndex(notation, "@"); idx != -1 {
		crossRepoSuffix = notation[idx:]
		reference = notation[:idx]
	}

	// Resolve the reference (try type prefix expansion, then fallback to path)
	resolvedPath, err := resolveReference(reference)
	if err != nil {
		return nil, err
	}

	// Infer type from resolved path
	refType := inferType(resolvedPath)

	return &Reference{
		ResolvedPath:    resolvedPath,
		CrossRepoSuffix: crossRepoSuffix,
		Type:            refType,
	}, nil
}

// resolveReference attempts to resolve a reference by trying type prefix expansion first,
// then falling back to treating it as a full path.
func resolveReference(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty reference")
	}

	// Try type prefix expansion
	if strings.HasPrefix(ref, "feature/") {
		return "spec/features/" + strings.TrimPrefix(ref, "feature/"), nil
	}
	if strings.HasPrefix(ref, "plan/") {
		return "spec/plans/" + strings.TrimPrefix(ref, "plan/"), nil
	}
	if strings.HasPrefix(ref, "doc/") {
		return "docs/" + strings.TrimPrefix(ref, "doc/"), nil
	}

	// Fallback: treat as full path
	return ref, nil
}

// inferType infers the resource type from the resolved path.
func inferType(resolvedPath string) string {
	if strings.HasPrefix(resolvedPath, "spec/features/") {
		return "feature"
	}
	if strings.HasPrefix(resolvedPath, "spec/plans/") {
		return "plan"
	}
	if strings.HasPrefix(resolvedPath, "docs/") {
		return "doc"
	}
	return ""
}

// SourceRef represents a source file reference (file + line number).
type SourceRef struct {
	FilePath    string
	LineNumber  int
	LineContent string
}

// ScanLine scans a single line for references and returns any found.
// Returns nil if no reference is detected.
func ScanLine(line string) *Reference {
	if !DetectReference(line) {
		return nil
	}
	extracted := ExtractReference(line)
	if extracted == "" {
		return nil
	}
	ref, err := ParseReference(extracted)
	if err != nil {
		// Silently skip invalid references for now (could be logged)
		return nil
	}
	return ref
}
