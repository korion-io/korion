/*
Copyright 2026 The Korion Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package graph

// Merge combines the Nodes and Edges of any number of partial Graphs (one
// per discovery source) into a single Graph, joining nodes by their stable
// ID so that two sources describing the same entity (e.g. K8s discovery and
// ArgoCD discovery both touching the same Deployment, once ArgoCD discovery
// lands in Phase 6) enrich one node instead of producing duplicate canvas
// nodes or one source's detail silently erasing another's (see joinNodes).
// Every merged node has BrandColor stamped from the single BrandColorFor
// table -- discoverers never set it themselves. Duplicate edges (same
// From/To/Type/Label reported by more than one source) collapse to one.
// Nil/empty parts are tolerated so a source that failed or found nothing
// never blocks the rest of the graph from building.
func Merge(parts ...Graph) Graph {
	order := make([]string, 0)
	byID := make(map[string]Node)

	for _, part := range parts {
		for _, n := range part.Nodes {
			existing, exists := byID[n.ID]
			if !exists {
				order = append(order, n.ID)
				byID[n.ID] = n
				continue
			}
			byID[n.ID] = joinNodes(existing, n)
		}
	}

	merged := Graph{
		Nodes: make([]Node, 0, len(order)),
		Edges: dedupeEdges(parts),
	}
	for _, id := range order {
		node := byID[id]
		node.BrandColor = BrandColorFor(node.Type)
		merged.Nodes = append(merged.Nodes, node)
	}

	return merged
}

// joinNodes merges a later discovery source's view of an already-seen node
// (matched by stable ID) into the existing one. Scalar fields (Type, Label,
// Status) are replaced only when the later source provides a non-empty
// value, and Metadata is shallow-merged key by key -- this is what lets a
// source that only contributes partial detail (e.g. ArgoCD adding sync
// status to a node K8s discovery already produced) enrich a node instead of
// erasing fields an earlier source already set.
func joinNodes(existing, next Node) Node {
	joined := existing
	if next.Type != "" {
		joined.Type = next.Type
	}
	if next.Label != "" {
		joined.Label = next.Label
	}
	if next.Status != "" {
		joined.Status = next.Status
	}
	if len(next.Metadata) > 0 {
		merged := make(map[string]any, len(existing.Metadata)+len(next.Metadata))
		for k, v := range existing.Metadata {
			merged[k] = v
		}
		for k, v := range next.Metadata {
			merged[k] = v
		}
		joined.Metadata = merged
	}
	return joined
}

// dedupeEdges flattens every part's edges into one slice, collapsing exact
// duplicates (same From/To/Type/Label) reported by more than one source
// down to a single edge, preserving first-seen order.
func dedupeEdges(parts []Graph) []Edge {
	type key struct{ from, to, typ, label string }
	seen := make(map[key]bool)
	edges := make([]Edge, 0)

	for _, part := range parts {
		for _, e := range part.Edges {
			k := key{e.From, e.To, e.Type, e.Label}
			if seen[k] {
				continue
			}
			seen[k] = true
			edges = append(edges, e)
		}
	}

	return edges
}
