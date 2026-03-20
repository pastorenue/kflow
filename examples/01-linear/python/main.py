# Example 01: linear order-processing pipeline
# ValidateOrder → CalculateTax → ChargePayment → SendConfirmation → end
import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../sdk/python"))
import kflow

from handlers import (
    STATE_VALIDATE_ORDER,
    STATE_CALCULATE_TAX,
    STATE_CHARGE_PAYMENT,
    STATE_SEND_CONFIRMATION,
    validate_order,
    calculate_tax,
    charge_payment,
    send_confirmation,
)


def main():
    wf = kflow.Workflow("order-processing")
    wf.task(STATE_VALIDATE_ORDER)(validate_order)
    wf.task(STATE_CALCULATE_TAX)(calculate_tax)
    wf.task(STATE_CHARGE_PAYMENT)(charge_payment)
    wf.task(STATE_SEND_CONFIRMATION)(send_confirmation)

    wf.flow(
        kflow.step(STATE_VALIDATE_ORDER).next(STATE_CALCULATE_TAX),
        kflow.step(STATE_CALCULATE_TAX).next(STATE_CHARGE_PAYMENT),
        kflow.step(STATE_CHARGE_PAYMENT).next(STATE_SEND_CONFIRMATION),
        kflow.step(STATE_SEND_CONFIRMATION).end(),
    )

    print("=== 01-linear: order-processing ===")
    kflow.run_local(wf, {"order_id": "ORD-2001", "customer": "alice", "total": 149.99})
    print("  → workflow COMPLETED")


if __name__ == "__main__":
    main()
