package handlers

import (
	"context"
	"fmt"

	"github.com/pastorenue/kflow/pkg/kflow"
)

const (
	StateAssessRisk    = "AssessRisk"
	StateRouteDecision = "RouteDecision"
	StateApproveLoan   = "ApproveLoan"
	StateRejectLoan    = "RejectLoan"
)

type LoanHandlers struct{}

func New() *LoanHandlers { return &LoanHandlers{} }

func (h *LoanHandlers) AssessRisk(_ context.Context, input kflow.Input) (kflow.Output, error) {
	score, _ := input["credit_score"].(float64)
	applicant := input["applicant"]
	fmt.Printf("  AssessRisk: applicant=%v credit_score=%.0f\n", applicant, score)
	return kflow.Output{
		"applicant":    applicant,
		"credit_score": score,
		"amount":       input["amount"],
	}, nil
}

func (h *LoanHandlers) RouteDecision(_ context.Context, input kflow.Input) (string, error) {
	score, _ := input["credit_score"].(float64)
	if score >= 700 {
		return StateApproveLoan, nil
	}
	return StateRejectLoan, nil
}

func (h *LoanHandlers) ApproveLoan(_ context.Context, input kflow.Input) (kflow.Output, error) {
	amount, _ := input["amount"].(float64)
	fmt.Printf("  ApproveLoan: %v approved for $%.2f\n", input["applicant"], amount)
	return kflow.Output{
		"applicant": input["applicant"],
		"decision":  "approved",
		"amount":    amount,
		"rate":      0.045,
	}, nil
}

func (h *LoanHandlers) RejectLoan(_ context.Context, input kflow.Input) (kflow.Output, error) {
	fmt.Printf("  RejectLoan: %v rejected (score=%.0f)\n",
		input["applicant"], input["credit_score"])
	return kflow.Output{
		"applicant": input["applicant"],
		"decision":  "rejected",
		"reason":    "credit score below threshold",
	}, nil
}
