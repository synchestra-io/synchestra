// pkg/state/syncconfig_test.go
package state

import (
	"testing"
	"time"
)

func TestParseSyncPolicy(t *testing.T) {
	tests := []struct {
		input    string
		policy   SyncPolicy
		interval time.Duration
		wantErr  bool
	}{
		{"on_commit", SyncOnCommit, 0, false},
		{"on_interval=5m", SyncOnInterval, 5 * time.Minute, false},
		{"on_interval=30s", SyncOnInterval, 30 * time.Second, false},
		{"on_session_end", SyncOnSessionEnd, 0, false},
		{"manual", SyncManual, 0, false},
		{"on_interval", "", 0, true}, // missing duration
		{"unknown", "", 0, true},
		{"", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			policy, interval, err := ParseSyncPolicy(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}
			if policy != tt.policy {
				t.Errorf("policy = %q, want %q", policy, tt.policy)
			}
			if interval != tt.interval {
				t.Errorf("interval = %v, want %v", interval, tt.interval)
			}
		})
	}
}
