package state

// Features implemented: state-store
// Features depended on:  project-definition

import (
	"fmt"
	"strings"
	"time"
)

// ParseSyncPolicy parses a sync policy string value from configuration.
// Handles plain values ("on_commit", "manual") and parameterized values
// ("on_interval=5m").
func ParseSyncPolicy(s string) (SyncPolicy, time.Duration, error) {
	switch {
	case s == string(SyncOnCommit):
		return SyncOnCommit, 0, nil
	case s == string(SyncOnSessionEnd):
		return SyncOnSessionEnd, 0, nil
	case s == string(SyncManual):
		return SyncManual, 0, nil
	case strings.HasPrefix(s, string(SyncOnInterval)+"="):
		durStr := strings.TrimPrefix(s, string(SyncOnInterval)+"=")
		d, err := time.ParseDuration(durStr)
		if err != nil {
			return "", 0, fmt.Errorf("invalid interval duration %q: %w", durStr, err)
		}
		return SyncOnInterval, d, nil
	case s == string(SyncOnInterval):
		return "", 0, fmt.Errorf("on_interval requires a duration (e.g., on_interval=5m)")
	default:
		return "", 0, fmt.Errorf("unknown sync policy %q", s)
	}
}
