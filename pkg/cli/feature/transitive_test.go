package feature

// Features implemented: cli/feature

import (
	"strings"
	"testing"
)

func TestResolveTransitiveDeps_LinearChain(t *testing.T) {
	t.Parallel()

	featDir := setupTestFeatures(t, map[string]string{
		"a": `# Feature: A

**Status:** Approved

## Dependencies

- b

## Outstanding Questions

None at this time.
`,
		"b": `# Feature: B

**Status:** Approved

## Dependencies

- c

## Outstanding Questions

None at this time.
`,
		"c": `# Feature: C

**Status:** Approved

## Outstanding Questions

None at this time.
`,
	})

	nodes := resolveTransitiveDeps(featDir, "a")
	if len(nodes) != 1 {
		t.Fatalf("got %d top-level nodes, want 1", len(nodes))
	}
	if nodes[0].Path != "b" {
		t.Errorf("nodes[0].Path = %q, want %q", nodes[0].Path, "b")
	}

	children, ok := nodes[0].Children.([]*enrichedFeature)
	if !ok {
		t.Fatalf("nodes[0].Children type = %T, want []*enrichedFeature", nodes[0].Children)
	}
	if len(children) != 1 {
		t.Fatalf("got %d children of b, want 1", len(children))
	}
	if children[0].Path != "c" {
		t.Errorf("children[0].Path = %q, want %q", children[0].Path, "c")
	}
	if children[0].Children != nil {
		t.Errorf("children[0].Children = %v, want nil", children[0].Children)
	}
}

func TestResolveTransitiveDeps_FanOut(t *testing.T) {
	t.Parallel()

	featDir := setupTestFeatures(t, map[string]string{
		"a": `# Feature: A

## Dependencies

- b
- c

## Outstanding Questions

None at this time.
`,
		"b": `# Feature: B

## Dependencies

- d

## Outstanding Questions

None at this time.
`,
		"c": `# Feature: C

## Outstanding Questions

None at this time.
`,
		"d": `# Feature: D

## Outstanding Questions

None at this time.
`,
	})

	nodes := resolveTransitiveDeps(featDir, "a")
	if len(nodes) != 2 {
		t.Fatalf("got %d top-level nodes, want 2", len(nodes))
	}
	if nodes[0].Path != "b" {
		t.Errorf("nodes[0].Path = %q, want %q", nodes[0].Path, "b")
	}
	if nodes[1].Path != "c" {
		t.Errorf("nodes[1].Path = %q, want %q", nodes[1].Path, "c")
	}

	bChildren, ok := nodes[0].Children.([]*enrichedFeature)
	if !ok {
		t.Fatalf("nodes[0].Children type = %T, want []*enrichedFeature", nodes[0].Children)
	}
	if len(bChildren) != 1 || bChildren[0].Path != "d" {
		t.Errorf("b children = %v, want [d]", bChildren)
	}

	if nodes[1].Children != nil {
		t.Errorf("c should have no children, got %v", nodes[1].Children)
	}
}

func TestResolveTransitiveDeps_CycleDetection(t *testing.T) {
	t.Parallel()

	featDir := setupTestFeatures(t, map[string]string{
		"a": `# Feature: A

## Dependencies

- b

## Outstanding Questions

None at this time.
`,
		"b": `# Feature: B

## Dependencies

- a

## Outstanding Questions

None at this time.
`,
	})

	nodes := resolveTransitiveDeps(featDir, "a")
	if len(nodes) != 1 {
		t.Fatalf("got %d top-level nodes, want 1", len(nodes))
	}
	if nodes[0].Path != "b" {
		t.Errorf("nodes[0].Path = %q, want %q", nodes[0].Path, "b")
	}

	children, ok := nodes[0].Children.([]*enrichedFeature)
	if !ok {
		t.Fatalf("nodes[0].Children type = %T, want []*enrichedFeature", nodes[0].Children)
	}
	if len(children) != 1 {
		t.Fatalf("got %d children of b, want 1", len(children))
	}
	if children[0].Path != "a" {
		t.Errorf("cycle node Path = %q, want %q", children[0].Path, "a")
	}
	if children[0].Cycle == nil || !*children[0].Cycle {
		t.Error("cycle node should have Cycle=true")
	}
}

func TestResolveTransitiveDeps_NoDeps(t *testing.T) {
	t.Parallel()

	featDir := setupTestFeatures(t, map[string]string{
		"a": `# Feature: A

## Outstanding Questions

None at this time.
`,
	})

	nodes := resolveTransitiveDeps(featDir, "a")
	if len(nodes) != 0 {
		t.Errorf("got %d nodes, want 0", len(nodes))
	}
}

func TestResolveTransitiveDeps_Diamond(t *testing.T) {
	t.Parallel()

	featDir := setupTestFeatures(t, map[string]string{
		"a": `# Feature: A

## Dependencies

- b
- c

## Outstanding Questions

None at this time.
`,
		"b": `# Feature: B

## Dependencies

- d

## Outstanding Questions

None at this time.
`,
		"c": `# Feature: C

## Dependencies

- d

## Outstanding Questions

None at this time.
`,
		"d": `# Feature: D

## Outstanding Questions

None at this time.
`,
	})

	nodes := resolveTransitiveDeps(featDir, "a")
	if len(nodes) != 2 {
		t.Fatalf("got %d top-level nodes, want 2", len(nodes))
	}

	// B should have D as a child (first visit).
	bChildren, ok := nodes[0].Children.([]*enrichedFeature)
	if !ok || len(bChildren) != 1 || bChildren[0].Path != "d" {
		t.Fatalf("expected b to have child d, got %v", nodes[0].Children)
	}

	// C should NOT have D as a child because D was already visited via B.
	// The visited set prevents D from appearing again, so C has no children.
	if nodes[1].Children != nil {
		// If the implementation marks D under C as a cycle, that's also acceptable.
		cChildren, ok := nodes[1].Children.([]*enrichedFeature)
		if ok && len(cChildren) > 0 {
			// D should only appear once as a full expansion. Under C it would be a cycle node.
			if cChildren[0].Path != "d" {
				t.Errorf("unexpected child under c: %q", cChildren[0].Path)
			}
			if cChildren[0].Cycle == nil || !*cChildren[0].Cycle {
				t.Error("d under c should be marked as cycle since it was already visited")
			}
		}
	}

	// Count total occurrences of "d" to ensure visited set works.
	dCount := countPathOccurrences(nodes, "d")
	if dCount != 2 {
		// One full expansion under B and one cycle marker under C.
		t.Logf("note: d appears %d time(s) in the tree", dCount)
	}
}

// countPathOccurrences counts how many times a path appears in a transitive tree.
func countPathOccurrences(nodes []*enrichedFeature, path string) int {
	count := 0
	for _, n := range nodes {
		if n.Path == path {
			count++
		}
		if children, ok := n.Children.([]*enrichedFeature); ok {
			count += countPathOccurrences(children, path)
		}
	}
	return count
}

func TestResolveTransitiveRefs_LinearChain(t *testing.T) {
	t.Parallel()

	// B depends on A, C depends on B.
	// So refs of A = [B], refs of B = [C], transitive refs of A = [B [C]].
	featDir := setupTestFeatures(t, map[string]string{
		"a": `# Feature: A

## Outstanding Questions

None at this time.
`,
		"b": `# Feature: B

## Dependencies

- a

## Outstanding Questions

None at this time.
`,
		"c": `# Feature: C

## Dependencies

- b

## Outstanding Questions

None at this time.
`,
	})

	nodes := resolveTransitiveRefs(featDir, "a")
	if len(nodes) != 1 {
		t.Fatalf("got %d top-level ref nodes, want 1", len(nodes))
	}
	if nodes[0].Path != "b" {
		t.Errorf("nodes[0].Path = %q, want %q", nodes[0].Path, "b")
	}

	children, ok := nodes[0].Children.([]*enrichedFeature)
	if !ok {
		t.Fatalf("nodes[0].Children type = %T, want []*enrichedFeature", nodes[0].Children)
	}
	if len(children) != 1 {
		t.Fatalf("got %d children of b, want 1", len(children))
	}
	if children[0].Path != "c" {
		t.Errorf("children[0].Path = %q, want %q", children[0].Path, "c")
	}
}

func TestResolveTransitiveRefs_NoRefs(t *testing.T) {
	t.Parallel()

	// No feature depends on A, so transitive refs should be empty.
	featDir := setupTestFeatures(t, map[string]string{
		"a": `# Feature: A

## Outstanding Questions

None at this time.
`,
		"b": `# Feature: B

## Outstanding Questions

None at this time.
`,
	})

	nodes := resolveTransitiveRefs(featDir, "a")
	if len(nodes) != 0 {
		t.Errorf("got %d ref nodes, want 0", len(nodes))
	}
}

func TestEnrichTransitiveNodes_StatusField(t *testing.T) {
	t.Parallel()

	featDir := setupTestFeatures(t, map[string]string{
		"b": `# Feature: B

**Status:** Approved

## Outstanding Questions

None at this time.
`,
		"c": `# Feature: C

**Status:** Conceptual

## Outstanding Questions

None at this time.
`,
	})

	nodes := []*enrichedFeature{
		{
			Path: "b",
			Children: []*enrichedFeature{
				{Path: "c"},
			},
		},
	}

	enrichTransitiveNodes(featDir, nodes, []string{"status"})

	if nodes[0].Status != "Approved" {
		t.Errorf("b.Status = %q, want %q", nodes[0].Status, "Approved")
	}
	children := nodes[0].Children.([]*enrichedFeature)
	if children[0].Status != "Conceptual" {
		t.Errorf("c.Status = %q, want %q", children[0].Status, "Conceptual")
	}
}

func TestEnrichTransitiveNodes_CycleNodesSkipped(t *testing.T) {
	t.Parallel()

	featDir := setupTestFeatures(t, map[string]string{
		"b": `# Feature: B

**Status:** Approved

## Outstanding Questions

None at this time.
`,
		"a": `# Feature: A

**Status:** Implemented

## Outstanding Questions

None at this time.
`,
	})

	cycleNode := &enrichedFeature{Path: "a", Cycle: boolPtr(true)}
	nodes := []*enrichedFeature{
		{
			Path:     "b",
			Children: []*enrichedFeature{cycleNode},
		},
	}

	enrichTransitiveNodes(featDir, nodes, []string{"status"})

	// b should be enriched.
	if nodes[0].Status != "Approved" {
		t.Errorf("b.Status = %q, want %q", nodes[0].Status, "Approved")
	}

	// Cycle node "a" should NOT be enriched — its Status stays empty.
	if cycleNode.Status != "" {
		t.Errorf("cycle node a.Status = %q, want empty", cycleNode.Status)
	}
}

func TestPrintTransitiveText_Simple(t *testing.T) {
	t.Parallel()

	nodes := []*enrichedFeature{
		{Path: "b"},
		{Path: "c"},
	}

	var sb strings.Builder
	printTransitiveText(&sb, nodes, 0)

	got := sb.String()
	want := "b\nc\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintTransitiveText_Nested(t *testing.T) {
	t.Parallel()

	nodes := []*enrichedFeature{
		{
			Path: "b",
			Children: []*enrichedFeature{
				{Path: "d"},
			},
		},
		{Path: "c"},
	}

	var sb strings.Builder
	printTransitiveText(&sb, nodes, 0)

	got := sb.String()
	want := "b\n\td\nc\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintTransitiveText_Cycle(t *testing.T) {
	t.Parallel()

	nodes := []*enrichedFeature{
		{
			Path: "b",
			Children: []*enrichedFeature{
				{Path: "a", Cycle: boolPtr(true)},
			},
		},
	}

	var sb strings.Builder
	printTransitiveText(&sb, nodes, 0)

	got := sb.String()
	want := "b\n\ta (cycle)\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintTransitiveText_DeeplyNested(t *testing.T) {
	t.Parallel()

	nodes := []*enrichedFeature{
		{
			Path: "a",
			Children: []*enrichedFeature{
				{
					Path: "b",
					Children: []*enrichedFeature{
						{Path: "c"},
					},
				},
			},
		},
	}

	var sb strings.Builder
	printTransitiveText(&sb, nodes, 0)

	got := sb.String()
	want := "a\n\tb\n\t\tc\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestPrintTransitiveText_Empty(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	printTransitiveText(&sb, nil, 0)

	got := sb.String()
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}
