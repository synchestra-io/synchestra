package feature

// Features implemented: cli/feature/deps, cli/feature/refs
// Features depended on:  cli/feature

import (
	"strings"
)

// resolveTransitiveDeps follows dependency chains recursively with cycle detection.
func resolveTransitiveDeps(featuresDir, startID string) []*enrichedFeature {
	visited := map[string]bool{startID: true}
	return walkTransitive(featuresDir, startID, visited, depsResolver)
}

// resolveTransitiveRefs follows reference chains recursively with cycle detection.
func resolveTransitiveRefs(featuresDir, startID string) []*enrichedFeature {
	visited := map[string]bool{startID: true}
	return walkTransitive(featuresDir, startID, visited, refsResolver)
}

type relationResolver func(featuresDir, featureID string) ([]string, error)

func depsResolver(featuresDir, featureID string) ([]string, error) {
	return parseDependencies(featureReadmePath(featuresDir, featureID))
}

func refsResolver(featuresDir, featureID string) ([]string, error) {
	return findFeatureRefs(featuresDir, featureID)
}

func walkTransitive(featuresDir, featureID string, visited map[string]bool, resolver relationResolver) []*enrichedFeature {
	related, err := resolver(featuresDir, featureID)
	if err != nil {
		return nil
	}

	var nodes []*enrichedFeature
	for _, r := range related {
		if visited[r] {
			nodes = append(nodes, &enrichedFeature{Path: r, Cycle: boolPtr(true)})
			continue
		}
		visited[r] = true
		node := &enrichedFeature{Path: r}
		children := walkTransitive(featuresDir, r, visited, resolver)
		if len(children) > 0 {
			node.Children = children
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// enrichTransitiveNodes adds field metadata to a transitive tree.
func enrichTransitiveNodes(featuresDir string, nodes []*enrichedFeature, fields []string) {
	for _, node := range nodes {
		if node.Cycle != nil && *node.Cycle {
			continue
		}
		enrichNodeFields(featuresDir, node, fields)
		if children, ok := node.Children.([]*enrichedFeature); ok {
			enrichTransitiveNodes(featuresDir, children, fields)
		}
	}
}

// enrichNodeFields copies resolved field values into a node without overwriting its tree children.
func enrichNodeFields(featuresDir string, node *enrichedFeature, fields []string) {
	resolved := resolveFields(featuresDir, node.Path, fields)
	node.Status = resolved.Status
	node.OQ = resolved.OQ
	node.Deps = resolved.Deps
	node.Refs = resolved.Refs
	node.Plans = resolved.Plans
	node.Proposals = resolved.Proposals
}

// printTransitiveText writes transitive results as indented text.
func printTransitiveText(sb *strings.Builder, nodes []*enrichedFeature, depth int) {
	for _, node := range nodes {
		for i := 0; i < depth; i++ {
			sb.WriteByte('\t')
		}
		sb.WriteString(node.Path)
		if node.Cycle != nil && *node.Cycle {
			sb.WriteString(" (cycle)")
		}
		sb.WriteByte('\n')
		if children, ok := node.Children.([]*enrichedFeature); ok {
			printTransitiveText(sb, children, depth+1)
		}
	}
}
