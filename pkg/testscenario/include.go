package testscenario

// Features implemented: testing-framework/test-runner

import (
	"fmt"
	"os"
	"path/filepath"
)

// IncludeResolver resolves sub-flow includes with cycle detection.
type IncludeResolver struct{}

// NewIncludeResolver creates a new include resolver.
func NewIncludeResolver() *IncludeResolver {
	return &IncludeResolver{}
}

// Resolve reads and parses an included scenario file. The seen set tracks
// visited paths for cycle detection. Pass nil for the initial call.
// After parsing, it recursively resolves any include references in steps
// to detect circular dependencies.
func (r *IncludeResolver) Resolve(path string, seen map[string]bool) (*Scenario, error) {
	if seen == nil {
		seen = make(map[string]bool)
	}
	if seen[path] {
		return nil, fmt.Errorf("circular include detected: %s", path)
	}
	seen[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading include %s: %w", path, err)
	}

	scenario, err := ParseScenario(data)
	if err != nil {
		return nil, err
	}

	// Recursively resolve any sub-flow includes to detect cycles.
	dir := filepath.Dir(path)
	for _, step := range scenario.Steps {
		if step.Include == "" {
			continue
		}
		includePath := step.Include
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(dir, includePath)
		}
		// Copy seen map so sibling includes don't interfere with each other.
		seenCopy := make(map[string]bool, len(seen))
		for k, v := range seen {
			seenCopy[k] = v
		}
		if _, err := r.Resolve(includePath, seenCopy); err != nil {
			return nil, err
		}
	}

	return scenario, nil
}
