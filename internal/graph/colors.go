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

// brandColors is the single, fixed lookup table of node border colors,
// transcribed from CLAUDE.md's "Tool brand colors for nodes" design rule
// (rule #7) and keyed by the exact Node.Type strings discovery engines
// produce. This is the only place brand colors are defined -- do not
// hardcode a color literal anywhere else. Extend this table (never
// duplicate it) as each Phase 6 discovery engine's Node.Type values are
// finalized.
var brandColors = map[string]string{
	// internal/discovery/k8s.go (Phase 2)
	"k8s-deployment": "#326CE5",
	"k8s-service":    "#326CE5",

	// Anticipated Phase 6 engine Type values -- adjust here if an engine's
	// actual Type string ends up different once implemented.
	"argocd-application":          "#EF7B4D",
	"istio-virtualservice":        "#466BB0",
	"istio-destinationrule":       "#466BB0",
	"kyverno-policyreport":        "#1E40AF",
	"kyverno-clusterpolicyreport": "#1E40AF",
	"github-actions-workflow":     "#3B82F6",
	"github-repository":           "#333333",
	"docker-image":                "#2496ED",
	"prometheus-metric":           "#EA580C",
	"grafana-dashboard":           "#F46800",
	"loki-log-source":             "#10B981",
}

// defaultBrandColor is used for any Node.Type not yet present in
// brandColors, so an unrecognized or not-yet-added type degrades to a
// neutral color instead of an empty border.
const defaultBrandColor = "#6B7280"

// BrandColorFor returns the fixed brand color for a node type, or
// defaultBrandColor if the type isn't in the table.
func BrandColorFor(nodeType string) string {
	if c, ok := brandColors[nodeType]; ok {
		return c
	}
	return defaultBrandColor
}
