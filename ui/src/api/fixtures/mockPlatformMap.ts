/**
 * Copyright 2026 The Korion Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { brandColorFor } from '../colors'
import type { GraphNode, PlatformMapView } from '../types'

// A richer, hand-built demo dataset for the SuperHeros platform (CLAUDE.md's
// canonical test case) so Phase 4 can render the *full* mockup layout --
// sidebar, canvas, ServiceDetails, DeploymentTimeline, PolicyPanel -- against
// something more representative than the two-node frozen contract fixture in
// ./sample-topology.json. Every node/edge here still conforms to the frozen
// Graph shape (internal/graph/types.go); only the volume of data is mocked
// ahead of the Phase 6 discovery engines that will eventually produce it.
const NAMESPACE = 'superheros'

function node(partial: Omit<GraphNode, 'brandColor'>): GraphNode {
  return { ...partial, brandColor: brandColorFor(partial.type) }
}

const nodes: GraphNode[] = [
  node({
    id: 'github-repository/superheros',
    type: 'github-repository',
    label: 'GitHub',
    status: 'healthy',
    metadata: { repo: 'gc-ghub/project-gc-industries-devops-superheros' },
  }),
  node({
    id: 'github-actions-workflow/superheros',
    type: 'github-actions-workflow',
    label: 'GitHub Actions',
    status: 'healthy',
    metadata: { lastRun: 'Successful', workflow: '.github/workflows/ci.yml' },
  }),
  node({
    id: 'docker-image/superheros',
    type: 'docker-image',
    label: 'Docker Hub',
    status: 'healthy',
    metadata: { image: 'gaurav/superhero-frontend:v12' },
  }),
  node({
    id: 'argocd-application/superheros-app',
    type: 'argocd-application',
    label: 'Argo CD',
    status: 'healthy',
    metadata: {
      app: 'superhero-app',
      syncStatus: 'Synced',
      healthStatus: 'Healthy',
      lastSyncCommit: 'abc123d',
    },
  }),
  node({
    id: 'service/superheros/frontend',
    type: 'k8s-service',
    label: 'frontend',
    status: 'healthy',
    metadata: { namespace: NAMESPACE, clusterIP: '10.0.0.10', type: 'ClusterIP' },
  }),
  node({
    id: 'deployment/superheros/frontend',
    type: 'k8s-deployment',
    label: 'frontend',
    status: 'healthy',
    metadata: {
      namespace: NAMESPACE,
      replicasReady: 3,
      replicasTotal: 3,
      image: 'gaurav/superhero-frontend:v12',
      argocdApp: 'superhero-app',
    },
  }),
  node({
    id: 'service/superheros/catalog',
    type: 'k8s-service',
    label: 'catalog',
    status: 'healthy',
    metadata: { namespace: NAMESPACE, clusterIP: '10.0.0.11', type: 'ClusterIP' },
  }),
  node({
    id: 'deployment/superheros/catalog-v1',
    type: 'k8s-deployment',
    label: 'catalog-v1',
    status: 'healthy',
    metadata: {
      namespace: NAMESPACE,
      replicasReady: 2,
      replicasTotal: 2,
      image: 'gaurav/superhero-catalog:v1',
      istioTrafficWeight: 60,
    },
  }),
  node({
    id: 'deployment/superheros/catalog-v2',
    type: 'k8s-deployment',
    label: 'catalog-v2',
    status: 'healthy',
    metadata: {
      namespace: NAMESPACE,
      replicasReady: 2,
      replicasTotal: 2,
      image: 'gaurav/superhero-catalog:v2',
      istioTrafficWeight: 30,
    },
  }),
  node({
    id: 'deployment/superheros/catalog-v3',
    type: 'k8s-deployment',
    label: 'catalog-v3',
    status: 'degraded',
    metadata: {
      namespace: NAMESPACE,
      replicasReady: 1,
      replicasTotal: 2,
      image: 'gaurav/superhero-catalog:v3',
      istioTrafficWeight: 10,
      podRestarts1h: 2,
    },
  }),
  node({
    id: 'service/superheros/inventory',
    type: 'k8s-service',
    label: 'inventory',
    status: 'healthy',
    metadata: { namespace: NAMESPACE, clusterIP: '10.0.0.12', type: 'ClusterIP' },
  }),
  node({
    id: 'deployment/superheros/inventory',
    type: 'k8s-deployment',
    label: 'inventory',
    status: 'healthy',
    metadata: {
      namespace: NAMESPACE,
      replicasReady: 2,
      replicasTotal: 2,
      image: 'gaurav/superhero-inventory:v4',
    },
  }),
  node({
    id: 'service/superheros/orders',
    type: 'k8s-service',
    label: 'orders',
    status: 'healthy',
    metadata: { namespace: NAMESPACE, clusterIP: '10.0.0.13', type: 'ClusterIP' },
  }),
  node({
    id: 'deployment/superheros/orders',
    type: 'k8s-deployment',
    label: 'orders',
    status: 'healthy',
    metadata: {
      namespace: NAMESPACE,
      replicasReady: 2,
      replicasTotal: 2,
      image: 'gaurav/superhero-orders:v7',
    },
  }),
  node({
    id: 'service/superheros/payment',
    type: 'k8s-service',
    label: 'payment',
    status: 'down',
    metadata: { namespace: NAMESPACE, clusterIP: '10.0.0.14', type: 'ClusterIP' },
  }),
  node({
    id: 'deployment/superheros/payment',
    type: 'k8s-deployment',
    label: 'payment',
    status: 'down',
    metadata: {
      namespace: NAMESPACE,
      replicasReady: 0,
      replicasTotal: 2,
      image: 'gaurav/superhero-payment:v3',
      podRestarts1h: 6,
    },
  }),
  node({
    id: 'istio-virtualservice/superheros/catalog',
    type: 'istio-virtualservice',
    label: 'catalog traffic split',
    status: 'healthy',
    metadata: { namespace: NAMESPACE, weights: { v1: 60, v2: 30, v3: 10 } },
  }),
  node({
    id: 'kyverno-clusterpolicyreport/superheros',
    type: 'kyverno-clusterpolicyreport',
    label: 'Kyverno',
    status: 'degraded',
    metadata: { namespace: NAMESPACE, pass: 28, warn: 3, fail: 1 },
  }),
]

export const mockPlatformMap: PlatformMapView = {
  name: 'superheros-platform',
  namespace: NAMESPACE,
  cluster: 'superhero-prod',
  topology: {
    nodes,
    edges: [
      { from: 'github-repository/superheros', to: 'github-actions-workflow/superheros', type: 'triggers' },
      { from: 'github-actions-workflow/superheros', to: 'docker-image/superheros', type: 'builds' },
      { from: 'docker-image/superheros', to: 'argocd-application/superheros-app', type: 'deploys' },
      { from: 'argocd-application/superheros-app', to: 'deployment/superheros/frontend', type: 'syncs' },
      { from: 'argocd-application/superheros-app', to: 'deployment/superheros/catalog-v1', type: 'syncs' },
      { from: 'argocd-application/superheros-app', to: 'deployment/superheros/catalog-v2', type: 'syncs' },
      { from: 'argocd-application/superheros-app', to: 'deployment/superheros/catalog-v3', type: 'syncs' },
      { from: 'argocd-application/superheros-app', to: 'deployment/superheros/inventory', type: 'syncs' },
      { from: 'argocd-application/superheros-app', to: 'deployment/superheros/orders', type: 'syncs' },
      { from: 'argocd-application/superheros-app', to: 'deployment/superheros/payment', type: 'syncs' },
      { from: 'service/superheros/frontend', to: 'deployment/superheros/frontend', type: 'routes-to' },
      { from: 'service/superheros/catalog', to: 'istio-virtualservice/superheros/catalog', type: 'routes-to' },
      { from: 'istio-virtualservice/superheros/catalog', to: 'deployment/superheros/catalog-v1', type: 'routes-to', label: '60%' },
      { from: 'istio-virtualservice/superheros/catalog', to: 'deployment/superheros/catalog-v2', type: 'routes-to', label: '30%' },
      { from: 'istio-virtualservice/superheros/catalog', to: 'deployment/superheros/catalog-v3', type: 'routes-to', label: '10%' },
      { from: 'service/superheros/inventory', to: 'deployment/superheros/inventory', type: 'routes-to' },
      { from: 'service/superheros/orders', to: 'deployment/superheros/orders', type: 'routes-to' },
      { from: 'service/superheros/payment', to: 'deployment/superheros/payment', type: 'routes-to' },
    ],
  },
  deploymentEvents: [
    { id: 'evt-1', timestamp: '10:00:15', title: 'GitHub Push', description: 'Code pushed to main branch', status: 'success', source: 'github' },
    { id: 'evt-2', timestamp: '10:00:18', title: 'GitHub Actions', description: 'Workflow started', status: 'success', source: 'github-actions' },
    { id: 'evt-3', timestamp: '10:01:42', title: 'Build & Test', description: 'Build completed successfully', status: 'success', source: 'github-actions' },
    { id: 'evt-4', timestamp: '10:02:31', title: 'Docker Push', description: 'Image pushed to Docker Hub', status: 'success', source: 'docker' },
    { id: 'evt-5', timestamp: '10:03:15', title: 'ArgoCD Sync', description: 'Changes detected and synced', status: 'success', source: 'argocd' },
    { id: 'evt-6', timestamp: '10:04:02', title: 'Deployment', description: 'payment pods crash-looping', status: 'failed', source: 'k8s' },
  ],
  policySummary: {
    total: 32,
    passed: 28,
    warnings: 3,
    failed: 1,
    violations: [
      {
        id: 'viol-1',
        policy: 'frontend deployment missing CPU limits',
        resource: 'Deployment/frontend',
        result: 'warn',
        message: 'Container "frontend" has no CPU limit set.',
        timestamp: '2m ago',
      },
      {
        id: 'viol-2',
        policy: 'Privileged container is not allowed',
        resource: 'Deployment/payment',
        result: 'fail',
        message: 'Container "payment" runs as privileged.',
        timestamp: '5m ago',
      },
      {
        id: 'viol-3',
        policy: 'Image tag should not be latest',
        resource: 'Deployment/orders',
        result: 'warn',
        message: 'Container "orders" uses the "latest" tag.',
        timestamp: '10m ago',
      },
    ],
  },
}
