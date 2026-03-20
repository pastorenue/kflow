# Skill: python-sdk-examples

Add or extend Python SDK examples under `examples/<NN>-<name>/python/`.

---

## Directory layout

```
examples/
  <NN>-<name>/
    python/
      handlers.py   ← state constants + handler functions
      main.py       ← workflow wiring + run_local call
```

Each example mirrors its Go counterpart in `examples/<NN>-<name>/main.go`.

---

## State constants

Use `SCREAMING_SNAKE_CASE`. Values are identical to the Go `const` block:

```python
# handlers.py
STATE_VALIDATE_ORDER    = "ValidateOrder"
STATE_CALCULATE_TAX     = "CalculateTax"
STATE_CHARGE_PAYMENT    = "ChargePayment"
STATE_SEND_CONFIRMATION = "SendConfirmation"
```

---

## Handler signatures

Regular task handler — takes `input: dict`, returns `dict`:
```python
def validate_order(input: dict) -> dict:
    total = input["total"]
    ...
    return {"order_id": input["order_id"], "total": total, "validated": True}
```

Choice handler — takes `input: dict`, returns `str` (the target state constant):
```python
def route_decision(input: dict) -> str:
    if input["credit_score"] >= 700:
        return STATE_APPROVE_LOAN
    return STATE_REJECT_LOAN
```

Stateful handler — use a class when a handler needs instance state (e.g. retry counters):
```python
class PipelineHandlers:
    def __init__(self, succeed_after: int):
        self._succeed_after = succeed_after
        self._attempt = 0

    def fetch_external_data(self, input: dict) -> dict:
        self._attempt += 1
        if self._attempt < self._succeed_after:
            raise RuntimeError(f"upstream timeout (attempt {self._attempt})")
        return {"record_id": input["record_id"], "raw": {...}}
```

---

## SDK path setup

Each `main.py` inserts `sdk/python` onto the path as a local-run fallback.
The Docker/pip install path takes precedence when the package is installed:

```python
import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../sdk/python"))
import kflow
```

---

## Wiring pattern in `main.py`

Use the decorator call form (not inline):

```python
wf = kflow.Workflow("order-processing")
wf.task(STATE_VALIDATE_ORDER)(validate_order)
wf.task(STATE_CALCULATE_TAX)(calculate_tax)

wf.flow(
    kflow.step(STATE_VALIDATE_ORDER).next(STATE_CALCULATE_TAX),
    ...
    kflow.step(STATE_SEND_CONFIRMATION).end(),
)

kflow.run_local(wf, {"order_id": "ORD-2001", "total": 149.99})
```

Choice state — use `wf.choice()`:
```python
wf.choice(STATE_ROUTE_DECISION)(route_decision)
```

Wait state — use `wf.wait(name, seconds)` (seconds is an `int`):
```python
wf.wait(STATE_WAIT_FOR_SEND_TIME, 2)
```

Retry and catch — set on `StepBuilder` in `wf.flow()`:
```python
wf.flow(
    kflow.step(STATE_FETCH_EXTERNAL_DATA)
        .next(STATE_ENRICH_RECORD)
        .catch(STATE_HANDLE_FETCH_ERROR)
        .retry(3, 0),
    ...
)
```

---

## Verification

```bash
# Run all four Python examples via Docker (no local Python required):
make py-examples

# Run Python SDK unit tests:
docker run --rm -v "$(pwd)":/workspace -w /workspace python:3.12-slim \
  sh -c "pip install -q sdk/python[dev] && pytest sdk/python/tests/"
```

---

## Existing examples

| Dir | Workflow | Pattern |
|-----|----------|---------|
| `examples/01-linear/python/` | order-processing | linear chain |
| `examples/02-branching/python/` | loan-approval | choice state |
| `examples/03-retry-catch/python/` | data-pipeline | retry + catch |
| `examples/04-wait/python/` | scheduled-notification | wait state |
