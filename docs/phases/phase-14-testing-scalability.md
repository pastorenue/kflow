# Phase 14 — Testing Harness & Horizontal Scalability

## Goal

Provide a first-class partial-flow test harness that lets users test any workflow or any subset of states without a Kubernetes cluster. Extend the store interface with a distributed claim mechanism so multiple orchestrator replicas can run concurrently without double-executing states. Add Helm HPA support to enable 100x horizontal scalability. Document the cluster compatibility matrix.

---

## DDD Classification

| DDD Construct | Type(s) in this phase |
|---|---|
| Value Object | `WorkflowTest` (test helper; wraps `RunLocal`) |
| Repository | `store.Store` (extended with `ClaimState`) |
| Application Service | `Executor` (extended to call `ClaimState` before `MarkRunning`) |
| Infrastructure | HPA manifest in Helm chart |

**Architectural position:** `pkg/kflow/testing.go` is part of the public SDK (test-only); it must not import `internal/`. `ClaimState` is added to `internal/store/store.go` and implemented in both `MemoryStore` and `MongoStore`. `Executor` changes are in `internal/engine/executor.go`.

---

## Phase Dependencies

- **Phase 1**: `pkg/kflow` types.
- **Phase 2**: `Executor`, `MemoryStore`, `Graph`.
- **Phase 3**: `MongoStore`.
- **Phase 10**: Helm chart (`deployments/k8s/`).

---

## Part A — Partial-Flow Test Harness

### `pkg/kflow/testing.go`

```go
// WorkflowTest is a test helper that wraps RunLocal to allow selective mocking,
// state skipping, custom entry points, and post-run assertions.
// All execution uses MemoryStore and the existing Executor — no new execution path.
// Intended for use in Go test files only.
type WorkflowTest struct {
    wf      *Workflow
    mocks   map[string]HandlerFunc
    skipped map[string]bool
    startFrom string
    store   store.Store    // populated after Run(); used by assertion helpers
    execID  string         // populated after Run()
}

// NewWorkflowTest creates a WorkflowTest wrapping wf.
func NewWorkflowTest(wf *Workflow) *WorkflowTest

// MockState replaces the handler for the named state with fn during test execution.
// Useful for isolating a specific state from its real dependencies.
func (t *WorkflowTest) MockState(name string, fn HandlerFunc) *WorkflowTest

// SkipState configures the named state to pass its input directly to the next state
// as its output unchanged. The state still appears in the execution record.
func (t *WorkflowTest) SkipState(name string) *WorkflowTest

// StartFrom sets the workflow entry point to the named state, bypassing all preceding
// states. The workflow graph must contain a path from that state to a terminal.
func (t *WorkflowTest) StartFrom(state string) *WorkflowTest

// Run executes the workflow using RunLocal with all configured mocks and overrides applied.
// Returns the final output and any execution error.
func (t *WorkflowTest) Run(ctx context.Context, input Input) (Output, error)

// AssertStateOrder asserts that the named states executed in the given order.
// States not listed are ignored. Calls t.Fatal on failure.
func (t *WorkflowTest) AssertStateOrder(tb testing.TB, states ...string)

// AssertStateOutput asserts that the named state produced the expected output.
// Calls t.Fatal on failure.
func (t *WorkflowTest) AssertStateOutput(tb testing.TB, state string, expected Output)

// StateOutputs returns a map from state name to its recorded output for all completed states.
func (t *WorkflowTest) StateOutputs() map[string]Output
```

#### Implementation notes

- `MockState` and `SkipState` wrap the original `HandlerFunc` stored in the `Workflow`'s task map before passing the workflow to `RunLocal`. The original `Workflow` is not mutated; `WorkflowTest` works on a shallow copy of the task registry.
- `StartFrom` sets `Graph.EntryState` on the compiled graph passed to the `Executor`, skipping earlier states from the execution loop. Skipped states are not written to the store.
- `Run` creates a `MemoryStore`, creates an `ExecutionRecord`, builds the `Graph`, and calls `Executor.Run`. After `Run`, the `MemoryStore` is retained for assertion helpers.
- `AssertStateOrder` reads `store.ListStateRecords(ctx, execID)`, sorts by `StartedAt`, and checks the subsequence.
- `AssertStateOutput` calls `store.GetStateOutput(ctx, execID, state)` and compares with `reflect.DeepEqual`.

#### Example test

```go
func TestOrderPipeline_SkipPayment(t *testing.T) {
    wf := buildOrderWorkflow()
    wt := kflow.NewWorkflowTest(wf).
        MockState("validate-order", func(_ context.Context, in kflow.Input) (kflow.Output, error) {
            return kflow.Output{"valid": true}, nil
        }).
        SkipState("charge-payment").
        StartFrom("validate-order")

    out, err := wt.Run(context.Background(), kflow.Input{"order_id": "abc"})
    if err != nil {
        t.Fatal(err)
    }
    wt.AssertStateOrder(t, "validate-order", "charge-payment", "cool-down")
    wt.AssertStateOutput(t, "validate-order", kflow.Output{"valid": true})
    _ = out
}
```

---

## Part B — Distributed Claim Mechanism (100x Scalability)

### `internal/store/store.go` — new method

```go
// ClaimState attempts to exclusively claim a state execution for the given workerID.
// It is called by the Executor before MarkRunning to prevent double-execution when
// multiple orchestrator replicas race to execute the same state.
//
// Returns claimed=true if this call successfully claimed the state.
// Returns claimed=false if another worker already claimed it (no error).
// Returns an error only for infrastructure failures (e.g. MongoDB timeout).
//
// Implementation contract:
//   - MongoStore: upsert with {execution_id, state_name, attempt, status: "Claimed"} using
//     a unique compound index on (execution_id, state_name, attempt). A duplicate-key error
//     means another replica won; return claimed=false, nil.
//   - MemoryStore: in-memory mutex-protected map; always claims successfully (single-process).
ClaimState(ctx context.Context, execID, stateName string, attempt int, workerID string) (claimed bool, err error)
```

### `internal/engine/executor.go` — change to `executeState`

Insert a `ClaimState` call between `WriteAheadState` and `MarkRunning`:

```
1. store.WriteAheadState(ctx, StateRecord{Status: Pending, Attempt: N})
2. store.ClaimState(ctx, execID, stateName, N, workerID)
   └── if claimed == false → return nil (skip; another replica owns this state)
3. store.MarkRunning(ctx, execID, stateName)
4. Call Handler
5. store.CompleteState / store.FailState
```

`workerID` is a stable identifier for the orchestrator replica, derived from the pod name (`KFLOW_WORKER_ID` env var, defaulting to the hostname).

### MongoDB index for `ClaimState`

Add to `MongoStore` initialisation (alongside existing indexes):

```go
// Unique compound index enforcing one claim per (execution_id, state_name, attempt).
// Used by ClaimState to implement distributed optimistic locking.
mongo.IndexModel{
    Keys: bson.D{
        {Key: "execution_id", Value: 1},
        {Key: "state_name",   Value: 1},
        {Key: "attempt",      Value: 1},
    },
    Options: options.Index().SetUnique(true).SetName("uidx_exec_state_attempt"),
}
```

This index is on the `state_claims` collection (separate from `state_records`) to avoid conflating claim state with execution state.

### Stateless replicas

Orchestrator replicas are fully stateless between executions:
- No in-memory execution state is shared between replicas.
- No sticky sessions or affinity rules are needed.
- Any replica can pick up an execution triggered by any other replica, as long as the MongoDB `ExecutionRecord` is visible (consistent reads).

---

## Part C — Helm HPA

Add to `deployments/k8s/values.yaml`:

```yaml
hpa:
  enabled: false
  minReplicas: 1
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
```

Add `deployments/k8s/templates/hpa.yaml`:

```yaml
{{- if .Values.hpa.enabled }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "kflow.fullname" . }}
  labels:
    {{- include "kflow.labels" . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "kflow.fullname" . }}
  minReplicas: {{ .Values.hpa.minReplicas }}
  maxReplicas: {{ .Values.hpa.maxReplicas }}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ .Values.hpa.targetCPUUtilizationPercentage }}
{{- end }}
```

**Recommended production minimum:** 3 replicas for high availability. Set `hpa.minReplicas: 3` in production values.

---

## Part D — Vendor-Agnostic Compatibility

### Cluster compatibility matrix

| Environment | Tested configurations |
|---|---|
| Local dev | `kind` ≥ 0.20, `minikube` ≥ 1.32, `k3s` ≥ 1.28 |
| Cloud managed | EKS (AWS) ≥ 1.28, GKE (Google Cloud) ≥ 1.28, AKS (Azure) ≥ 1.28, DigitalOcean Kubernetes ≥ 1.28 |
| On-premise / edge | bare-metal Kubernetes ≥ 1.24, k3s ≥ 1.24, RKE2 ≥ 1.24 |

**Hard requirements:**
- Kubernetes ≥ 1.24 (stable Job and Deployment APIs, RBAC GA)
- RBAC enabled
- Standard Job and Deployment API — no cloud-provider extensions required
- Persistent volumes not required (MongoDB is external)

**Object store compatibility:**
Any S3-compatible endpoint is supported via `KFLOW_OBJECT_STORE_URI`:
- MinIO (self-hosted)
- Ceph RGW
- AWS S3
- Google Cloud Storage (S3 interoperability mode)
- Azure Blob Storage (with S3-compatible adapter)

kflow uses no cloud-provider SDK directly. All object store access goes through the standard AWS S3 API (`aws-sdk-go-v2/service/s3`), which works with any S3-compatible endpoint.

---

## Acceptance Criteria / Verification

- [ ] `WorkflowTest.MockState` replaces handler output without mutating the original `Workflow`.
- [ ] `WorkflowTest.SkipState` passes input unchanged to next state and still records a state entry.
- [ ] `WorkflowTest.StartFrom` skips all states before the named entry point.
- [ ] `WorkflowTest.AssertStateOrder` fails (calls `t.Fatal`) when states execute out of order.
- [ ] `WorkflowTest.AssertStateOutput` fails when output does not match expected.
- [ ] `store.ClaimState` returns `claimed=false` for a second caller with identical `(execID, stateName, attempt)`.
- [ ] `MemoryStore` satisfies `ClaimState` with mutex-based single-claim semantics.
- [ ] `MongoStore` satisfies `ClaimState` using the unique compound index; duplicate-key error maps to `claimed=false, nil`.
- [ ] Integration test: two concurrent `Executor` instances backed by the same `MongoStore` execute each state exactly once (verified by `ListStateRecords` count).
- [ ] Helm chart renders a valid `HorizontalPodAutoscaler` when `hpa.enabled=true`.
- [ ] `helm lint deployments/k8s` passes with `hpa.enabled=true`.
- [ ] `go test ./pkg/kflow/... -run TestWorkflowTest` passes with no external dependencies.
