# Multi-Language Workflows

This document explains how to build a workflow whose states are implemented in different languages (Go, Python, Rust), and how each SDK binary relates to the containers that the Control Plane spawns.

---

## The dual-role binary

Every SDK binary — regardless of language — serves two roles. The role is selected at startup by checking the command-line flags:

```
if --state=<name> is present:
    Role A — Job container: execute one handler, report output, exit.
else if --service=<name> is present:
    Role B — Service worker: enter service execution path (Deployment or Job mode).
else:
    Role C — Client: register the workflow/service with the Control Plane.
```

**The container image IS the SDK binary.** There is no separate runner agent injected into containers. When the Control Plane spawns a K8s Job for a state, it runs your compiled binary with `--state=<name>` and three environment variables:

| Variable | Purpose |
|---|---|
| `KFLOW_STATE_TOKEN` | HMAC-SHA256 signed token identifying this specific state execution |
| `KFLOW_GRPC_ENDPOINT` | Address of the Control Plane's internal `RunnerService` (`:9090`) |
| `KFLOW_EXECUTION_ID` | Execution UUID — for logging and observability only |

The container never touches MongoDB. All input/output exchange goes through `RunnerService` over gRPC.

---

## How a state executes inside a container

Once the K8s Job starts, the SDK's `run()` entry point detects `--state=<name>` and runs the runner protocol:

```
1. Read KFLOW_STATE_TOKEN, KFLOW_GRPC_ENDPOINT, KFLOW_EXECUTION_ID from env
2. Dial KFLOW_GRPC_ENDPOINT
3. RunnerService.GetInput(token)          → receives this state's input as a JSON dict
4. Look up the registered HandlerFunc for stateName
5. HandlerFunc(ctx, input)               → runs your actual code
6. On success: RunnerService.CompleteState(token, output)  → exit 0
   On error:   RunnerService.FailState(token, errMsg)      → exit 1
```

The Control Plane's `RunnerServiceServer` receives the `CompleteState` or `FailState` call and writes the result to MongoDB. The container has no direct database access.

---

## Setting up a multi-language workflow

### Scenario

A workflow defined in Go with:
- `validate-order` — a Go handler (runs in-process or as a Go container)
- `charge-payment` — a Rust handler (separate Rust binary + container image)
- `send-receipt` — a Python handler (separate Python binary + container image)

### Two approaches

There are two ways to wire a foreign-language handler into a workflow state:

**Approach A — Inline image reference** (image specified in the Go workflow):
```go
wf.Task("charge-payment", nil).WithImage("ghcr.io/your-org/rust-payment:v1.2")
```

**Approach B — Named job registration** (Rust/Python binary pre-registers itself; Go workflow references by name):
```go
wf.Task("charge-payment", nil).InvokeService("charge-payment-job")
```

Approach B is the recommended pattern for production. It is described in detail in the [Named Job Registration](#named-job-registration) section below.

---

### Go project — defines and registers the workflow

The Go binary is the authoritative source of the workflow shape. It defines all states and their transitions, but only needs to contain handlers for Go-language states. Foreign states are referenced by name via `InvokeService`.

```go
// cmd/main.go
package main

import (
    "context"
    "github.com/your-org/kflow/pkg/kflow"
)

func main() {
    wf := kflow.New("order-pipeline")

    // Go state — handler defined inline
    wf.Task("validate-order", func(ctx context.Context, in kflow.Input) (kflow.Output, error) {
        return kflow.Output{"valid": true}, nil
    })

    // Rust state — delegated to the registered "charge-payment-job" service
    wf.Task("charge-payment", nil).InvokeService("charge-payment-job")

    // Python state — delegated to the registered "send-receipt-job" service
    wf.Task("send-receipt", nil).InvokeService("send-receipt-job")

    wf.Flow(
        kflow.Step("validate-order").Next("charge-payment"),
        kflow.Step("charge-payment").Next("send-receipt"),
        kflow.Step("send-receipt").End(),
    )

    // Serialises the graph and POSTs to POST /api/v1/workflows, then POST .../run
    kflow.Run(wf)
}
```

The Go binary does not need to know the container images of the Rust or Python states. That knowledge lives in the Rust and Python projects respectively.

---

### Rust project — registers and handles `charge-payment`

The Rust binary does two things depending on how it is invoked:

- **Without any flag** (registration path): POSTs a `ServiceDef` to the Control Plane. The Control Plane stores the service name and image in `kflow_services`.
- **With `--service=charge-payment-job`** (execution path): runs the handler, reports output via `RunnerService`, exits.

```rust
// src/main.rs
use kflow::{ServiceDef, ServiceMode, Input, Output};

#[tokio::main]
async fn main() {
    let svc = ServiceDef::new("charge-payment-job")
        .mode(ServiceMode::Job)
        .image("ghcr.io/your-org/rust-payment:v1.2")  // the image to spawn
        .handler(Box::new(|input: Input| Box::pin(async move {
            // charge the card
            let mut out = Output::new();
            out.insert("charged".into(), true.into());
            Ok(out)
        })));

    kflow::run_service(svc).await;
    // With --service=charge-payment-job → runs handler, reports output, exits
    // Without flag → POSTs ServiceDef to Control Plane API (registration)
}
```

Build this into a container image and push it:
```bash
docker build -t ghcr.io/your-org/rust-payment:v1.2 .
docker push ghcr.io/your-org/rust-payment:v1.2
```

---

### Python project — registers and handles `send-receipt`

Same dual-role pattern in Python:

```python
# main.py
import kflow

app = kflow.new_service("send-receipt-job")
app.mode(kflow.ServiceMode.Job)
app.image("ghcr.io/your-org/python-email:v1.0")

@app.handler
def send_receipt(input: kflow.Input) -> kflow.Output:
    # send the email
    return {"sent": True}

if __name__ == "__main__":
    kflow.run_service(app)
    # With --service=send-receipt-job → runs handler, reports output, exits
    # Without flag → POSTs ServiceDef to Control Plane API (registration)
```

---

## Execution flow for the full workflow

Once all three binaries are registered and the Go workflow is submitted, the Control Plane executes:

```
Executor: "validate-order"
  → handler is inline Go fn
  → store.WriteAheadState → store.MarkRunning → fn() → store.CompleteState
  → output: {"valid": true}

Executor: "charge-payment"
  → InvokeService("charge-payment-job")
  → store.GetService("charge-payment-job") → ServiceRecord{Image: "ghcr.io/.../rust-payment:v1.2"}
  → store.WriteAheadState → store.MarkRunning
  → ServiceDispatcher.Dispatch:
      CreateJob(image="ghcr.io/.../rust-payment:v1.2", args=["--service=charge-payment-job"])
      inject: KFLOW_STATE_TOKEN, KFLOW_GRPC_ENDPOINT, KFLOW_EXECUTION_ID
      WaitForJob(...)
      [Rust container: GetInput → handler() → CompleteState → exit 0]
      store.GetStateOutput(execID, "charge-payment") → output: {"charged": true}
  → DeleteJob (best-effort)

Executor: "send-receipt"
  → InvokeService("send-receipt-job")
  → store.GetService("send-receipt-job") → ServiceRecord{Image: "ghcr.io/.../python-email:v1.0"}
  → ServiceDispatcher.Dispatch (same flow as above for Python container)
  → output: {"sent": true}

Executor: terminal state reached → store.CompleteExecution
```

Each container is completely isolated. The Rust and Python containers never communicate with each other or with MongoDB. All coordination goes through the Control Plane's `RunnerService`.

---

## Named job registration

This is the recommended pattern for multi-language workflows. Instead of embedding the container image URL inside the Go workflow definition, each language project self-registers as a named job. The Go workflow references jobs by their registered name.

### Why this is better than inline image references

| Concern | Inline image in Go workflow | Named job registration |
|---|---|---|
| Image versioning | Changing the Rust image requires editing the Go project | Each project controls its own image; Go workflow is unchanged |
| Decoupling | Go and Rust deployments are coupled | Fully independent deploy cycles |
| Discoverability | Image URLs scattered across workflow definitions | Central registry: `GET /api/v1/services` lists all registered jobs |
| Multi-team | One team must know all other teams' image tags | Each team registers and owns their job |

### Registration flow

When the Rust or Python binary is run without any flag (e.g., as a one-time registration step in a CI pipeline or as a Kubernetes init container), it calls the Control Plane:

```
POST /api/v1/services
{
  "name":  "charge-payment-job",
  "mode":  1,              // ServiceMode.Job
  "image": "ghcr.io/your-org/rust-payment:v1.2"
}
→ 201 Created
```

The Control Plane stores this in MongoDB (`kflow_services` collection) with `Status: Running`. For `ServiceMode.Job`, there is nothing to deploy upfront — the K8s Job is created on demand when `InvokeService` is triggered.

### Invocation flow

When the Executor reaches a state with `InvokeService("charge-payment-job")`:

```
ServiceDispatcher.Dispatch:
  1. store.GetService("charge-payment-job")
     → ServiceRecord{Name: "charge-payment-job", Mode: Job, Image: "ghcr.io/.../rust-payment:v1.2"}
  2. k8s.CreateJob(JobSpec{
         Image: ServiceRecord.Image,           ← per-job image, not the global executor image
         Args:  ["--service=charge-payment-job"],
         Env:   [KFLOW_STATE_TOKEN, KFLOW_GRPC_ENDPOINT, KFLOW_EXECUTION_ID],
     })
  3. k8s.WaitForJob(...)
  4. store.GetStateOutput(execID, stateName)
  5. k8s.DeleteJob (best-effort)
```

The image comes from the `ServiceRecord`, not from the Executor's global `Image` field. The global image is only used for Go states that run as K8s Jobs (Task states with an inline `HandlerFunc`).

### Re-registration and updates

To update the Rust image to a new version, re-register the service (upsert by name):

```
POST /api/v1/services
{
  "name":  "charge-payment-job",
  "mode":  1,
  "image": "ghcr.io/your-org/rust-payment:v1.3"   ← new image tag
}
```

All subsequent workflow executions will use the new image. In-flight executions that already started the Job continue with the old image until the Job completes.

---

## What each language binary actually contains

A critical detail: the Rust and Python binaries contain **only their own handlers**. They do not need to know the full workflow graph. The graph lives exclusively in the Go project and is registered with the Control Plane from there.

| Project | Contains | Does NOT contain |
|---|---|---|
| Go | Full workflow graph + Go handlers | Rust/Python handler code |
| Rust | `charge-payment-job` handler only | Workflow graph, Python handler |
| Python | `send-receipt-job` handler only | Workflow graph, Rust handler |

The Rust binary only needs to know its own service name (`charge-payment-job`) to handle `--service=charge-payment-job`. It does not need to register the workflow at all.

---

## Operational checklist for a multi-language workflow

1. **Build and push each language image:**
   ```bash
   docker build -t ghcr.io/your-org/rust-payment:v1.2 ./rust-project
   docker build -t ghcr.io/your-org/python-email:v1.0 ./python-project
   docker push ghcr.io/your-org/rust-payment:v1.2
   docker push ghcr.io/your-org/python-email:v1.0
   ```

2. **Register each job with the Control Plane** (once per image version; CI-friendly):
   ```bash
   # Run the Rust binary without --service flag to trigger registration
   docker run --rm ghcr.io/your-org/rust-payment:v1.2 \
     --control-plane=https://kflow.internal \
     --api-key=$KFLOW_API_KEY

   # Run the Python binary the same way
   docker run --rm ghcr.io/your-org/python-email:v1.0 \
     --control-plane=https://kflow.internal \
     --api-key=$KFLOW_API_KEY
   ```

3. **Run the Go workflow** (triggers execution; the Go binary knows the full graph):
   ```bash
   ./order-pipeline-binary \
     --control-plane=https://kflow.internal \
     --api-key=$KFLOW_API_KEY
   ```

4. **Monitor execution:**
   ```bash
   kflow execution get --id=<execution_id>
   kflow execution get --id=<execution_id>/states
   ```

---

## RunLocal and testing

For local development, `RunLocal` only runs Go handlers in-process. States that use `InvokeService` cannot be executed in a local run unless the service is mocked using the `WorkflowTest` harness (Phase 14):

```go
wt := kflow.NewWorkflowTest(wf).
    MockState("charge-payment", func(_ context.Context, in kflow.Input) (kflow.Output, error) {
        return kflow.Output{"charged": true}, nil
    }).
    MockState("send-receipt", func(_ context.Context, in kflow.Input) (kflow.Output, error) {
        return kflow.Output{"sent": true}, nil
    })

out, err := wt.Run(context.Background(), kflow.Input{"order_id": "test-123"})
wt.AssertStateOrder(t, "validate-order", "charge-payment", "send-receipt")
```

No Rust or Python binaries, no cluster, no Docker required.
