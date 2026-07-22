package runner

// Features implemented: cli/runner/dispatch
// Features depended on:  repo-config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	dispatchcontract "github.com/synchestra-io/synchestra-servers/pkg/dispatch-contract"
	"gopkg.in/yaml.v3"
)

var (
	planTitlePattern   = regexp.MustCompile(`(?m)^#\s+Plan:\s*(.+?)\s*$`)
	taskTitlePattern   = regexp.MustCompile(`(?m)^#\s+(?:Task:\s*)?(.+?)\s*$`)
	taskHeadingPattern = regexp.MustCompile(`(?m)^(#{3,6})\s+(?:Task\s+)?([0-9]+(?:\.[0-9]+)*):\s*(.+?)\s*$`)
	markdownIDPattern  = regexp.MustCompile(`(?mi)^\*\*(?:ID|Plan ID|Task ID):\*\*\s*` + "`?" + `([^\s` + "`" + `]+)` + "`?" + `\s*$`)
)

type targetSelector struct {
	Kind  dispatchcontract.SpecScoreTargetKind
	Value string
}

type targetCandidate struct {
	Kind    dispatchcontract.SpecScoreTargetKind
	ID      string
	Name    string
	Path    string
	Aliases []string
	Content []byte
}

func resolveTarget(ctx context.Context, repo repositoryContext, selector targetSelector) (dispatchcontract.SpecScoreSource, error) {
	selector.Value = strings.TrimSpace(selector.Value)
	if selector.Value == "" {
		return dispatchcontract.SpecScoreSource{}, invalidArgs("SpecScore target cannot be empty")
	}

	candidates, err := discoverTargets(ctx, repo)
	if err != nil {
		return dispatchcontract.SpecScoreSource{}, err
	}
	if selector.Kind != "" {
		candidates = filterTargetKind(candidates, selector.Kind)
	}

	matches := matchTargetCandidates(repo, selector.Value, candidates)
	if len(matches) == 0 {
		kind := "Plan or Task"
		if selector.Kind != "" {
			kind = strings.ToUpper(string(selector.Kind)[:1]) + string(selector.Kind)[1:]
		}
		return dispatchcontract.SpecScoreSource{}, notFound(fmt.Sprintf("SpecScore %s target %q was not found at HEAD", kind, selector.Value))
	}
	if len(matches) > 1 {
		sort.Slice(matches, func(i, j int) bool {
			if matches[i].Kind != matches[j].Kind {
				return matches[i].Kind < matches[j].Kind
			}
			if matches[i].ID != matches[j].ID {
				return matches[i].ID < matches[j].ID
			}
			return matches[i].Path < matches[j].Path
		})
		descriptions := make([]string, 0, len(matches))
		for _, match := range matches {
			descriptions = append(descriptions, fmt.Sprintf("%s:%s (%s)", match.Kind, match.ID, match.Path))
		}
		commandErr := invalidArgs(fmt.Sprintf("target %q is ambiguous: %s", selector.Value, strings.Join(descriptions, ", ")))
		commandErr.apiError.Details = map[string]string{"candidates": strings.Join(descriptions, ",")}
		return dispatchcontract.SpecScoreSource{}, commandErr
	}

	match := matches[0]
	digest := sha256.Sum256(match.Content)
	return dispatchcontract.SpecScoreSource{
		TargetKind:     match.Kind,
		TargetID:       match.ID,
		TargetPath:     match.Path,
		TargetRevision: repo.Snapshot.BaseRevision,
		SnapshotHash:   fmt.Sprintf("sha256:%x", digest),
	}, nil
}

func discoverTargets(ctx context.Context, repo repositoryContext) ([]targetCandidate, error) {
	files, err := repo.listFiles(ctx)
	if err != nil {
		return nil, err
	}
	var candidates []targetCandidate
	for _, repoPath := range files {
		projectPath := strings.TrimPrefix(repoPath, repo.Subdirectory+"/")
		if !strings.HasSuffix(strings.ToLower(projectPath), ".md") {
			continue
		}
		planPath := isPlanPath(projectPath)
		taskPath := isTaskPath(projectPath)
		if !planPath && !taskPath {
			continue
		}
		data, readErr := repo.readBlob(ctx, repoPath)
		if readErr != nil {
			return nil, unexpected("read SpecScore target at HEAD", readErr)
		}
		if planPath {
			plan, ok := parsePlanCandidate(repoPath, projectPath, data)
			if ok {
				candidates = append(candidates, plan)
				candidates = append(candidates, parsePlanTasks(plan, data)...)
			}
		}
		if taskPath {
			if task, ok := parseTaskFileCandidate(repoPath, projectPath, data); ok {
				candidates = append(candidates, task)
			}
		}
	}
	return deduplicateTargets(candidates), nil
}

func isPlanPath(projectPath string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(projectPath), "./")
	if clean == "spec/plans/README.md" || clean == "plans/README.md" {
		return false
	}
	return strings.HasPrefix(clean, "spec/plans/") || strings.HasPrefix(clean, "plans/")
}

func isTaskPath(projectPath string) bool {
	clean := strings.TrimPrefix(filepath.ToSlash(projectPath), "./")
	return strings.HasPrefix(clean, "tasks/") || strings.HasPrefix(clean, ".synchestra/tasks/")
}

func parsePlanCandidate(repoPath, projectPath string, data []byte) (targetCandidate, bool) {
	titleMatch := planTitlePattern.FindSubmatch(data)
	if titleMatch == nil {
		return targetCandidate{}, false
	}
	id := documentID(data)
	if id == "" {
		id = planSlug(projectPath)
	}
	if id == "" {
		return targetCandidate{}, false
	}
	return targetCandidate{
		Kind:    dispatchcontract.SpecScoreTargetPlan,
		ID:      id,
		Name:    strings.TrimSpace(string(titleMatch[1])),
		Path:    repoPath,
		Aliases: []string{id, "plan/" + id, repoPath, projectPath, planSlug(projectPath)},
		Content: data,
	}, true
}

func parsePlanTasks(plan targetCandidate, data []byte) []targetCandidate {
	matches := taskHeadingPattern.FindAllSubmatchIndex(data, -1)
	tasks := make([]targetCandidate, 0, len(matches))
	for index, match := range matches {
		if len(match) < 8 {
			continue
		}
		sectionEnd := len(data)
		if index+1 < len(matches) {
			sectionEnd = matches[index+1][0]
		}
		number := string(data[match[4]:match[5]])
		name := strings.TrimSpace(string(data[match[6]:match[7]]))
		taskID := plan.ID + "#task-" + strings.ReplaceAll(number, ".", "-")
		tasks = append(tasks, targetCandidate{
			Kind:    dispatchcontract.SpecScoreTargetTask,
			ID:      taskID,
			Name:    name,
			Path:    plan.Path,
			Aliases: []string{taskID, "Task " + number, number, name},
			Content: data[match[0]:sectionEnd],
		})
	}
	return tasks
}

func parseTaskFileCandidate(repoPath, projectPath string, data []byte) (targetCandidate, bool) {
	titleMatch := taskTitlePattern.FindSubmatch(data)
	if titleMatch == nil {
		return targetCandidate{}, false
	}
	id := documentID(data)
	if id == "" {
		id = taskSlug(projectPath)
	}
	if id == "" {
		return targetCandidate{}, false
	}
	return targetCandidate{
		Kind:    dispatchcontract.SpecScoreTargetTask,
		ID:      id,
		Name:    strings.TrimSpace(string(titleMatch[1])),
		Path:    repoPath,
		Aliases: []string{id, "task/" + id, repoPath, projectPath, taskSlug(projectPath)},
		Content: data,
	}, true
}

func documentID(data []byte) string {
	if markdownMatch := markdownIDPattern.FindSubmatch(data); markdownMatch != nil {
		return strings.TrimSpace(string(markdownMatch[1]))
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return ""
	}
	remainder := strings.TrimPrefix(text, "---\n")
	end := strings.Index(remainder, "\n---")
	if end < 0 {
		return ""
	}
	var metadata map[string]any
	if err := yaml.Unmarshal([]byte(remainder[:end]), &metadata); err != nil {
		return ""
	}
	for _, key := range []string{"id", "plan_id", "task_id"} {
		if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func planSlug(projectPath string) string {
	clean := filepath.ToSlash(projectPath)
	clean = strings.TrimPrefix(clean, "spec/plans/")
	clean = strings.TrimPrefix(clean, "plans/")
	if strings.HasSuffix(clean, "/README.md") {
		clean = strings.TrimSuffix(clean, "/README.md")
	} else {
		clean = strings.TrimSuffix(clean, filepath.Ext(clean))
	}
	return strings.Trim(clean, "/")
}

func taskSlug(projectPath string) string {
	clean := filepath.ToSlash(projectPath)
	clean = strings.TrimPrefix(clean, ".synchestra/tasks/")
	clean = strings.TrimPrefix(clean, "tasks/")
	if strings.HasSuffix(clean, "/README.md") {
		clean = strings.TrimSuffix(clean, "/README.md")
	} else {
		clean = strings.TrimSuffix(clean, filepath.Ext(clean))
	}
	return strings.Trim(clean, "/")
}

func matchTargetCandidates(repo repositoryContext, value string, candidates []targetCandidate) []targetCandidate {
	pathValue := normalizeTargetPath(repo, value)
	if pathValue != "" {
		if pathMatches := candidatesMatching(candidates, func(candidate targetCandidate) bool {
			return containsExact(candidate.Aliases, pathValue)
		}); len(pathMatches) > 0 {
			return pathMatches
		}
	}
	if idMatches := candidatesMatching(candidates, func(candidate targetCandidate) bool {
		return strings.EqualFold(candidate.ID, value) || containsFold(candidate.Aliases, value)
	}); len(idMatches) > 0 {
		return idMatches
	}
	normalized := normalizeName(value)
	if normalized == "" {
		return nil
	}
	if exactNameMatches := candidatesMatching(candidates, func(candidate targetCandidate) bool {
		return normalizeName(candidate.Name) == normalized
	}); len(exactNameMatches) > 0 {
		return exactNameMatches
	}
	return candidatesMatching(candidates, func(candidate targetCandidate) bool {
		candidateName := normalizeName(candidate.Name)
		return strings.Contains(candidateName, normalized)
	})
}

func normalizeTargetPath(repo repositoryContext, value string) string {
	pathPart := strings.SplitN(value, "#", 2)[0]
	if filepath.IsAbs(pathPart) {
		rel, err := filepath.Rel(repo.Root, filepath.Clean(pathPart))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return ""
		}
		return filepath.ToSlash(rel)
	}
	clean := filepath.ToSlash(filepath.Clean(pathPart))
	clean = strings.TrimPrefix(clean, "./")
	if repo.Subdirectory != "" && !strings.HasPrefix(clean, repo.Subdirectory+"/") {
		return repo.Subdirectory + "/" + clean
	}
	return clean
}

func filterTargetKind(candidates []targetCandidate, kind dispatchcontract.SpecScoreTargetKind) []targetCandidate {
	return candidatesMatching(candidates, func(candidate targetCandidate) bool { return candidate.Kind == kind })
}

func candidatesMatching(candidates []targetCandidate, match func(targetCandidate) bool) []targetCandidate {
	result := make([]targetCandidate, 0)
	for _, candidate := range candidates {
		if match(candidate) {
			result = append(result, candidate)
		}
	}
	return deduplicateTargets(result)
}

func deduplicateTargets(candidates []targetCandidate) []targetCandidate {
	seen := make(map[string]struct{}, len(candidates))
	result := make([]targetCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := string(candidate.Kind) + "\x00" + candidate.ID + "\x00" + candidate.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func containsExact(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func normalizeName(value string) string {
	var builder strings.Builder
	space := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if space && builder.Len() > 0 {
				_ = builder.WriteByte(' ')
			}
			_, _ = builder.WriteRune(char)
			space = false
			continue
		}
		space = true
	}
	return builder.String()
}
