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

package discovery

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v68/github"
	corev1 "k8s.io/api/core/v1"

	"github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/graph"
)

// SecretResolver reads a single key out of a Kubernetes Secret. It exists so
// Secret-backed engines (currently just GitHubDiscoverer) depend on a small
// interface instead of a concrete kubernetes.Interface, keeping unit tests
// free of a fake clientset -- see ClientsetSecretResolver for the real
// implementation wired up in cmd/manager/main.go.
type SecretResolver interface {
	Resolve(ctx context.Context, namespace string, ref *corev1.SecretKeySelector) (string, error)
}

// GitHubDiscoverer discovers GitHub Actions workflow status for the
// repository configured on the PlatformMap. It requires
// spec.tools.github.enabled and spec.repository to be set; a missing token
// Secret or repository is a discovery error (not silently skipped), since
// unlike ArgoCD/Istio/Kyverno, GitHub isn't an "optional CRD that may not be
// installed" case -- it's explicit user configuration.
type GitHubDiscoverer struct {
	Secrets SecretResolver

	// BaseURL overrides the GitHub API endpoint, used by tests to point at
	// an httptest.Server instead of api.github.com.
	BaseURL *url.URL
}

func (d *GitHubDiscoverer) Name() string { return "GitHub" }

func (d *GitHubDiscoverer) Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult {
	result := DiscoveryResult{Source: d.Name()}

	if pm.Spec.Tools.GitHub == nil || !pm.Spec.Tools.GitHub.Enabled {
		return result
	}

	owner, repo, err := parseGitHubRepo(pm.Spec.Repository)
	if err != nil {
		result.Err = fmt.Errorf("parsing spec.repository: %w", err)
		return result
	}

	token, err := d.resolveToken(ctx, pm)
	if err != nil {
		result.Err = fmt.Errorf("resolving github token: %w", err)
		return result
	}

	client := github.NewClient(nil).WithAuthToken(token)
	if d.BaseURL != nil {
		client.BaseURL = d.BaseURL
	}

	result.Nodes = append(result.Nodes, githubRepositoryNode(owner, repo))

	workflows, _, err := client.Actions.ListWorkflows(ctx, owner, repo, nil)
	if err != nil {
		result.Err = fmt.Errorf("listing github actions workflows for %s/%s: %w", owner, repo, err)
		return result
	}

	for _, wf := range workflows.Workflows {
		result.Nodes = append(result.Nodes, githubWorkflowNode(owner, repo, wf, d.latestRun(ctx, client, owner, repo, wf.GetID())))
	}

	return result
}

// latestRun fetches the single most recent run of a workflow, returning nil
// if none exists yet or the call fails -- a workflow with no runs (or a
// transient API error on this one workflow) still gets a node, just without
// last-run detail, rather than failing the whole engine.
func (d *GitHubDiscoverer) latestRun(ctx context.Context, client *github.Client, owner, repo string, workflowID int64) *github.WorkflowRun {
	runs, _, err := client.Actions.ListWorkflowRunsByID(ctx, owner, repo, workflowID, &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 1},
	})
	if err != nil || runs == nil || len(runs.WorkflowRuns) == 0 {
		return nil
	}
	return runs.WorkflowRuns[0]
}

func (d *GitHubDiscoverer) resolveToken(ctx context.Context, pm *v1alpha1.PlatformMap) (string, error) {
	ref := pm.Spec.Tools.GitHub.TokenSecretRef
	if ref == nil {
		return "", fmt.Errorf("spec.tools.github.tokenSecretRef is required when github discovery is enabled")
	}
	return d.Secrets.Resolve(ctx, pm.GetNamespace(), ref)
}

func githubRepositoryNode(owner, repo string) graph.Node {
	return graph.Node{
		ID:     githubRepositoryNodeID(owner, repo),
		Type:   "github-repository",
		Label:  fmt.Sprintf("%s/%s", owner, repo),
		Status: "healthy",
		Metadata: map[string]any{
			"owner":        owner,
			"repo":         repo,
			"htmlURL":      fmt.Sprintf("https://github.com/%s/%s", owner, repo),
			"resourceKind": "Repository",
		},
	}
}

func githubWorkflowNode(owner, repo string, wf *github.Workflow, run *github.WorkflowRun) graph.Node {
	node := graph.Node{
		ID:     githubWorkflowNodeID(owner, repo, wf.GetID()),
		Type:   "github-actions-workflow",
		Label:  wf.GetName(),
		Status: "unknown",
		Metadata: map[string]any{
			"owner":        owner,
			"repo":         repo,
			"path":         wf.GetPath(),
			"state":        wf.GetState(),
			"resourceKind": "Workflow",
		},
	}
	if run != nil {
		node.Status = githubConclusionToNodeStatus(run.GetConclusion(), run.GetStatus())
		node.Metadata["lastRunStatus"] = run.GetStatus()
		node.Metadata["lastRunConclusion"] = run.GetConclusion()
		node.Metadata["lastRunURL"] = run.GetHTMLURL()
		node.Metadata["lastRunHeadBranch"] = run.GetHeadBranch()
		node.Metadata["lastRunHeadSHA"] = run.GetHeadSHA()
		if run.CreatedAt != nil {
			node.Metadata["lastRunAt"] = run.GetCreatedAt().Format("2006-01-02T15:04:05Z07:00")
		}
	}
	return node
}

// githubConclusionToNodeStatus maps a workflow run's status/conclusion down
// to Korion's coarse Node.Status. An in-progress run (no conclusion yet)
// reports "unknown" rather than "degraded" -- it hasn't failed, it's just
// not done.
func githubConclusionToNodeStatus(conclusion, status string) string {
	if status != "completed" {
		return "unknown"
	}
	switch conclusion {
	case "success":
		return "healthy"
	case "failure", "timed_out", "cancelled", "action_required":
		return "degraded"
	default:
		return "unknown"
	}
}

func githubRepositoryNodeID(owner, repo string) string {
	return fmt.Sprintf("github-repository/%s/%s", owner, repo)
}

func githubWorkflowNodeID(owner, repo string, workflowID int64) string {
	return fmt.Sprintf("github-actions-workflow/%s/%s/%d", owner, repo, workflowID)
}

// parseGitHubRepo extracts "owner", "repo" from a GitHub repository URL
// (e.g. "https://github.com/gc-ghub/superheros" or with a trailing
// ".git"/"/").
func parseGitHubRepo(repoURL string) (owner, repo string, err error) {
	if repoURL == "" {
		return "", "", fmt.Errorf("spec.repository is required for github discovery")
	}
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL %q: %w", repoURL, err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository URL %q does not look like a GitHub owner/repo URL", repoURL)
	}
	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	return owner, repo, nil
}
