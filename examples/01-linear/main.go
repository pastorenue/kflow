// Example 01: linear order-processing pipeline
// ValidateOrder → CalculateTax → ChargePayment → SendConfirmation → end
package main

import (
	"fmt"
	"log"

	"github.com/pastorenue/kflow/examples/01-linear/handlers"
	"github.com/pastorenue/kflow/internal/local"
	"github.com/pastorenue/kflow/pkg/kflow"
)

func main() {
	h := handlers.New()

	wf := kflow.New("order-processing")
	wf.Task("ValidateOrder", h.ValidateOrder)
	wf.Task("CalculateTax", h.CalculateTax)
	wf.Task("ChargePayment", h.ChargePayment)
	wf.Task("SendConfirmation", h.SendConfirmation)

	wf.Flow(
		kflow.Step("ValidateOrder").Next("CalculateTax"),
		kflow.Step("CalculateTax").Next("ChargePayment"),
		kflow.Step("ChargePayment").Next("SendConfirmation"),
		kflow.Step("SendConfirmation").End(),
	)

	fmt.Println("=== 01-linear: order-processing ===")
	if err := local.RunLocal(wf, kflow.Input{
		"order_id": "ORD-2001",
		"customer": "alice",
		"total":    149.99,
	}); err != nil {
		log.Fatalf("workflow failed: %v", err)
	}
	fmt.Println("  → workflow COMPLETED")
}
