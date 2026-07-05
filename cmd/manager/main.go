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

// Command manager is Korion's controller entrypoint: it runs the PlatformMap
// reconciler and the read-only HTTP API side by side in a single process.
package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	korionv1alpha1 "github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/api"
	"github.com/korion-io/korion/internal/controller"
	"github.com/korion-io/korion/internal/discovery"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(korionv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var apiAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&apiAddr, "api-bind-address", ":8082", "The address the read-only PlatformMap API binds to.")

	// Cluster-wide per-engine kill switches. These complement (do not replace)
	// the per-PlatformMap spec.tools.*.enabled toggles: a disabled engine here
	// is never registered at all, so no PlatformMap can trigger it regardless
	// of its spec. The Helm chart wires these from discovery.tools.*.enabled.
	// K8s core discovery is always on -- it is the baseline vertical slice.
	var enableArgoCD, enableIstio, enableKyverno, enableGitHub, enablePrometheus bool
	flag.BoolVar(&enableArgoCD, "enable-argocd", true, "Enable the ArgoCD discovery engine.")
	flag.BoolVar(&enableIstio, "enable-istio", true, "Enable the Istio discovery engine.")
	flag.BoolVar(&enableKyverno, "enable-kyverno", true, "Enable the Kyverno (PolicyReport) discovery engine.")
	flag.BoolVar(&enableGitHub, "enable-github", true, "Enable the GitHub Actions discovery engine.")
	flag.BoolVar(&enablePrometheus, "enable-prometheus", true, "Enable the Prometheus discovery engine.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create kubernetes clientset")
		os.Exit(1)
	}

	dynamicClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create dynamic client")
		os.Exit(1)
	}

	// K8s core discovery is always registered; the rest are gated by their
	// enable flags so an operator can disable an engine cluster-wide.
	discoverers := []discovery.Discoverer{&discovery.K8sDiscoverer{Clientset: clientset}}
	if enableArgoCD {
		discoverers = append(discoverers, &discovery.ArgoCDDiscoverer{Dynamic: dynamicClient, Discovery: clientset.Discovery()})
	}
	if enableIstio {
		discoverers = append(discoverers, &discovery.IstioDiscoverer{Dynamic: dynamicClient, Discovery: clientset.Discovery()})
	}
	if enableKyverno {
		discoverers = append(discoverers, &discovery.KyvernoDiscoverer{Dynamic: dynamicClient, Discovery: clientset.Discovery()})
	}
	if enableGitHub {
		discoverers = append(discoverers, &discovery.GitHubDiscoverer{Secrets: &discovery.ClientsetSecretResolver{Clientset: clientset}})
	}
	if enablePrometheus {
		discoverers = append(discoverers, &discovery.PrometheusDiscoverer{})
	}

	reconciler := &controller.PlatformMapReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Discoverers: discoverers,
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PlatformMap")
		os.Exit(1)
	}

	if err := mgr.Add(&api.Server{Client: mgr.GetClient(), Addr: apiAddr}); err != nil {
		setupLog.Error(err, "unable to add read API server")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "api-bind-address", apiAddr)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
