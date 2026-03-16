package testscenario

// Features implemented: testing-framework/test-runner

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var varPattern = regexp.MustCompile(`\$\{\{\s*([^}]+?)\s*\}\}`)

// ExecContext holds variable state during scenario execution.
// It is safe for concurrent use by parallel steps.
type ExecContext struct {
	mu          sync.RWMutex
	contextVars map[string]string
	stepOutputs map[string]map[string]string
}

// NewExecContext creates a new empty execution context.
func NewExecContext() *ExecContext {
	return &ExecContext{
		contextVars: make(map[string]string),
		stepOutputs: make(map[string]map[string]string),
	}
}

// StoreOutput stores a named output from a step.
func (c *ExecContext) StoreOutput(stepName, name, value string, store OutputStore) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch store {
	case StoreContext, StoreBoth:
		if _, exists := c.contextVars[name]; exists {
			return fmt.Errorf("duplicate context key %q", name)
		}
		c.contextVars[name] = value
	}
	switch store {
	case StoreStep, StoreBoth:
		if c.stepOutputs[stepName] == nil {
			c.stepOutputs[stepName] = make(map[string]string)
		}
		c.stepOutputs[stepName][name] = value
	}
	return nil
}

// ContextVarsAsEnv returns all context variables as KEY=VALUE env var pairs.
func (c *ExecContext) ContextVarsAsEnv() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	env := make([]string, 0, len(c.contextVars))
	for k, v := range c.contextVars {
		env = append(env, k+"="+v)
	}
	return env
}

// ResolveVar resolves a variable reference like "context.pid" or "steps.create.outputs.id".
func (c *ExecContext) ResolveVar(ref string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if name, ok := strings.CutPrefix(ref, "context."); ok {
		if val, found := c.contextVars[name]; found {
			return val, nil
		}
		return "", fmt.Errorf("unknown context variable %q", name)
	}
	if rest, ok := strings.CutPrefix(ref, "steps."); ok {
		parts := strings.SplitN(rest, ".outputs.", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid step output reference %q", ref)
		}
		stepName, outputName := parts[0], parts[1]
		if outputs, found := c.stepOutputs[stepName]; found {
			if val, found := outputs[outputName]; found {
				return val, nil
			}
		}
		return "", fmt.Errorf("unknown step output %q", ref)
	}
	return "", fmt.Errorf("unknown variable reference %q", ref)
}

// ResolveString replaces all ${{ ... }} references in a string.
func (c *ExecContext) ResolveString(s string) (string, error) {
	var resolveErr error
	result := varPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := varPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		val, err := c.ResolveVar(strings.TrimSpace(sub[1]))
		if err != nil {
			resolveErr = err
			return match
		}
		return val
	})
	return result, resolveErr
}
