# Korion — K8s-Native Platform Topology Engine

## What Korion Is

Korion is an open-source, Kubernetes-native platform topology engine. It auto-discovers
the complete DevOps stack of any deployed application and renders it as an interactive
live diagram — from developer commit to running pod — in one unified dashboard.

Engineers install Korion into their cluster the same way they install ArgoCD or Kyverno:
apply a CRD, and Korion discovers and maps everything automatically. No manual catalog
files. No configuration. Zero setup beyond the CRD.

**The problem it solves:** Engineers juggle 6+ separate tools simultaneously — GitHub
Actions, ArgoCD, Kiali, Grafana, Prometheus, Kyverno — to answer the question "what is
happening on my platform right now?" Korion replaces all of those open tabs with one
interactive view that auto-discovers the complete pipeline and shows live health on every
node.

**Primary differentiators:**
- Zero-configuration auto-discovery (unlike Backstage which requires catalog-info.yaml)
- Full pipeline visibility: GitHub Actions → Docker → ArgoCD → K8s → Istio (unlike
  Karpor which only shows K8s resources)
- Interactive topology canvas where every node links to the live tool dashboard
- System-specific AI via ARIA — knows YOUR topology, not generic K8s advice
- K8s-native operator — install with Helm, manage with kubectl, GitOps-compatible

**License:** Apache 2.0. This is non-negotiable and must appear in every source file.

**GitHub org:** korion-io (separate from personal gc-ghub account — CNCF requirement)

**CNCF target:** Sandbox. Keep this in mind for all architecture decisions — the project
must be governance-ready, multi-cloud capable, and clearly differentiated from kagent
(generic agent framework) and Karpor (K8s resource explorer).

---

## Core Architecture

Korion ships two CRDs managed by a single controller binary.

### CRD 1: PlatformMap (Discovery + Visualisation)

The PlatformMap CRD triggers auto-discovery when applied. The controller queries:

| Source | What it discovers |
|--------|------------------|
| K8s API | Namespaces, Deployments, Services, Pods, installed CRDs |
| ArgoCD API | Application resources, sync state, last commit SHA, drift |
| GitHub API | Workflow files, last CI run status per service |
| Istio API | VirtualServices, DestinationRules, traffic weights per version |
| Kyverno API | ClusterPolicyReport, violation counts and details |
| Prometheus API | Error rate, latency p99, CPU/memory per service |
| Loki API | Recent error log patterns per service (last 30 min) |

The controller builds a directed graph of nodes and edges, storing it in the PlatformMap
`.status.topology` field. The frontend renders this graph as a React Flow canvas.

**Sample PlatformMap:**
```yaml
apiVersion: korion.io/v1alpha1
kind: PlatformMap
metadata:
  name: superheros-platform
  namespace: superheros
spec:
  repository: https://github.com/gc-ghub/project-gc-industries-devops-superheros
  namespace: superheros
  autoDiscover: true
  tools:
    argocd:
      enabled: true
      url: ""                   # auto-detected if empty
    kiali:
      enabled: true
    prometheus:
      enabled: true
    loki:
      enabled: true
    github:
      enabled: true
      tokenSecretRef:
        name: korion-github-secret
        key: token
  refreshInterval: 30s
```

### CRD 2: PlatformAgent (ARIA — Autonomous Remediation and Intelligence Agent)

ARIA is the AI intelligence layer. It references a PlatformMap and gains full system
context before every LLM call. ARIA is NEVER generic — it knows the specific topology
of the platform it monitors.

**Sample PlatformAgent:**
```yaml
apiVersion: korion.io/v1alpha1
kind: PlatformAgent
metadata:
  name: aria-superheros
  namespace: superheros
spec:
  platformMap: superheros-platform
  autonomyLevel: suggest              # suggest | approve | auto
  llmProvider:
    provider: anthropic               # anthropic | openai | ollama
    model: claude-sonnet-4-6
    apiKeySecretRef:
      name: aria-llm-secret
      key: api-key
  features:
    alertEnrichment:
      enabled: true
      slackWebhookSecretRef:
        name: aria-slack-secret
        key: webhook-url
    healthAdvisor:
      enabled: true
      schedule: "0 6 * * *"
    sreDiagnostics:
      enabled: true
    canaryDecision:
      enabled: true
  runbooks:
    - name: crashloopbackoff
      trigger: "podRestarts > 3"
      action: diagnose-and-suggest
    - name: high-error-rate
      trigger: "errorRate > 5"
      action: diagnose-and-suggest
    - name: argocd-drift
      trigger: "syncStatus == OutOfSync"
      action: auto-sync-after-validation
```

**ARIA capabilities:**
1. **Alert enrichment** — Alertmanager webhook → assemble platform context → Claude →
   structured SRE analysis (root cause, explanation, kubectl commands, prevention) → Slack
2. **Health advisor** — scheduled scan → platform health report (critical findings,
   warnings, cost observations, architecture recommendations) → Slack + dashboard
3. **SRE diagnostics** — manual or triggered investigation → full context snapshot →
   diagnosis with confidence score + specific remediation steps
4. **Canary decision** — queries Istio metrics for all versions → recommends
   promote/hold/rollback with specific reasoning based on error rates and latency
5. **Learning store** — records every incident, fix applied, and outcome; ARIA's
   recommendations improve over time as it accumulates system history

---

## Technology Stack

### Controller (Go)
- Go 1.22+
- controller-runtime v0.17+ for operator scaffolding
- client-go for K8s API access
- kubebuilder markers for CRD generation
- Standard project layout: cmd/, internal/, api/

### ARIA (Python)
- Python 3.12+
- FastAPI for alert webhook endpoints
- anthropic Python SDK (primary LLM provider)
- kubernetes Python client for cluster queries
- httpx for async HTTP (Prometheus, Loki, ArgoCD, GitHub APIs)
- PostgreSQL via asyncpg for learning store
- pydantic for data validation and context schema

### Frontend (TypeScript + React)
- React 18 + TypeScript
- React Flow for the interactive topology canvas
- Tailwind CSS for styling
- Dark theme ONLY — do not add light mode
  - Background: #040912
  - Primary accent: #00C8FF (Korion cyan)
  - Secondary: #8B5CF6 (Korion violet)
  - Node borders use tool brand colors (ArgoCD #EF7B4D, K8s #326CE5, etc.)
- Vite for build tooling

### Packaging
- Helm chart (helm/korion/)
- Installs into korion-system namespace
- Single helm install deploys controller + ARIA + frontend

---

## Repository Structure

```
korion/
├── CLAUDE.md                         # This file — always read first
├── LICENSE                           # Apache 2.0
├── README.md                         # Overview + mockup screenshots
├── GOVERNANCE.md                     # Open governance (CNCF requirement)
├── SECURITY.md                       # Security disclosure process (CNCF requirement)
├── Makefile                          # All build/test/generate targets
│
├── api/
│   └── v1alpha1/
│       ├── platformmap_types.go      # PlatformMap CRD types
│       ├── platformagent_types.go    # PlatformAgent CRD types
│       └── groupversion_info.go     # API group: korion.io
│
├── cmd/
│   └── manager/
│       └── main.go                  # Controller entrypoint
│
├── internal/
│   ├── controller/
│   │   ├── platformmap_controller.go
│   │   └── platformagent_controller.go
│   ├── discovery/                   # Auto-discovery engines (one file per source)
│   │   ├── k8s.go                   # K8s API discovery
│   │   ├── argocd.go                # ArgoCD Application discovery
│   │   ├── github.go                # GitHub Actions workflow discovery
│   │   ├── istio.go                 # Istio mesh discovery
│   │   ├── kyverno.go               # Kyverno policy report discovery
│   │   └── prometheus.go            # Prometheus metrics discovery
│   └── graph/
│       ├── builder.go               # Assembles topology graph from discoveries
│       ├── types.go                 # Node and Edge type definitions
│       └── builder_test.go          # Unit tests for graph construction
│
├── aria/                            # ARIA Python service (separate binary)
│   ├── collectors/                  # One file per data source
│   │   ├── k8s_collector.py
│   │   ├── argocd_collector.py
│   │   ├── istio_collector.py
│   │   ├── prometheus_collector.py
│   │   ├── loki_collector.py
│   │   ├── kyverno_collector.py
│   │   └── github_collector.py
│   ├── context_builder.py           # Assembles PlatformContext snapshot
│   ├── agent.py                     # ReAct loop + LLM reasoning
│   ├── learning_store.py            # Incident history + outcome tracking
│   ├── prompt_templates/            # System prompts for each ARIA mode
│   │   ├── sre_diagnosis.txt
│   │   ├── alert_enrichment.txt
│   │   ├── health_advisor.txt
│   │   └── canary_decision.txt
│   ├── main.py                      # FastAPI app (webhook endpoints)
│   ├── models.py                    # Pydantic models
│   └── requirements.txt
│
├── ui/                              # React frontend
│   ├── src/
│   │   ├── components/
│   │   │   ├── TopologyCanvas/      # React Flow canvas — primary view
│   │   │   ├── NodeTypes/           # Custom node per tool type (ArgoCD, K8s, etc.)
│   │   │   ├── ServiceDetails/      # Right panel shown on node click
│   │   │   ├── DeploymentTimeline/  # Bottom-left: event stream
│   │   │   ├── PolicyPanel/         # Bottom-right: Kyverno violations
│   │   │   └── Sidebar/             # Left nav: All, GitHub, ArgoCD, Istio, etc.
│   │   ├── hooks/
│   │   │   └── usePlatformMap.ts    # Fetches and polls PlatformMap status
│   │   └── api/
│   │       └── client.ts            # API client for Korion backend
│   ├── package.json
│   └── vite.config.ts
│
├── config/
│   ├── crd/                         # Generated — do not edit manually
│   ├── rbac/                        # ClusterRole, ClusterRoleBinding, ServiceAccount
│   └── samples/
│       ├── platformmap-superheros.yaml
│       └── platformagent-superheros.yaml
│
└── helm/
    └── korion/
        ├── Chart.yaml
        ├── values.yaml
        └── templates/
```

---

## Key Design Decisions — Do Not Change Without Explicit Discussion

**1. Apache 2.0 license everywhere**
Every .go, .py, and .ts source file must have the Apache 2.0 license header.

**2. No kagent dependency**
ARIA is built independently. kagent is a generic agent framework — Korion's value
comes from system-specific topology context that only PlatformMap provides. A kagent
dependency would make Korion a plugin, not an independent CNCF project. Build the LLM
client, controller scaffolding, and tool integrations from scratch.

**3. ARIA never runs kubectl apply directly**
When ARIA takes remediation action, it commits the change to Git and lets ArgoCD
reconcile. GitOps principle is sacred. The only kubectl operations ARIA performs are
read-only: get pods, get events, describe deployment, get logs.

**4. Context assembly before every LLM call**
ARIA never calls the LLM without first building a complete PlatformContext snapshot
from the PlatformMap status. This snapshot includes live service health, recent
deployments, policy violations, Istio traffic weights, and error log patterns. Generic
LLM advice is worthless. System-specific context is everything.

**5. PlatformMap status is the single source of truth**
The Go controller writes the topology graph to PlatformMap status. ARIA reads from
PlatformMap status, not directly from the cluster. The frontend reads from PlatformMap
status. This separation keeps concerns clean and testable.

**6. React Flow for the canvas — do not replace**
React Flow handles zoom, pan, drag, edge routing, and custom node components correctly.
D3 or custom canvas would require reimplementing all of this. React Flow is the right
library for node-based interactive diagrams.

**7. Tool brand colors for nodes**
Each tool type uses its brand color for the node border:
- GitHub: #333333
- GitHub Actions: #3B82F6
- Docker: #2496ED
- ArgoCD: #EF7B4D
- Kubernetes/EKS: #326CE5
- Istio: #466BB0
- Kyverno: #1E40AF
- Prometheus: #EA580C
- Grafana: #F46800
- Loki: #10B981

**8. Confidence scoring on ARIA responses**
Every ARIA diagnosis must include a confidence score (0-100). Low confidence
(<60) should always require human approval regardless of autonomyLevel setting.

**9. Multi-cloud support from v0.1**
Discovery engines must not be AWS-specific. Use generic K8s API calls where possible.
Cloud-specific logic (EKS vs GKE vs AKS) should be behind interfaces, not hardcoded.

---

## Build and Run Commands

```bash
# Install tools (first time only)
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

# Generate CRD manifests and DeepCopy from Go types — run after any type changes
make generate
make manifests

# Run all Go tests
make test

# Run Go tests with coverage
make test COVERAGE=true

# Build controller binary
make build

# Build and push Docker image
make docker-build IMG=korion/korion:dev
make docker-push IMG=korion/korion:dev

# Install CRDs to current kubeconfig cluster
make install

# Run controller locally against current kubeconfig (for development)
make run

# Deploy controller to cluster
make deploy IMG=korion/korion:dev

# Uninstall CRDs
make uninstall

# ARIA Python service — development
cd aria
pip install -r requirements.txt
uvicorn main:app --reload --port 8080

# ARIA Python tests
cd aria
pytest tests/ -v

# Frontend — development
cd ui
npm install
npm run dev                     # starts on localhost:3000

# Frontend — production build
cd ui
npm run build

# Apply sample PlatformMap (requires Kind cluster with SuperHeros deployed)
kubectl apply -f config/samples/platformmap-superheros.yaml
kubectl apply -f config/samples/platformagent-superheros.yaml

# Watch PlatformMap status
kubectl get platformmap superheros-platform -n superheros -o yaml -w
```

---

## ARIA Context Builder — Critical Implementation Detail

The PlatformContext structure assembled before every LLM call must follow this schema.
This is the most important data structure in the project:

```python
@dataclass
class PlatformContext:
    platform_name: str
    cluster: str
    namespace: str
    timestamp: str

    services: dict[str, ServiceHealth]     # key = service name
    gitops: GitOpsState
    recent_deployments: list[DeploymentEvent]  # last 2 hours
    policy_violations: list[PolicyViolation]
    recent_errors: list[LogPattern]             # last 30 minutes
    ci_status: dict[str, CIRunStatus]           # key = service name
    cluster_resources: ClusterResources

@dataclass
class ServiceHealth:
    name: str
    namespace: str
    replicas_ready: int
    replicas_total: int
    image: str
    image_tag: str
    error_rate_pct: float
    latency_p99_ms: float
    cpu_usage_m: int
    memory_usage_mi: int
    istio_traffic_weight: int       # 0-100, only for canary services
    pod_restarts_1h: int
    last_deployed: str              # ISO timestamp
    argocd_app: str

@dataclass
class GitOpsState:
    app_name: str
    sync_status: str                # Synced | OutOfSync | Unknown
    health_status: str              # Healthy | Degraded | Missing
    last_sync_time: str
    last_sync_commit: str
    last_sync_commit_message: str
    revision: int
```

The context_builder.py assembles this from all collectors in parallel (asyncio.gather).
Total assembly time must be under 5 seconds. If any collector fails, it returns a
partial context with the failed source marked as unavailable — never block on failures.

---

## Primary Test Case — SuperHeros Platform

The canonical test case for all Korion development is the SuperHeros project:
`https://github.com/gc-ghub/project-gc-industries-devops-superheros`

**Services:** frontend (React), catalog-v1/v2/v3 (Go), inventory (Python/FastAPI),
orders (Node.js), payment (.NET)

**Stack:** AWS EKS, GitHub Actions CI, ArgoCD GitOps, Istio service mesh (canary
traffic splitting on catalog), Kyverno policies, Prometheus + Grafana + Loki + Tempo
observability stack, Alertmanager + Slack notifications

**v0.1 acceptance criteria:**
Apply platformmap-superheros.yaml to a Kind cluster running SuperHeros. Within 60
seconds, the Korion UI must show:
- All 6 services as nodes on the topology canvas
- ArgoCD node showing current sync status
- GitHub Actions node showing last run status
- Catalog-v1/v2/v3 nodes showing Istio traffic weights
- Kyverno violation count badge on affected nodes
- Clicking any service node opens ServiceDetails panel with live metrics
- Deployment Timeline panel shows last 3 ArgoCD syncs as a chronological stream

---

## ARIA Prompt Template Guidelines

All prompt templates in aria/prompt_templates/ follow this structure:
1. Role definition: "You are an expert SRE engineer monitoring the {platform_name} platform"
2. Context injection: full PlatformContext as structured JSON
3. Task specification: what to diagnose/recommend
4. Output format: structured JSON with required fields (root_cause, explanation,
   confidence, kubectl_commands, prevention, severity)
5. Constraints: "Base your analysis ONLY on the provided context. If you cannot
   determine root cause from the context, say so explicitly. Do not give generic
   Kubernetes advice — be specific to THIS platform."

Never let prompts produce generic advice. The system-specific constraint is critical.

---

## What Success Looks Like

**v0.1** — PlatformMap discovery works against SuperHeros cluster. Canvas renders
correct topology. Node click shows ServiceDetails. No ARIA yet.

**v0.2** — ARIA alert enrichment working. Alertmanager → ARIA → Slack with
structured SRE analysis that references specific SuperHeros services.

**v0.3** — ARIA health advisor producing daily platform health reports. Canary
decision assistant recommending promote/hold/rollback on catalog v1/v2/v3.

**v1.0** — Full feature set working. Helm chart installable in any K8s cluster.
CNCF Sandbox application submitted.