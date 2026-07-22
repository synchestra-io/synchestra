package dispatchcontract

// Features implemented: dispatch, dispatch/scheduler, dispatch/worker, model-selection

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
)

var fullGitRevision = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

var scpCloneIdentity = regexp.MustCompile(`^[^\s@/:]+@(?:\[[^\]\s]+\]|[^\s@/:]+):([^\s?#]+)$`)

func validProfile(profile ExecutionProfile) bool {
	switch profile {
	case ProfileFast, ProfileBalanced, ProfileLarge:
		return true
	default:
		return false
	}
}

// Validate checks immutable dispatch intent before persistence.
func (i DispatchIntent) Validate() error {
	if err := i.Source.Validate(); err != nil {
		return err
	}
	if err := i.Repository.Validate(); err != nil {
		return err
	}
	if err := i.Requested.Validate(); err != nil {
		return err
	}
	if i.RetryPolicy.MaxAttempts < 0 || i.RetryPolicy.BackoffSeconds < 0 {
		return fmt.Errorf("retry policy values cannot be negative")
	}
	return nil
}

// Validate checks the dispatch source tagged union.
func (s DispatchSource) Validate() error {
	switch s.Kind {
	case SourceKindAdHoc:
		if s.AdHoc == nil || s.SpecScore != nil {
			return fmt.Errorf("ad_hoc source requires only ad_hoc payload")
		}
		if strings.TrimSpace(s.AdHoc.Prompt) == "" {
			return fmt.Errorf("ad_hoc prompt is required")
		}
	case SourceKindSpecScore:
		if s.SpecScore == nil || s.AdHoc != nil {
			return fmt.Errorf("specscore source requires only specscore payload")
		}
		t := s.SpecScore
		if t.TargetKind != SpecScoreTargetPlan && t.TargetKind != SpecScoreTargetTask {
			return fmt.Errorf("invalid SpecScore target kind %q", t.TargetKind)
		}
		if t.TargetID == "" || !fullGitRevision.MatchString(t.TargetRevision) || t.SnapshotHash == "" {
			return fmt.Errorf("SpecScore target requires id, immutable revision, and snapshot hash")
		}
	default:
		return fmt.Errorf("invalid source kind %q", s.Kind)
	}
	return nil
}

// Validate rejects symbolic base revisions and credential-bearing clone URLs.
// An SSH transport username is identity metadata, not a credential: SSH keys
// and agents remain external to the persisted URL. Passwords and HTTP(S)
// userinfo are never allowed.
func (r RepositorySnapshot) Validate() error {
	if r.CanonicalID == "" || r.CloneURL == "" {
		return fmt.Errorf("repository canonical_id and clone_url are required")
	}
	if !fullGitRevision.MatchString(r.BaseRevision) {
		return fmt.Errorf("repository base_revision must be a full 40- or 64-character Git object id")
	}
	if err := validateCloneURL(r.CloneURL); err != nil {
		return err
	}
	cleanSubdirectory := path.Clean(r.Subdirectory)
	if path.IsAbs(r.Subdirectory) || cleanSubdirectory == ".." || strings.HasPrefix(cleanSubdirectory, "../") {
		return fmt.Errorf("repository subdirectory must not traverse parents")
	}
	return nil
}

func validateCloneURL(cloneURL string) error {
	// Go's URL parser treats SCP-style Git identities as opaque schemes. Accept
	// their credential-free user@host:path form explicitly and reject the
	// user:password@host:path lookalike before parsing it as an opaque URL.
	if !strings.Contains(cloneURL, "://") {
		if strings.ContainsAny(cloneURL, "?#") {
			return fmt.Errorf("repository clone_url must not contain query or fragment data")
		}
		if scpMatch := scpCloneIdentity.FindStringSubmatch(cloneURL); scpMatch != nil {
			if strings.Trim(scpMatch[1], "/") == "" {
				return fmt.Errorf("repository clone_url requires a non-root repository path")
			}
			return nil
		}
		at := strings.IndexByte(cloneURL, '@')
		if at > 0 && strings.Contains(cloneURL[:at], ":") && strings.Contains(cloneURL[at+1:], ":") {
			return fmt.Errorf("repository clone_url must not contain an inline SSH password")
		}
	}

	parsed, err := url.Parse(cloneURL)
	if err != nil {
		return fmt.Errorf("repository clone_url is invalid: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "http", "https", "ssh", "git":
	default:
		return fmt.Errorf("repository clone_url scheme %q is unsupported", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("repository clone_url requires a non-empty host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("repository clone_url must not contain query or fragment data")
	}
	if strings.Trim(parsed.Path, "/") == "" {
		return fmt.Errorf("repository clone_url requires a non-root repository path")
	}
	if parsed.User == nil {
		return nil
	}
	if scheme != "ssh" {
		return fmt.Errorf("repository clone_url must not contain HTTP(S) or other inline userinfo")
	}
	if _, hasPassword := parsed.User.Password(); hasPassword {
		return fmt.Errorf("repository clone_url must not contain an inline SSH password")
	}
	if parsed.User.Username() == "" {
		return fmt.Errorf("repository clone_url SSH username must not be empty")
	}
	return nil
}

// Validate checks requested selection and explicit fallback policy.
func (r RequestedExecution) Validate() error {
	if r.Profile != "" && !validProfile(r.Profile) {
		return fmt.Errorf("invalid execution profile %q", r.Profile)
	}
	mode := r.Fallback.Mode
	if mode == "" {
		mode = FallbackReject
	}
	switch mode {
	case FallbackReject:
		if len(r.Fallback.AllowedModels) != 0 {
			return fmt.Errorf("reject fallback cannot include allowed models")
		}
	case FallbackConfigured:
		if len(r.Fallback.AllowedModels) == 0 || strings.TrimSpace(r.Fallback.Reason) == "" {
			return fmt.Errorf("configured fallback requires allowed models and reason")
		}
	default:
		return fmt.Errorf("invalid fallback mode %q", mode)
	}
	return nil
}

// Validate checks the complete auditable routing decision.
func (r ResolvedExecution) Validate() error {
	if !validProfile(r.Profile) {
		return fmt.Errorf("resolved profile is invalid: %q", r.Profile)
	}
	if r.Agent == "" || r.Model == "" || r.MappingVersion == "" || r.RoutingReason == "" {
		return fmt.Errorf("resolved execution requires agent, model, mapping_version, and routing_reason")
	}
	return nil
}

// Validate checks a worker advertisement used for scheduler matching.
func (w WorkerCapabilities) Validate() error {
	if w.Identity.WorkerID == "" || w.Identity.HostID == "" {
		return fmt.Errorf("worker_id and host_id are required")
	}
	if w.MaxConcurrent < 1 || w.ActiveAttempts < 0 || w.ActiveAttempts > w.MaxConcurrent {
		return fmt.Errorf("invalid worker concurrency")
	}
	if !slices.Contains(w.ProtocolVersions, ProtocolVersionV1) {
		return fmt.Errorf("worker does not advertise %s", ProtocolVersionV1)
	}
	if len(w.Agents) == 0 {
		return fmt.Errorf("worker must advertise at least one agent")
	}
	for _, agent := range w.Agents {
		if agent.Agent == "" || len(agent.Profiles) == 0 || len(agent.Models) == 0 {
			return fmt.Errorf("agent capability requires agent, profiles, and models")
		}
		for _, profile := range agent.Profiles {
			if !validProfile(profile) {
				return fmt.Errorf("invalid advertised profile %q", profile)
			}
		}
	}
	return nil
}

// Validate checks a successful branch-only result.
func (r BranchResult) Validate() error {
	if r.RepositoryID == "" || r.Branch == "" || r.Summary == "" {
		return fmt.Errorf("branch result requires repository_id, branch, and summary")
	}
	if !fullGitRevision.MatchString(r.BaseRevision) || !fullGitRevision.MatchString(r.Commit) {
		return fmt.Errorf("branch result requires immutable base and commit revisions")
	}
	if !strings.HasPrefix(r.Branch, "synchestra/") {
		return fmt.Errorf("branch result must use the synchestra/ namespace")
	}
	if len(r.Validation) == 0 {
		return fmt.Errorf("branch result requires validation evidence")
	}
	for _, evidence := range r.Validation {
		if evidence.Name == "" {
			return fmt.Errorf("validation evidence name is required")
		}
		switch evidence.Status {
		case ValidationPassed, ValidationFailed, ValidationSkipped:
		default:
			return fmt.Errorf("invalid validation status %q", evidence.Status)
		}
	}
	return nil
}
