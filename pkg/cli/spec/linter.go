package spec

// Features depended on: cli/feature

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// linter orchestrates rule checking across the spec tree.
type linter struct {
	opts    LintOptions
	ruleSet map[string]checker
}

// checker is the interface for individual rule implementations.
type checker interface {
	check(specRoot string) ([]Violation, error)
	name() string
	severity() string
}

func newLinter(opts LintOptions) *linter {
	l := &linter{
		opts:    opts,
		ruleSet: make(map[string]checker),
	}

	l.registerChecker(newReadmeExistsChecker())
	l.registerChecker(newOQSectionChecker())
	l.registerChecker(newIndexEntriesChecker())
	l.registerChecker(newHeadingLevelsChecker())
	l.registerChecker(newFeatureRefSyntaxChecker())
	l.registerChecker(newInternalLinksChecker())
	l.registerChecker(newForwardRefsChecker())
	l.registerChecker(newCodeAnnotationsChecker())
	l.registerChecker(newPlanHierarchyChecker())

	return l
}

func (l *linter) registerChecker(c checker) {
	l.ruleSet[c.name()] = c
}

func (l *linter) isRuleEnabled(ruleName string) bool {
	if len(l.opts.Rules) > 0 {
		for _, r := range l.opts.Rules {
			if r == ruleName {
				return true
			}
		}
		return false
	}

	if len(l.opts.Ignore) > 0 {
		for _, r := range l.opts.Ignore {
			if r == ruleName {
				return false
			}
		}
		return true
	}

	return true
}

// lint runs all enabled checkers and returns violations.
func (l *linter) lint() ([]Violation, error) {
	var violations []Violation

	for ruleName, c := range l.ruleSet {
		if !l.isRuleEnabled(ruleName) {
			continue
		}

		v, err := c.check(l.opts.SpecRoot)
		if err != nil {
			return nil, fmt.Errorf("checker %s: %v", ruleName, err)
		}
		violations = append(violations, v...)
	}

	return violations, nil
}

// walkSpecDirs returns subdirectory paths under specRoot, skipping hidden dirs
// except .github (whose children are traversed but .github itself is skipped).
func walkSpecDirs(specRoot string, fn func(dirPath, relPath string) error) error {
	return filepath.Walk(specRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			if info.Name() == ".github" {
				return nil // traverse children but skip .github itself
			}
			return filepath.SkipDir
		}
		relPath, _ := filepath.Rel(specRoot, path)
		if relPath == "." {
			relPath = specRoot
		}
		return fn(path, relPath)
	})
}
