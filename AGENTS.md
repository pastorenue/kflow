# Kubernetes-Native Serverless Workflow Engine

## Project Overview

This project is a **serverless-style workflow orchestration tool** built in Go that runs on top of Kubernetes. It is conceptually similar to AWS Step Functions or AWS Lambda — but self-hosted, cost-optimized, and Kubernetes-native.

Users write only their business logic. The system handles containerization, scheduling, and lifecycle management by spinning up Kubernetes Jobs, Deployments, or Pods as needed.

---

## Core Concept

### Developer Experience

Users write Go functions and register them as states:

```go
package main

import (
    "context"
    "github.com/your-org/kflow"
)

func main() {
    wf := kflow.NewWorkflow("order-pipeline")

    wf.Task("ValidateOrder", func(ctx context.Context, input kflow.Input) (kflow.Output, error) {
        // validation logic
        return kflow.Output{"valid": true}, nil
    })

    wf.Task("ChargePayment", func(ctx context.Context, input kflow.Input) (kflow.Output, error) {
        // payment logic
        return kflow.Output{"charged": true}, nil
    })

    wf.Task("HandleFailure", func(ctx context.Context, input kflow.Input) (kflow.Output, error) {
        // error handling logic
        return kflow.Output{}, nil
    })

    wf.Flow(
        kflow.Step("ValidateOrder").Next("ChargePayment").Catch("HandleFailure"),
        kflow.Step("ChargePayment").Next(kflow.Succeed),
        kflow.Step("HandleFailure").End(),
    )

    kflow.Run(wf)
}
```

The engine:
1. Compiles the workflow graph from registered functions + flow definitions
2. Submits each state as a Kubernetes Job (isolated execution) or runs in-process (fast mode)
3. Passes input/output between states via the state store
4. Handles retries, branching, error catching, and terminal states

---

## Architecture

```
┌──────────────────────────────────────────────────────-┐
│              kflow.Run(wf)  (Go binary)               │
│  - Serializes workflow graph                          │
│  - Submits execution request to Control Plane API     │
└───────────────────┬──────────────────────────────────-┘
                    │ HTTP / gRPC
┌───────────────────▼──────────────────────────────────-┐
│              Control Plane (Go)                       │
│  - Owns execution lifecycle                           │
│  - Resolves next state on each transition             │
│  - Persists state (write-ahead before execution)      │
└───────┬──────────────────────────┬───────────────────-┘
        │                          │
┌───────▼────────┐      ┌──────────▼──────────────────-┐
│  State Store   │      │   Kubernetes Client (Go)     │
│  (MongoDB)     │      │   - Spawns Job per state     │
└────────────────┘      │   - Injects function via     │
                        │     shared binary or image   │
                        └─────────────────────────────-┘
```
---

## State Types

These are expressed as Go builder methods on the workflow object, not YAML keys.

| Method               | Behaviour                                              |
|----------------------|--------------------------------------------------------|
| `wf.Task(name, fn)`  | Runs fn in a Kubernetes Job; transitions on return     |
| `wf.Choice(name, fn)`| fn returns a branch key; engine routes accordingly     |
| `wf.Wait(name, dur)` | Pauses execution for a fixed duration                  |
| `wf.Parallel(name)`  | Spawns multiple branches concurrently, waits for all   |
| `kflow.Succeed`      | Terminal success state (built-in)                      |
| `kflow.Fail`         | Terminal failure state (built-in)                      |

### Choice Example

```go
wf.Choice("RouteOrder", func(ctx context.Context, input kflow.Input) (string, error) {
    if input["amount"].(float64) > 1000 {
        return "HighValuePath", nil
    }
    return "StandardPath", nil
})
```

---

## Execution Model

### How Go Functions Become Kubernetes Jobs

Each registered function is resolved at runtime via one of two strategies:

**Strategy A — Shared binary (default, fast)**
The same compiled Go binary is deployed as the orchestrator. When a state needs to run, the control plane spawns a Kubernetes Job using that same image, passing a `--state=<name>` flag. The binary detects the flag, runs only that function, writes output to the state store, and exits.

```
kflow.Run(wf)
  └─ if --state flag present: execute single function, write output, exit
  └─ else: submit workflow graph to control plane, wait/poll
```

**Strategy B — In-process (dev mode)**
For local development or low-latency needs, functions run in the same process. No Kubernetes Jobs are spawned. Retries and state transitions still go through the same engine logic, just without container isolation.

Set via: `kflow.RunLocal(wf)`

---

## Input / Output Between States

- Each function receives a `kflow.Input` (map[string]any) from the previous state's output
- Return a `kflow.Output` (map[string]any) which becomes the next state's input
- The control plane stores each state's output in Postgres keyed by `(execution_id, state_name)`
- No shared volumes or environment variable passing — state store is the single source of truth

---

## Error Handling

```go
wf.Flow(
    kflow.Step("ChargePayment").
        Retry(kflow.RetryPolicy{MaxAttempts: 3, BackoffSeconds: 5}).
        Catch("HandleFailure").
        Next("FulfillOrder"),
)
```

- `Retry` — retries the same function on error before catching
- `Catch` — routes to a named error-handler state on final failure
- Error handler states receive the original input plus an `_error` key with the error message

---

## File Structure

```
/
├── cmd/
│   └── orchestrator/        # Control plane entrypoint
├── internal/
│   ├── api/                 # HTTP/gRPC handlers (submit execution, poll status)
│   ├── controller/          # Reconciler loop: watches executions, drives transitions
│   ├── engine/              # State machine graph: parse, validate, resolve next state
│   ├── k8s/                 # Kubernetes client wrappers (create/watch/delete Jobs)
│   └── store/               # Postgres-backed execution + state output store
├── pkg/
│   └── kflow/               # Public SDK: Workflow, Task, Choice, Flow, Run, RunLocal
│       ├── workflow.go
│       ├── state.go
│       ├── input.go
│       └── runner.go
├── deployments/
│   └── k8s/                 # Helm chart for the control plane itself
└── AGENTS.md
```

---

## Key Invariants for the Agent

- **Never parse YAML for state definitions.** States are Go functions only.
- **The public SDK lives in `pkg/kflow`** — this is what user code imports.
- **Write-ahead persistence** — state must be written to the store before a Job is created, not after.
- **The `--state` flag pattern** is how the shared binary knows to run a single function vs. orchestrate.
- **`kflow.Input` and `kflow.Output` are `map[string]any`** — keep them JSON-serialisable.
- **RunLocal is for dev only** — never default to it in production paths.

---

## Open Questions / TODOs

- [ ] How to handle large outputs between states (blob storage vs inline in Postgres)
- [ ] Auth model for the control plane submission API
- [ ] OpenTelemetry spans per state transition for observability
- [ ] Cost accounting: CPU/memory per execution
- [ ] Workflow versioning: what happens when a function signature changes mid-execution
