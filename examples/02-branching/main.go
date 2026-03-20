// Example 02: choice state — loan approval branching on credit score
// AssessRisk → (choice: credit ≥ 700 → ApproveLoan, else → RejectLoan) → end
package main

import (
	"fmt"
	"log"

	"github.com/pastorenue/kflow/examples/02-branching/handlers"
	"github.com/pastorenue/kflow/internal/local"
	"github.com/pastorenue/kflow/pkg/kflow"
)

func buildWorkflow(h *handlers.LoanHandlers) *kflow.Workflow {
	wf := kflow.New("loan-approval")
	wf.Task(handlers.StateAssessRisk, h.AssessRisk)
	wf.Choice(handlers.StateRouteDecision, h.RouteDecision)
	wf.Task(handlers.StateApproveLoan, h.ApproveLoan)
	wf.Task(handlers.StateRejectLoan, h.RejectLoan)

	wf.Flow(
		kflow.Step(handlers.StateAssessRisk).Next(handlers.StateRouteDecision),
		kflow.Step(handlers.StateRouteDecision).Next(handlers.StateApproveLoan),
		kflow.Step(handlers.StateApproveLoan).End(),
		kflow.Step(handlers.StateRejectLoan).End(),
	)
	return wf
}

func run(applicant string, score, amount float64) {
	fmt.Printf("\n--- applicant=%s score=%.0f ---\n", applicant, score)
	wf := buildWorkflow(handlers.New())
	if err := local.RunLocal(wf, kflow.Input{
		"applicant":    applicant,
		"credit_score": score,
		"amount":       amount,
	}); err != nil {
		log.Fatalf("workflow failed: %v", err)
	}
	fmt.Println("  → workflow COMPLETED")
}

func main() {
	fmt.Println("=== 02-branching: loan-approval ===")
	run("alice", 750, 25000)
	run("bob", 620, 10000)
}
