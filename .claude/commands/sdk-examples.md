# sdk-examples

Create SDK example programs under `examples/` that demonstrate kflow workflow features running fully in-process via `internal/local.RunLocal` (no Kubernetes required).

## Usage

```
/sdk-examples
```

## What this skill does

1. Explores `pkg/kflow/` to understand available types and builder methods.
2. Creates one Go `main` package per example under `examples/<NN>-<name>/main.go`.
3. Each example runs end-to-end using `local.RunLocal(wf, input)` and prints human-readable state-by-state output.
4. Adds an `examples` Makefile target that runs all examples via Docker.
5. Runs `make examples` and `go test ./...` to verify everything passes.

## Examples to create (or extend)

| Dir | Workflow | Features demonstrated |
|-----|----------|-----------------------|
| `examples/01-linear/` | `order-processing` | Linear chain, input→output threading |
| `examples/02-branching/` | `loan-approval` | Choice state, two branches |
| `examples/03-retry-catch/` | `data-pipeline` | RetryPolicy, Catch state, `_error` key |
| `examples/04-wait/` | `scheduled-notification` | Wait state, timed pause |

Add further examples as new numbered directories when new SDK features are implemented (e.g., Parallel states, InvokeService).

## Handler pattern

Each example has a `handlers/` sub-package with a named struct and methods:

```
examples/<NN>-<name>/
  handlers/handlers.go    ← <Domain>Handlers struct + methods
  main.go                 ← thin wiring: h := handlers.New(); wf.Task("...", h.Method)
```

Handler methods match `kflow.HandlerFunc` or `kflow.ChoiceFunc` exactly:

```go
func (h *OrderHandlers) ValidateOrder(ctx context.Context, input kflow.Input) (kflow.Output, error)
func (h *LoanHandlers) RouteDecision(ctx context.Context, input kflow.Input) (string, error)
```

`main.go` wires them with:

```go
h := handlers.New()
wf.Task("ValidateOrder", h.ValidateOrder)
wf.Choice("RouteDecision", h.RouteDecision)
```

Stateful handlers (e.g. retry simulation) accept constructor arguments:

```go
handlers.New(succeedAfter int) *PipelineHandlers
```

## Invariants

- All examples import only `pkg/kflow`, `internal/local`, and their own `handlers/` package — no direct store or engine imports.
- Every example exits 0 on success; `log.Fatalf` on workflow error.
- Handler methods print a one-line summary per state so output is readable without a debugger.
- No external services required (MemoryStore, in-process execution).

## State name constants

Every `handlers/` package exports a `const` block. `main.go` (and any test file)
references state names only through these constants — never as bare strings.

Naming convention: `State<StateName> = "<StateName>"`

Examples:
```go
handlers.StateValidateOrder   // "ValidateOrder"
handlers.StateRouteDecision   // "RouteDecision"
```

`ChoiceFunc` methods that return routing targets also use the constants
(no package prefix needed since they are in the same package):
```go
return StateApproveLoan, nil  // instead of "ApproveLoan"
```

## Engine fix: Choice state pass-through

Choice states must preserve the full input context for downstream states. The `local/runner.go` handler merges the routing key into a copy of the input rather than replacing it:

```go
if td.IsChoice() {
    choice, err := td.ChoiceFn()(ctx, inp)
    if err != nil {
        return nil, err
    }
    out := make(kflow.Output, len(inp)+1)
    for k, v := range inp {
        out[k] = v
    }
    out["__choice__"] = choice
    return out, nil
}
```

Verify this is in place before writing branching examples.

## Makefile target

```makefile
## examples: run all SDK example programs locally (no Kubernetes required)
examples:
	$(DOCKER_RUN) go run ./examples/01-linear
	$(DOCKER_RUN) go run ./examples/02-branching
	$(DOCKER_RUN) go run ./examples/03-retry-catch
	$(DOCKER_RUN) go run ./examples/04-wait
```

## Verification steps

```bash
make examples   # all four examples print output and exit 0
make test       # all unit tests pass
```
