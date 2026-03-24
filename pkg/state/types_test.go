package state

// Features depended on: state-store

import (
	"testing"
	"time"
)

func TestSyncPolicyValues(t *testing.T) {
	tests := []struct {
		policy SyncPolicy
		want   string
	}{
		{SyncOnCommit, "on_commit"},
		{SyncOnInterval, "on_interval"},
		{SyncOnSessionEnd, "on_session_end"},
		{SyncManual, "manual"},
	}
	for _, tt := range tests {
		if string(tt.policy) != tt.want {
			t.Errorf("SyncPolicy %q != %q", tt.policy, tt.want)
		}
	}
}

func TestSyncConfigDefaults(t *testing.T) {
	var cfg SyncConfig
	if cfg.Pull != "" {
		t.Error("zero-value Pull should be empty string")
	}
	cfg = SyncConfig{
		Pull:         SyncOnCommit,
		PullInterval: 0,
		Push:         SyncOnInterval,
		PushInterval: 5 * time.Minute,
	}
	if cfg.Pull != SyncOnCommit {
		t.Error("unexpected Pull value")
	}
}
