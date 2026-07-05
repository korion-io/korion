# Korion

Korion is an open-source, Kubernetes-native platform topology engine. It
auto-discovers the complete DevOps stack of any deployed application — from
developer commit to running pod — and renders it as an interactive, live
diagram in one unified dashboard.

Engineers install Korion the same way they install ArgoCD or Kyverno: apply a
CRD, and Korion discovers and maps everything automatically. No manual catalog
files. No configuration. Zero setup beyond the CRD.

Engineers today juggle 6+ separate tools — GitHub Actions, ArgoCD, Kiali,
Grafana, Prometheus, Kyverno — just to answer "what is happening on my
platform right now?" Korion replaces all of those open tabs with one
interactive, zero-configuration view of the full pipeline — GitHub Actions →
Docker → ArgoCD → Kubernetes → Istio — with live health on every node, and a
system-specific AI layer (ARIA) that reasons about *your* topology instead of
giving generic Kubernetes advice.

See [`CLAUDE.md`](./CLAUDE.md) for the full architecture and design decisions,
[`docs/PLAN.md`](./docs/PLAN.md) for the phased implementation plan,
[`task.md`](./task.md) for implementation progress, and
[`decisions.md`](./decisions.md) for the architectural decision log.

Status: pre-v0.1, under active development. Not yet ready for production use.

## License

Apache 2.0 — see [`LICENSE`](./LICENSE).
