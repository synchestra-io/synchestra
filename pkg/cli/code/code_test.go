package code

// Features implemented: cli/code/deps
// https://synchestra.io/github.com/synchestra-io/synchestra/spec/features/cli/code/deps

import (
	"os"
	"testing"
)

// TestIntegration_DepsCommand tests the code deps command end-to-end.
func TestIntegration_DepsCommand(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalCwd); err != nil {
			t.Errorf("Failed to restore directory: %v", err)
		}
	}()

	// Create test files with references
	tests := []struct {
		name        string
		files       map[string]string
		pathPattern string
		typeFilter  string
		expectError bool
		expectLines []string
		description string
	}{
		{
			name: "single_file_with_references",
			files: map[string]string{
				"file1.go": "// synchestra:feature/cli/code/deps\n// synchestra:plan/v2\n",
			},
			pathPattern: "file1.go",
			typeFilter:  "",
			expectError: false,
			expectLines: []string{
				"spec/features/cli/code/deps",
				"spec/plans/v2",
			},
			description: "Single file should output flat list",
		},
		{
			name: "multiple_files",
			files: map[string]string{
				"file1.go": "// synchestra:feature/cli/code/deps\n",
				"file2.go": "// synchestra:plan/v2\n",
			},
			pathPattern: "*.go",
			typeFilter:  "",
			expectError: false,
			expectLines: []string{
				"file1.go",
				"  spec/features/cli/code/deps",
				"file2.go",
				"  spec/plans/v2",
			},
			description: "Multiple files should be grouped with headers",
		},
		{
			name: "type_filter_feature",
			files: map[string]string{
				"file1.go": "// synchestra:feature/cli/code/deps\n// synchestra:plan/v2\n",
			},
			pathPattern: "file1.go",
			typeFilter:  "feature",
			expectError: false,
			expectLines: []string{
				"spec/features/cli/code/deps",
			},
			description: "Type filter should exclude non-matching types",
		},
		{
			name: "no_references",
			files: map[string]string{
				"file1.go": "// just a comment\nfunc main() {}\n",
			},
			pathPattern: "file1.go",
			typeFilter:  "",
			expectError: false,
			expectLines: []string{},
			description: "File with no references should output nothing",
		},
		{
			name:        "invalid_glob_pattern",
			files:       map[string]string{},
			pathPattern: "[invalid",
			typeFilter:  "",
			expectError: true,
			expectLines: []string{},
			description: "Invalid glob pattern should return error",
		},
		{
			name:        "invalid_type_filter",
			files:       map[string]string{},
			pathPattern: "*.go",
			typeFilter:  "invalid",
			expectError: true,
			expectLines: []string{},
			description: "Invalid type filter should return error",
		},
		{
			name: "cross_repo_references",
			files: map[string]string{
				"file1.go": "// synchestra:feature/x@github.com/acme/repo\n",
			},
			pathPattern: "file1.go",
			typeFilter:  "",
			expectError: false,
			expectLines: []string{
				"spec/features/x@github.com/acme/repo",
			},
			description: "Cross-repo references should include suffix",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create test files
			for filename, content := range test.files {
				if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}
			defer func() {
				// Clean up test files
				for filename := range test.files {
					_ = os.Remove(filename)
				}
			}()

			// Create a mock command for testing
			cmd := depsCommand()
			_ = cmd.Flags().Set("path", test.pathPattern)
			if test.typeFilter != "" {
				_ = cmd.Flags().Set("type", test.typeFilter)
			}

			// Run the command
			err := cmd.RunE(cmd, []string{})

			// Check error expectation
			if (err != nil) != test.expectError {
				t.Errorf("%s: error = %v, expectError = %v", test.description, err, test.expectError)
				return
			}

			// If we expect an error, we're done
			if test.expectError {
				return
			}

			// TODO: Capture output and verify it matches expectations
			// This requires modifying the command to support output capture
		})
	}
}
