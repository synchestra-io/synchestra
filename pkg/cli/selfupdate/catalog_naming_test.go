package selfupdate

// Features implemented: cli/self-update

import (
	"os"
	"regexp"
	"testing"

	"github.com/strongo/cli-helpers/cliinstall"
	"gopkg.in/yaml.v3"
)

// TestCatalogEntryMatchesGoReleaserAndMirrorWorkflow pins
// cli-install#req:catalog-identity-single-source's drift guard offline: it
// reads THIS repository's own .goreleaser.yml and
// .github/workflows/release.yml — never the cliinstall catalog entry's
// intent restated by hand — and asserts they still agree with
// cliinstall.ByID("synchestra") on every field the library uses to resolve
// a release: repository, tag prefix, and archive/checksum naming. A CLI
// that changes its GoReleaser archive or checksum naming, or its mirror
// workflow's tag prefix, without first updating the cli-helpers catalog
// entry and bumping to the release that carries it, fails HERE — this
// CLI's own CI — rather than surfacing as a broken `install synchestra`
// (or a broken `synchestra self-update`) for some other fleet CLI's user.
func TestCatalogEntryMatchesGoReleaserAndMirrorWorkflow(t *testing.T) {
	entry, ok := cliinstall.ByID(catalogID)
	if !ok {
		t.Fatalf("cliinstall has no catalog entry for id %q", catalogID)
	}

	gr := readGoReleaserConfig(t, "../../../.goreleaser.yml")

	// project_name/binary MUST equal the catalog id: the checksum
	// name_template below is "{{ .ProjectName }}_..." and the library's
	// default AssetName/ChecksumsName both start with "<binary>_", so a
	// mismatch here would silently break the default naming this entry
	// relies on (AssetName/ChecksumsName are both nil below).
	if gr.ProjectName != entry.ID {
		t.Errorf(".goreleaser.yml project_name = %q, want catalog id %q", gr.ProjectName, entry.ID)
	}

	if len(gr.Archives) == 0 {
		t.Fatal(".goreleaser.yml has no archives entry")
	}
	wantArchiveName := entry.ID + "_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
	if got := gr.Archives[0].NameTemplate; got != wantArchiveName {
		t.Errorf(".goreleaser.yml archives[0].name_template = %q, want %q (the library's default AssetName — entry.AssetName is nil, so it MUST match exactly)", got, wantArchiveName)
	}

	wantChecksumName := "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
	if got := gr.Checksum.NameTemplate; got != wantChecksumName {
		t.Errorf(".goreleaser.yml checksum.name_template = %q, want %q (the library's default ChecksumsName — entry.ChecksumsName is nil, so it MUST match exactly, given project_name equals the catalog id)", got, wantChecksumName)
	}
	if entry.AssetName != nil {
		t.Error("catalog entry overrides AssetName; this test only proves the DEFAULT naming matches .goreleaser.yml")
	}
	if entry.ChecksumsName != nil {
		t.Error("catalog entry overrides ChecksumsName; this test only proves the DEFAULT naming matches .goreleaser.yml")
	}

	// The public mirror repository and its "cli-" tag prefix are declared
	// in .github/workflows/release.yml's publish-releases job, not in
	// .goreleaser.yml (release.disable: true — GoReleaser publishes no
	// GitHub Release in THIS repository at all; see that file's own
	// comment on why the prefixed, cross-repo publish can't be
	// GoReleaser's job).
	workflow := readFile(t, "../../../.github/workflows/release.yml")

	tagPrefixRe := regexp.MustCompile(`RELEASE_TAG="([^$"]*)\$\{TAG\}"`)
	m := tagPrefixRe.FindStringSubmatch(workflow)
	if m == nil {
		t.Fatal(`.github/workflows/release.yml: no RELEASE_TAG="<prefix>${TAG}" assignment found`)
	}
	if got := m[1]; got != entry.TagPrefix {
		t.Errorf(".github/workflows/release.yml RELEASE_TAG prefix = %q, want catalog TagPrefix %q", got, entry.TagPrefix)
	}

	repoRe := regexp.MustCompile(`--repo\s+(\S+)[\s\\]*dist/\*\.tar\.gz`)
	rm := repoRe.FindStringSubmatch(workflow)
	if rm == nil {
		t.Fatal(".github/workflows/release.yml: no `gh release upload --repo <repo> dist/*.tar.gz` line found")
	}
	if got := rm[1]; got != entry.Repository {
		t.Errorf(".github/workflows/release.yml publishes to repo %q, want catalog Repository %q", got, entry.Repository)
	}
}

// goreleaserConfig is the minimal shape this test reads out of
// .goreleaser.yml — only the fields the naming comparison above needs, not
// a full schema.
type goreleaserConfig struct {
	ProjectName string `yaml:"project_name"`
	Archives    []struct {
		NameTemplate string `yaml:"name_template"`
	} `yaml:"archives"`
	Checksum struct {
		NameTemplate string `yaml:"name_template"`
	} `yaml:"checksum"`
}

func readGoReleaserConfig(t *testing.T, path string) goreleaserConfig {
	t.Helper()
	var cfg goreleaserConfig
	if err := yaml.Unmarshal([]byte(readFile(t, path)), &cfg); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return cfg
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}
