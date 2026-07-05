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

// Package controller holds the reconcilers for Korion's CRDs.
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	korionv1alpha1 "github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/discovery"
	"github.com/korion-io/korion/internal/graph"
)

// discoveryTimeout bounds each individual Discoverer call so one slow or
// hanging source (e.g. a GitHub API rate limit) can never block the whole
// reconcile past the 60s acceptance budget once more engines land.
const discoveryTimeout = 10 * time.Second

// defaultRefreshInterval is used when spec.refreshInterval is unset -- the
// CRD default already covers this via +kubebuilder:default, but a
// programmatically-constructed PlatformMap (e.g. in tests) may leave it
// zero.
const defaultRefreshInterval = 30 * time.Second

// PlatformMapReconciler reconciles a PlatformMap by running every
// registered Discoverer concurrently, merging their results into a single
// topology graph, and writing it to status.
type PlatformMapReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Discoverers []discovery.Discoverer
}

// +kubebuilder:rbac:groups=korion.io,resources=platformmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=korion.io,resources=platformmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices;destinationrules,verbs=get;list;watch
// +kubebuilder:rbac:groups=wgpolicyk8s.io,resources=policyreports;clusterpolicyreports,verbs=get;list;watch

func (r *PlatformMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var pm korionv1alpha1.PlatformMap
	if err := r.Get(ctx, req.NamespacedName, &pm); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !pm.Spec.AutoDiscover {
		return ctrl.Result{}, nil
	}

	results := r.runDiscoverers(ctx, &pm)

	parts := make([]graph.Graph, 0, len(results))
	for _, res := range results {
		parts = append(parts, graph.Graph{Nodes: res.Nodes, Edges: res.Edges})
		meta.SetStatusCondition(&pm.Status.Conditions, discoveryCondition(res, pm.Generation))
	}
	merged := graph.Merge(parts...)

	raw, err := json.Marshal(merged)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("marshaling topology: %w", err)
	}
	pm.Status.Topology = &runtime.RawExtension{Raw: raw}
	now := metav1.Now()
	pm.Status.LastDiscoveryTime = &now

	if err := r.Status().Update(ctx, &pm); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("discovery reconciled", "nodes", len(merged.Nodes), "edges", len(merged.Edges))

	interval := pm.Spec.RefreshInterval.Duration
	if interval <= 0 {
		interval = defaultRefreshInterval
	}
	return ctrl.Result{RequeueAfter: interval}, nil
}

// runDiscoverers runs every registered Discoverer concurrently, each under
// its own timeout. A Discoverer's error is carried in its DiscoveryResult
// rather than failing the errgroup -- one source erroring (or an optional
// tool's CRD not being installed) must never prevent the rest of the graph
// from building.
func (r *PlatformMapReconciler) runDiscoverers(ctx context.Context, pm *korionv1alpha1.PlatformMap) []discovery.DiscoveryResult {
	results := make([]discovery.DiscoveryResult, len(r.Discoverers))

	g, gctx := errgroup.WithContext(ctx)
	for i, d := range r.Discoverers {
		g.Go(func() error {
			dctx, cancel := context.WithTimeout(gctx, discoveryTimeout)
			defer cancel()
			results[i] = d.Discover(dctx, pm)
			return nil
		})
	}
	_ = g.Wait()

	return results
}

func discoveryCondition(res discovery.DiscoveryResult, generation int64) metav1.Condition {
	cond := metav1.Condition{
		Type:               res.Source + "Detected",
		ObservedGeneration: generation,
	}
	if res.Err != nil {
		cond.Status = metav1.ConditionFalse
		cond.Reason = "DiscoveryFailed"
		cond.Message = res.Err.Error()
	} else {
		cond.Status = metav1.ConditionTrue
		cond.Reason = "DiscoverySucceeded"
		cond.Message = fmt.Sprintf("%d nodes, %d edges discovered", len(res.Nodes), len(res.Edges))
	}
	return cond
}

// SetupWithManager registers this reconciler with mgr. Watching only
// PlatformMap (not the Deployments/Services it discovers) is a deliberate
// Phase 2 simplification -- staleness between discoveries is bounded by
// spec.refreshInterval instead. Watching the discovered resource types can
// be added later if that interval proves too coarse.
func (r *PlatformMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&korionv1alpha1.PlatformMap{}).
		Complete(r)
}
