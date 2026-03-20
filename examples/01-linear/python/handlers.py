STATE_VALIDATE_ORDER    = "ValidateOrder"
STATE_CALCULATE_TAX     = "CalculateTax"
STATE_CHARGE_PAYMENT    = "ChargePayment"
STATE_SEND_CONFIRMATION = "SendConfirmation"


def validate_order(input: dict) -> dict:
    order_id = input["order_id"]
    total = input["total"]
    if total <= 0:
        raise ValueError(f"invalid order total: {total}")
    print(f"  ValidateOrder: order={order_id} total=${total:.2f} ✓")
    return {"order_id": order_id, "total": total, "validated": True}


def calculate_tax(input: dict) -> dict:
    total = input["total"]
    tax = total * 0.08
    print(f"  CalculateTax: subtotal=${total:.2f} tax=${tax:.2f}")
    return {
        "order_id":   input["order_id"],
        "total":      total,
        "tax":        tax,
        "grand_total": total + tax,
    }


def charge_payment(input: dict) -> dict:
    grand = input["grand_total"]
    print(f"  ChargePayment: charging ${grand:.2f}")
    return {
        "order_id":       input["order_id"],
        "grand_total":    grand,
        "payment_status": "captured",
        "txn_id":         "txn_abc123",
    }


def send_confirmation(input: dict) -> dict:
    print(f"  SendConfirmation: order={input['order_id']} txn={input['txn_id']} → email sent")
    return {"order_id": input["order_id"], "confirmed": True}
