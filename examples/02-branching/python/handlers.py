STATE_ASSESS_RISK    = "AssessRisk"
STATE_ROUTE_DECISION = "RouteDecision"
STATE_APPROVE_LOAN   = "ApproveLoan"
STATE_REJECT_LOAN    = "RejectLoan"


def assess_risk(input: dict) -> dict:
    score = input["credit_score"]
    applicant = input["applicant"]
    print(f"  AssessRisk: applicant={applicant} credit_score={score:.0f}")
    return {
        "applicant":    applicant,
        "credit_score": score,
        "amount":       input["amount"],
    }


def route_decision(input: dict) -> str:
    if input["credit_score"] >= 700:
        return STATE_APPROVE_LOAN
    return STATE_REJECT_LOAN


def approve_loan(input: dict) -> dict:
    amount = input["amount"]
    print(f"  ApproveLoan: {input['applicant']} approved for ${amount:.2f}")
    return {
        "applicant": input["applicant"],
        "decision":  "approved",
        "amount":    amount,
        "rate":      0.045,
    }


def reject_loan(input: dict) -> dict:
    print(f"  RejectLoan: {input['applicant']} rejected (score={input['credit_score']:.0f})")
    return {
        "applicant": input["applicant"],
        "decision":  "rejected",
        "reason":    "credit score below threshold",
    }
