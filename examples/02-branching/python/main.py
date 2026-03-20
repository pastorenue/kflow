# Example 02: choice state — loan approval branching on credit score
# AssessRisk → (choice: credit ≥ 700 → ApproveLoan, else → RejectLoan) → end
import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../sdk/python"))
import kflow

from handlers import (
    STATE_ASSESS_RISK,
    STATE_ROUTE_DECISION,
    STATE_APPROVE_LOAN,
    STATE_REJECT_LOAN,
    assess_risk,
    route_decision,
    approve_loan,
    reject_loan,
)


def build_workflow():
    wf = kflow.Workflow("loan-approval")
    wf.task(STATE_ASSESS_RISK)(assess_risk)
    wf.choice(STATE_ROUTE_DECISION)(route_decision)
    wf.task(STATE_APPROVE_LOAN)(approve_loan)
    wf.task(STATE_REJECT_LOAN)(reject_loan)

    wf.flow(
        kflow.step(STATE_ASSESS_RISK).next(STATE_ROUTE_DECISION),
        kflow.step(STATE_ROUTE_DECISION).next(STATE_APPROVE_LOAN),
        kflow.step(STATE_APPROVE_LOAN).end(),
        kflow.step(STATE_REJECT_LOAN).end(),
    )
    return wf


def run(applicant: str, score: float, amount: float):
    print(f"\n--- applicant={applicant} score={score:.0f} ---")
    kflow.run_local(build_workflow(), {
        "applicant":    applicant,
        "credit_score": score,
        "amount":       amount,
    })
    print("  → workflow COMPLETED")


def main():
    print("=== 02-branching: loan-approval ===")
    run("alice", 750, 25000)
    run("bob", 620, 10000)


if __name__ == "__main__":
    main()
