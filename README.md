# kflow

**Kubernetes-native serverless workflow engine — self-hosted AWS Step Functions + Lambda**

Define state-machine workflows using a Go SDK; kflow handles containerisation, scheduling, and lifecycle management on Kubernetes. Supports local in-process execution (no cluster required) and full K8s execution via a gRPC runner protocol.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Interface Adapters                                                         │
│    internal/api/          grpc-gateway HTTP mux, WSHub, auth middleware     │
│    internal/grpc/         gRPC server, interceptors, gateway wiring         │
│    cmd/orchestrator/      Binary entry point, composition root              │
├─────────────────────────────────────────────────────────────────────────────┤
│  Application Layer                                                          │
│    internal/engine/       Executor, K8sExecutor (use cases)                 │
│    internal/controller/   ServiceDispatcher (use case)                      │
│    internal/runner/       RunnerServiceServer (use case: container callback)│
├─────────────────────────────────────────────────────────────────────────────┤
│  Domain Layer (no external imports)                                         │
│    pkg/kflow/             Workflow, TaskDef, ServiceDef, Input, Output,     │
│                           RetryPolicy, HandlerFunc — public SDK types       │
│    internal/store/        Store interface, ExecutionRecord, StateRecord,    │
│                           ServiceRecord, Status                             │
│    internal/engine/       Graph, Node (Domain Services)                     │
├─────────────────────────────────────────────────────────────────────────────┤
│  Infrastructure Layer                                                       │
│    internal/store/        MongoStore, MemoryStore, ObjectStore              │
│    internal/k8s/          K8s client, Job/Deployment/Ingress CRUD           │
│    internal/telemetry/    ClickHouse — Anti-Corruption Layer (ACL)          │
│    internal/config/       Environment-variable loader                       │
│    internal/gen/          buf-generated protobuf + gRPC + gateway code      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Two Runtime Contexts

The same binary serves two roles selected by flag at startup:

- **Control Plane** (`kflow.Run(wf)`): registers the workflow and drives the Executor.
- **State/Service worker** (`--state=<name>` / `--service=<name>`): runs inside a Kubernetes Job or Deployment, executes exactly one handler, and reports back via gRPC.

`RunLocal(wf)` runs everything in-process using `MemoryStore` — no Kubernetes needed.

### Execution Flow

```
SDK: kflow.Run(wf)
  → Control Plane API (grpc-gateway): POST /api/v1/workflows/:name/run
  → Executor.Run(ctx, execID, graph, input)
      for each state:
        1. store.WriteAheadState(...)        ← always first
        2. store.MarkRunning(...)
        3. Handler (inline / K8s Job / Service dispatch)
           ├── RunLocal: HandlerFunc called directly
           ├── K8s Job: container dials KFLOW_GRPC_ENDPOINT
           │     → RunnerService.GetInput(token)
           │     → HandlerFunc(ctx, input)
           │     → RunnerService.CompleteState / FailState
           └── Service: ServiceDispatcher → ServiceRunnerService.Invoke (gRPC)
        4. store.CompleteState / FailState (via RunnerServiceServer)
      → WSHub.Broadcast(event)
      → EventWriter.RecordStateTransition() (async)
```

### Three Bounded Contexts

1. **Workflow Execution** — `pkg/kflow`, `internal/engine`, `internal/store`
2. **Service Management** — `internal/controller`, `internal/k8s`
3. **Observability (Read Model)** — `internal/telemetry`, `ui/`

---

## Prerequisites

- Go 1.22+
- Docker & Docker Compose (all builds run in containers — no local Go install required)
- `make`
- Optional: `grpcurl` for gRPC testing

---

## Quick Start (Docker Compose)

```bash
# Start MongoDB, ClickHouse, and the orchestrator
make up

# Build the CLI binary
make build-cli

# Register a workflow
./bin/kflow workflow register --file=examples/demo-wf.json

# Run the workflow
./bin/kflow workflow run --name=demo-wf --input='{"hello":"world"}'
# → {"execution_id":"<uuid>"}

# Check execution status
./bin/kflow execution get --id=<uuid>
# → {"status":"Completed", ...}

# Tail orchestrator logs
make logs

# Stop all services
make down
```

---

## Environment Variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `KFLOW_MONGO_URI` | Yes | — | MongoDB connection URI |
| `KFLOW_MONGO_DB` | No | `kflow` | MongoDB database name |
| `KFLOW_NAMESPACE` | No | `kflow` | Kubernetes namespace for workloads |
| `KFLOW_GRPC_PORT` | No | `8080` | gRPC + grpc-gateway public port |
| `KFLOW_RUNNER_GRPC_PORT` | No | `9090` | RunnerService internal port |
| `KFLOW_RUNNER_GRPC_ENDPOINT` | No | `kflow-cp.kflow.svc.cluster.local:9090` | RunnerService address injected into Job containers |
| `KFLOW_RUNNER_TOKEN_SECRET` | Yes (prod) | — | HMAC-SHA256 key for state tokens (min 32 bytes) |
| `KFLOW_GRPC_TLS_CERT` | No | — | TLS cert path (empty = no TLS) |
| `KFLOW_GRPC_TLS_KEY` | No | — | TLS key path |
| `KFLOW_SERVICE_GRPC_PORT` | No | `9091` | Port for Deployment-mode service pods |
| `KFLOW_API_KEY` | No | — | Bearer token for API auth (empty = dev mode) |
| `KFLOW_CLICKHOUSE_DSN` | No | — | ClickHouse DSN (empty = telemetry disabled) |
| `KFLOW_IMAGE` | No | — | Container image tag for K8s Job execution |
| `KFLOW_STATE_TOKEN` | Yes (Lambda) | — | HMAC-SHA256 signed token injected into Job containers |
| `KFLOW_EXECUTION_ID` | Yes (Lambda) | — | Execution UUID injected into Job containers (logging) |
| `KFLOW_GRPC_ENDPOINT` | Yes (Lambda) | — | RunnerService address injected into Job containers |

---

## Go SDK Usage

```go
package main

import (
    "context"
    "time"

    "github.com/your-org/kflow/pkg/kflow"
)

func main() {
    wf := kflow.New("order-pipeline")

    validateOrder := wf.Task("validate-order", func(ctx context.Context, in kflow.Input) (kflow.Output, error) {
        // inline handler — runs in-process (RunLocal) or in a K8s Job
        return kflow.Output{"valid": true}, nil
    }).Retry(kflow.RetryPolicy{MaxAttempts: 3, IntervalSeconds: 2})

    wf.Task("handle-error", func(ctx context.Context, in kflow.Input) (kflow.Output, error) {
        return kflow.Output{"recovered": true}, nil
    })

    _ = validateOrder.Catch("handle-error")

    wf.Task("charge-payment", func(ctx context.Context, in kflow.Input) (kflow.Output, error) {
        return kflow.Output{"charged": true}, nil
    })

    wf.Wait("cool-down", 5*time.Second)

    wf.Flow(
        kflow.Step("validate-order").Next("charge-payment"),
        kflow.Step("charge-payment").Next("cool-down"),
        kflow.Step("cool-down").End(),
    )

    // In-process execution — no Kubernetes required
    kflow.RunLocal(wf)

    // Register with Control Plane and trigger execution
    // kflow.Run(wf)
}
```

**State types:**
- `wf.Task(name, fn)` — runs a handler function
- `wf.Choice(name, fn)` — branches based on a `ChoiceFunc`
- `wf.Wait(name, duration)` — pauses execution for a fixed duration
- `wf.Parallel(name, fn)` — fan-out parallel execution

---

## CLI Reference

Build: `make build-cli` → `./bin/kflow`

```
kflow [--server=URL] [--api-key=KEY] <group> <command> [flags]
```

| Command | Flags | Description |
|---|---|---|
| `kflow workflow register` | `--file=<path>` | Register a workflow from a JSON graph file |
| `kflow workflow list` | — | List all registered workflows |
| `kflow workflow run` | `--name=<name>` `--input=<json>` | Start a workflow execution |
| `kflow execution get` | `--id=<uuid>` | Get execution status and output |
| `kflow execution list` | `--workflow=<name>` | List executions (filterable by workflow name) |

**Global flags:**
- `--server=http://localhost:8080` — orchestrator base URL
- `--api-key=<key>` — bearer token for API auth

---

## REST API (grpc-gateway)

All endpoints are served on port `8080`. Auth header: `Authorization: Bearer <KFLOW_API_KEY>` (skipped when `KFLOW_API_KEY` is unset).

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/workflows` | Register a workflow |
| `GET` | `/api/v1/workflows` | List workflows |
| `GET` | `/api/v1/workflows/{name}` | Get a workflow |
| `POST` | `/api/v1/workflows/{name}/run` | Start an execution |
| `GET` | `/api/v1/executions` | List executions (`?workflow=`, `?status=`, `?limit=`) |
| `GET` | `/api/v1/executions/{id}` | Get execution status |
| `GET` | `/api/v1/executions/{id}/states` | List state records for an execution |
| `GET` | `/healthz` | Liveness probe (auth-exempt) |
| `GET` | `/readyz` | Readiness probe (auth-exempt) |
| `GET` | `/api/v1/ws` | WebSocket event stream |

---

## gRPC RunnerService (internal)

A real gRPC server runs on port `9090` (not HTTP). It is used exclusively by K8s Job containers to report execution outcomes back to the Control Plane.

```bash
# List available services (dev only — not exposed outside cluster in prod)
grpcurl -plaintext localhost:9090 list
```

All calls require an HMAC-SHA256 signed state token (`KFLOW_STATE_TOKEN`). This port is not exposed outside the cluster in production.

---

## Deployment (Kubernetes / Helm)

```bash
helm install kflow deployments/k8s \
  --set mongodb.uri="mongodb://your-mongo:27017" \
  --set image.repository=ghcr.io/your-org/kflow \
  --set image.tag=v0.1.0
```

Key Helm values:

| Value | Description |
|---|---|
| `mongodb.uri` | MongoDB connection URI |
| `clickhouse.dsn` | ClickHouse DSN (optional) |
| `image.repository` | Container image repository |
| `image.tag` | Container image tag (never `:latest`) |
| `ingress.enabled` | Enable Kubernetes Ingress |
| `ingress.host` | Ingress hostname |
| `workloadNamespace` | Namespace for spawned Job/Deployment workloads |

---

## Development

All build and test commands run inside Docker — no local Go or Node installation required.

```bash
make build              # compile all packages
make test               # unit tests (no external deps)
make test-race          # unit tests with race detector
make test-integration   # integration tests (requires make up first)
make vet                # go vet
make proto-gen          # regenerate protobuf code from proto/kflow/v1/
make ui-dev             # SvelteKit dev server at localhost:5173
make ui-build           # build SvelteKit dashboard
make docker-build       # build the kflow:dev container image
make clean              # remove build artifacts
```

---

## Project Layout

```
cmd/orchestrator/          Control Plane binary entrypoint
cmd/kflow/                 CLI client binary
internal/
  api/                     HTTP server, WebSocket hub, auth middleware
  config/                  Config struct, LoadConfig() from env vars
  controller/              ServiceDispatcher
  engine/                  Graph (compiled workflow), Executor (state machine driver)
  gen/                     buf-generated protobuf + gRPC + grpc-gateway code
  grpc/                    gRPC server, interceptors, grpc-gateway mux
  k8s/                     Kubernetes client: Jobs, Deployments, Services, Ingress
  runner/                  RunnerServiceServer, state token security (HMAC-SHA256)
  store/                   Store interface, MemoryStore, MongoStore, ObjectStore
  telemetry/               ClickHouse client, EventWriter, MetricsWriter, LogWriter
pkg/kflow/                 Public Go SDK (TaskDef, ServiceDef, Workflow, RunLocal, RunService)
proto/
  kflow/v1/                Protobuf definitions
  buf.yaml                 buf tool config
  buf.gen.yaml             Code generation: Go, gRPC, grpc-gateway, OpenAPI
sdk/python/                Python SDK
sdk/rust/                  Rust SDK
ui/                        SvelteKit dashboard
deployments/k8s/           Helm chart
docs/phases/               Phase reference files (authoritative design docs)
```
