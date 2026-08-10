package runner

// Features implemented: cli/runner/dispatch
// Features depended on:  repo-config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

var scpRemotePattern = regexp.MustCompile(`^([^\s@/:]+)@([^\s/:]+):(.+)$`)

type repositoryContext struct {
	Snapshot     dispatchcontract.RepositorySnapshot
	Root         string
	ProjectRoot  string
	Subdirectory string
}

type gitReader struct{}

func (gitReader) run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	commandArgs := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", commandArgs...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errorsAsExit(err, &exitErr) {
			message := strings.TrimSpace(string(exitErr.Stderr))
			if message != "" {
				return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
			}
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return output, nil
}

// errorsAsExit is isolated to keep command errors concise and testable.
func errorsAsExit(err error, target **exec.ExitError) bool {
	exitErr, ok := err.(*exec.ExitError)
	if ok {
		*target = exitErr
	}
	return ok
}

func resolveRepository(ctx context.Context, cwd string) (repositoryContext, error) {
	reader := gitReader{}
	rootOutput, err := reader.run(ctx, cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return repositoryContext{}, notFound("current directory is not inside a Git repository")
	}
	root := strings.TrimSpace(string(rootOutput))
	absCWD, err := filepath.Abs(cwd)
	if err != nil {
		return repositoryContext{}, unexpected("resolve current directory", err)
	}
	relationRoot, relationCWD := resolvePhysicalPaths(root, absCWD)
	physicalProjectRoot := findProjectRoot(relationCWD, relationRoot)
	subdirectory, err := filepath.Rel(relationRoot, physicalProjectRoot)
	if err != nil {
		return repositoryContext{}, unexpected("resolve project subdirectory", err)
	}
	if subdirectory == "." {
		subdirectory = ""
	}
	subdirectory = filepath.ToSlash(subdirectory)
	root = lexicalRepositoryRoot(absCWD, relationRoot, relationCWD, root)
	projectRoot := root
	if subdirectory != "" {
		projectRoot = filepath.Join(root, filepath.FromSlash(subdirectory))
	}

	revisionOutput, err := reader.run(ctx, root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return repositoryContext{}, unexpected("resolve immutable HEAD revision", err)
	}
	revision := strings.TrimSpace(string(revisionOutput))

	remoteOutput, err := reader.run(ctx, root, "config", "--get", "remote.origin.url")
	if err != nil {
		return repositoryContext{}, notFound("Git remote 'origin' is required for remote dispatch")
	}
	canonicalID, cloneURL, err := normalizeRemote(strings.TrimSpace(string(remoteOutput)))
	if err != nil {
		return repositoryContext{}, invalidArgs(err.Error())
	}

	baseRef := ""
	refOutput, refErr := reader.run(ctx, root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if refErr == nil {
		baseRef = strings.TrimSpace(string(refOutput))
	}

	projectID, err := readProjectID(projectRoot)
	if err != nil {
		return repositoryContext{}, unexpected("read project identity", err)
	}
	snapshot := dispatchcontract.RepositorySnapshot{
		CanonicalID:  canonicalID,
		CloneURL:     cloneURL,
		BaseRevision: revision,
		BaseRef:      baseRef,
		Subdirectory: subdirectory,
		ProjectID:    projectID,
	}
	if err := snapshot.Validate(); err != nil {
		return repositoryContext{}, invalidArgs(fmt.Sprintf("invalid repository snapshot: %v", err))
	}
	return repositoryContext{
		Snapshot: snapshot, Root: root, ProjectRoot: projectRoot, Subdirectory: subdirectory,
	}, nil
}

func lexicalRepositoryRoot(cwd, physicalRoot, physicalCWD, fallback string) string {
	relativeCWD, err := filepath.Rel(physicalRoot, physicalCWD)
	if err != nil || relativeCWD == "." || strings.HasPrefix(relativeCWD, ".."+string(filepath.Separator)) || relativeCWD == ".." {
		return fallback
	}
	lexicalRoot := cwd
	for range strings.Split(relativeCWD, string(filepath.Separator)) {
		lexicalRoot = filepath.Dir(lexicalRoot)
	}
	return lexicalRoot
}

// resolvePhysicalPaths aligns Git's physical top-level path with the caller's
// path before computing a subdirectory. macOS commonly exposes /var through a
// /private/var symlink; mixing those spellings would otherwise manufacture a
// parent-traversing relative path for a project that is inside its repository.
func resolvePhysicalPaths(root, cwd string) (string, string) {
	physicalRoot, rootErr := filepath.EvalSymlinks(root)
	physicalCWD, cwdErr := filepath.EvalSymlinks(cwd)
	if rootErr != nil || cwdErr != nil {
		return root, cwd
	}
	return physicalRoot, physicalCWD
}

func findProjectRoot(start, gitRoot string) string {
	current := start
	for {
		for _, name := range []string{"synchestra.yaml", "specscore.yaml", "specscore-spec-repo.yaml"} {
			if _, err := os.Stat(filepath.Join(current, name)); err == nil {
				return current
			}
		}
		if current == gitRoot {
			return gitRoot
		}
		parent := filepath.Dir(current)
		if parent == current || !pathWithin(gitRoot, parent) {
			return gitRoot
		}
		current = parent
	}
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func readProjectID(projectRoot string) (string, error) {
	cfg, err := readDispatchConfig(filepath.Join(projectRoot, "synchestra.yaml"), false)
	if err != nil {
		return "", err
	}
	if cfg.Hub == nil {
		return "", nil
	}
	return strings.TrimSpace(cfg.Hub.ID), nil
}

func normalizeRemote(raw string) (canonicalID string, cloneURL string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("git remote 'origin' is empty")
	}
	if strings.ContainsAny(raw, "?#") {
		return "", "", fmt.Errorf("git origin must not contain query or fragment data")
	}

	if !strings.Contains(raw, "://") {
		at := strings.IndexByte(raw, '@')
		if at > 0 && strings.Contains(raw[:at], ":") && strings.Contains(raw[at+1:], ":") {
			return "", "", fmt.Errorf("git SSH origin must not contain an inline password")
		}
	}

	if matches := scpRemotePattern.FindStringSubmatch(raw); matches != nil && !strings.Contains(raw, "://") {
		transportUser := matches[1] + "@"
		host := strings.ToLower(matches[2])
		repoPath := cleanRepositoryPath(matches[3])
		if host == "" || repoPath == "" {
			return "", "", fmt.Errorf("unsupported Git origin URL")
		}
		return host + "/" + repoPath, transportUser + host + ":" + repoPath + ".git", nil
	}

	parsed, parseErr := url.Parse(raw)
	if parseErr != nil || parsed.Host == "" {
		return "", "", fmt.Errorf("unsupported Git origin URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "ssh" && parsed.Scheme != "git" {
		return "", "", fmt.Errorf("unsupported Git origin scheme %q; configure an HTTP(S) or hosted SSH origin", parsed.Scheme)
	}
	host := strings.ToLower(parsed.Hostname())
	if parsed.Port() != "" {
		host += ":" + parsed.Port()
	}
	repoPath := cleanRepositoryPath(parsed.Path)
	if host == "" || repoPath == "" || !strings.Contains(repoPath, "/") {
		return "", "", fmt.Errorf("unsupported Git origin URL")
	}
	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "http", "https":
		if parsed.User != nil {
			return "", "", fmt.Errorf("git HTTP(S) origin must not contain inline userinfo")
		}
		cloneURL = scheme + "://" + host + "/" + repoPath + ".git"
	case "ssh":
		transportUser := ""
		if parsed.User != nil {
			if _, hasPassword := parsed.User.Password(); hasPassword {
				return "", "", fmt.Errorf("git SSH origin must not contain an inline password")
			}
			if parsed.User.Username() == "" {
				return "", "", fmt.Errorf("git SSH origin username must not be empty")
			}
			transportUser = url.User(parsed.User.Username()).String() + "@"
		}
		cloneURL = "ssh://" + transportUser + host + "/" + repoPath + ".git"
	case "git":
		if parsed.User != nil {
			return "", "", fmt.Errorf("git origin must not contain inline userinfo")
		}
		cloneURL = "git://" + host + "/" + repoPath + ".git"
	}
	return host + "/" + repoPath, cloneURL, nil
}

func cleanRepositoryPath(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	value = strings.TrimSuffix(value, ".git")
	return value
}

func (r repositoryContext) listFiles(ctx context.Context) ([]string, error) {
	output, err := (gitReader{}).run(ctx, r.Root, "ls-tree", "-r", "--name-only", "-z", "HEAD")
	if err != nil {
		return nil, unexpected("list repository files at HEAD", err)
	}
	parts := strings.Split(string(output), "\x00")
	files := make([]string, 0, len(parts))
	for _, file := range parts {
		if file == "" {
			continue
		}
		if r.Subdirectory != "" && file != r.Subdirectory && !strings.HasPrefix(file, r.Subdirectory+"/") {
			continue
		}
		files = append(files, file)
	}
	return files, nil
}

func (r repositoryContext) readBlob(ctx context.Context, path string) ([]byte, error) {
	output, err := (gitReader{}).run(ctx, r.Root, "show", "HEAD:"+path)
	if err != nil {
		return nil, err
	}
	return output, nil
}
