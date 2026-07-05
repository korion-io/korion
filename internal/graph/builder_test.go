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

import "testing"

func TestMerge(t *testing.T) {
	tests := []struct {
		name      string
		parts     []Graph
		wantNodes []string // expected Node IDs, in expected order
		wantEdges int
	}{
		{
			name:      "empty input",
			parts:     nil,
			wantNodes: []string{},
			wantEdges: 0,
		},
		{
			name: "single source",
			parts: []Graph{
				{
					Nodes: []Node{{ID: "a"}, {ID: "b"}},
					Edges: []Edge{{From: "a", To: "b"}},
				},
			},
			wantNodes: []string{"a", "b"},
			wantEdges: 1,
		},
		{
			name: "multi-source merge dedupes by ID, later wins",
			parts: []Graph{
				{Nodes: []Node{{ID: "a", Status: "unknown"}}},
				{Nodes: []Node{{ID: "a", Status: "healthy"}, {ID: "b"}}},
			},
			wantNodes: []string{"a", "b"},
			wantEdges: 0,
		},
		{
			name: "erroring source contributes nothing but doesn't block the rest",
			parts: []Graph{
				{},
				{Nodes: []Node{{ID: "a"}}, Edges: []Edge{{From: "a", To: "a"}}},
			},
			wantNodes: []string{"a"},
			wantEdges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.parts...)
			if len(got.Nodes) != len(tt.wantNodes) {
				t.Fatalf("got %d nodes, want %d (%v)", len(got.Nodes), len(tt.wantNodes), got.Nodes)
			}
			for i, id := range tt.wantNodes {
				if got.Nodes[i].ID != id {
					t.Errorf("node[%d].ID = %q, want %q", i, got.Nodes[i].ID, id)
				}
			}
			if len(got.Edges) != tt.wantEdges {
				t.Errorf("got %d edges, want %d", len(got.Edges), tt.wantEdges)
			}
		})
	}

	t.Run("later source wins on conflicting fields", func(t *testing.T) {
		got := Merge(
			Graph{Nodes: []Node{{ID: "a", Status: "unknown"}}},
			Graph{Nodes: []Node{{ID: "a", Status: "healthy"}}},
		)
		if got.Nodes[0].Status != "healthy" {
			t.Errorf("Status = %q, want %q", got.Nodes[0].Status, "healthy")
		}
	})

	t.Run("every merged node gets BrandColor stamped from the lookup table", func(t *testing.T) {
		got := Merge(Graph{Nodes: []Node{
			{ID: "d", Type: "k8s-deployment"},
			{ID: "x", Type: "totally-unknown-type"},
		}})

		byID := make(map[string]Node)
		for _, n := range got.Nodes {
			byID[n.ID] = n
		}
		if c := byID["d"].BrandColor; c != "#326CE5" {
			t.Errorf("k8s-deployment BrandColor = %q, want %q", c, "#326CE5")
		}
		if c := byID["x"].BrandColor; c != defaultBrandColor {
			t.Errorf("unknown-type BrandColor = %q, want fallback %q", c, defaultBrandColor)
		}
	})

	t.Run("a later source's partial metadata enriches rather than erases an earlier source's fields", func(t *testing.T) {
		got := Merge(
			Graph{Nodes: []Node{{
				ID:   "svc/catalog",
				Type: "k8s-service",
				Metadata: map[string]any{
					"namespace": "superheros",
					"clusterIP": "10.0.0.1",
				},
			}}},
			// A hypothetical second source contributing only sync status --
			// must not wipe out the namespace/clusterIP the first source set.
			Graph{Nodes: []Node{{
				ID: "svc/catalog",
				Metadata: map[string]any{
					"syncStatus": "Synced",
				},
			}}},
		)

		if len(got.Nodes) != 1 {
			t.Fatalf("got %d nodes, want 1 (joined by ID)", len(got.Nodes))
		}
		md := got.Nodes[0].Metadata
		if md["namespace"] != "superheros" || md["clusterIP"] != "10.0.0.1" || md["syncStatus"] != "Synced" {
			t.Errorf("expected joined metadata from both sources, got %+v", md)
		}
	})

	t.Run("identical edges from multiple sources collapse to one", func(t *testing.T) {
		dup := Edge{From: "a", To: "b", Type: "routes-to"}
		got := Merge(
			Graph{Nodes: []Node{{ID: "a"}, {ID: "b"}}, Edges: []Edge{dup}},
			Graph{Edges: []Edge{dup}},
		)
		if len(got.Edges) != 1 {
			t.Errorf("got %d edges, want 1 deduped edge: %+v", len(got.Edges), got.Edges)
		}
	})
}
