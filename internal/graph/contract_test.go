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

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// TestFrozenTopologyContract guards testdata/sample-topology.json -- the
// committed JSON shape internal/discovery engines and, from Phase 4 on, the
// frontend's hand-written TS types must match. It builds the same graph a
// K8s discovery source would report (pre-Merge, i.e. no BrandColor set) and
// asserts that running it through Merge produces JSON structurally
// identical to the fixture. If this test fails after a deliberate schema
// change, update the fixture (and the frontend types) in the same change.
func TestFrozenTopologyContract(t *testing.T) {
	fixtureBytes, err := os.ReadFile("testdata/sample-topology.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	sourceGraph := Graph{
		Nodes: []Node{
			{
				ID:     "deployment/superheros/catalog",
				Type:   "k8s-deployment",
				Label:  "catalog",
				Status: "healthy",
				Metadata: map[string]any{
					"namespace":     "superheros",
					"replicasReady": 2,
					"replicasTotal": 2,
					"image":         "example/catalog:v1",
					"generation":    1,
					"resourceKind":  "Deployment",
				},
			},
			{
				ID:     "service/superheros/catalog",
				Type:   "k8s-service",
				Label:  "catalog",
				Status: "healthy",
				Metadata: map[string]any{
					"namespace":    "superheros",
					"clusterIP":    "10.0.0.1",
					"type":         "ClusterIP",
					"resourceKind": "Service",
				},
			},
		},
		Edges: []Edge{
			{From: "service/superheros/catalog", To: "deployment/superheros/catalog", Type: "routes-to"},
		},
	}

	got := Merge(sourceGraph)

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshaling built graph: %v", err)
	}

	var gotGeneric, wantGeneric any
	if err := json.Unmarshal(gotJSON, &gotGeneric); err != nil {
		t.Fatalf("unmarshaling built graph JSON: %v", err)
	}
	if err := json.Unmarshal(fixtureBytes, &wantGeneric); err != nil {
		t.Fatalf("unmarshaling fixture: %v", err)
	}

	if !reflect.DeepEqual(gotGeneric, wantGeneric) {
		t.Errorf("built graph JSON does not match frozen fixture testdata/sample-topology.json\ngot:  %s\nwant: %s", gotJSON, fixtureBytes)
	}
}
