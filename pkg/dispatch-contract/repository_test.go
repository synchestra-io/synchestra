package dispatchcontract_test

// Features implemented: dispatch, dispatch/scheduler, dispatch/worker

import (
	"strings"
	"testing"

	dispatchcontract "github.com/synchestra-io/synchestra/pkg/dispatch-contract"
)

func TestRepositorySnapshotAcceptsCredentialFreeCloneIdentities(t *testing.T) {
	t.Parallel()

	tests := []string{
		"https://github.com/example/repo.git",
		"http://localhost:8080/example/repo.git",
		"ssh://git@github.com/example/repo.git",
		"ssh://deploy@git.example.test:2222/group/repo.git",
		"git@github.com:example/repo.git",
		"deploy@git.example.test:group/repo.git",
		"git://git.example.test/group/repo.git",
	}
	for _, cloneURL := range tests {
		cloneURL := cloneURL
		t.Run(cloneURL, func(t *testing.T) {
			t.Parallel()
			repository := dispatchcontract.RepositorySnapshot{
				CanonicalID:  "git.example.test/group/repo",
				CloneURL:     cloneURL,
				BaseRevision: baseRevision,
			}
			if err := repository.Validate(); err != nil {
				t.Fatalf("credential-free clone identity rejected: %v", err)
			}
		})
	}
}

func TestRepositorySnapshotRejectsInlineCloneCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cloneURL string
		want     string
	}{
		{cloneURL: "https://token@github.com/example/repo.git", want: "userinfo"},
		{cloneURL: "https://user:password@github.com/example/repo.git", want: "userinfo"},
		{cloneURL: "http://user@localhost:8080/example/repo.git", want: "userinfo"},
		{cloneURL: "ssh://git:password@github.com/example/repo.git", want: "SSH password"},
		{cloneURL: "ssh://git:@github.com/example/repo.git", want: "SSH password"},
		{cloneURL: "git:password@github.com:example/repo.git", want: "SSH password"},
		{cloneURL: "git://user@git.example.test/group/repo.git", want: "userinfo"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.cloneURL, func(t *testing.T) {
			t.Parallel()
			repository := dispatchcontract.RepositorySnapshot{
				CanonicalID:  "git.example.test/group/repo",
				CloneURL:     test.cloneURL,
				BaseRevision: baseRevision,
			}
			err := repository.Validate()
			if err == nil {
				t.Fatal("credential-bearing clone URL accepted")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %q, want substring %q", err, test.want)
			}
		})
	}
}

func TestRepositorySnapshotRejectsUnsupportedCloneIdentities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cloneURL string
		want     string
	}{
		{cloneURL: "/tmp/repo.git", want: "scheme"},
		{cloneURL: "../repo.git", want: "scheme"},
		{cloneURL: "github.com/example/repo.git", want: "scheme"},
		{cloneURL: "github.com:example/repo.git", want: "scheme"},
		{cloneURL: "file:///tmp/repo.git", want: "unsupported"},
		{cloneURL: "ftp://git.example.test/group/repo.git", want: "unsupported"},
		{cloneURL: "https:///group/repo.git", want: "host"},
		{cloneURL: "ssh:///group/repo.git", want: "host"},
		{cloneURL: "git:///group/repo.git", want: "host"},
		{cloneURL: "host.example.test:group/repo.git", want: "unsupported"},
		{cloneURL: "git @host.example.test:group/repo.git", want: "invalid"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.cloneURL, func(t *testing.T) {
			t.Parallel()
			repository := dispatchcontract.RepositorySnapshot{
				CanonicalID:  "git.example.test/group/repo",
				CloneURL:     test.cloneURL,
				BaseRevision: baseRevision,
			}
			err := repository.Validate()
			if err == nil {
				t.Fatal("unsupported clone identity accepted")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %q, want substring %q", err, test.want)
			}
		})
	}
}
