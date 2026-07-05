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
// per discovery source) into a single Graph, deduping nodes by ID so that
// two sources describing the same entity (e.g. K8s discovery and ArgoCD
// discovery both touching the same Deployment, once ArgoCD discovery lands
// in Phase 6) don't produce duplicate canvas nodes. Later parts win on
// conflicting fields. Nil/empty parts are tolerated so a source that failed
// or found nothing never blocks the rest of the graph from building.
func Merge(parts ...Graph) Graph {
	order := make([]string, 0)
	byID := make(map[string]Node)

	for _, part := range parts {
		for _, n := range part.Nodes {
			if _, exists := byID[n.ID]; !exists {
				order = append(order, n.ID)
			}
			byID[n.ID] = n
		}
	}

	merged := Graph{
		Nodes: make([]Node, 0, len(order)),
		Edges: make([]Edge, 0),
	}
	for _, id := range order {
		merged.Nodes = append(merged.Nodes, byID[id])
	}
	for _, part := range parts {
		merged.Edges = append(merged.Edges, part.Edges...)
	}

	return merged
}
