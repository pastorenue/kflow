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
	wf.Task(handlers.StateValidateOrder, h.ValidateOrder)
	wf.Task(handlers.StateCalculateTax, h.CalculateTax)
	wf.Task(handlers.StateChargePayment, h.ChargePayment)
	wf.Task(handlers.StateSendConfirmation, h.SendConfirmation)

	wf.Flow(
		kflow.Step(handlers.StateValidateOrder).Next(handlers.StateCalculateTax),
		kflow.Step(handlers.StateCalculateTax).Next(handlers.StateChargePayment),
		kflow.Step(handlers.StateChargePayment).Next(handlers.StateSendConfirmation),
		kflow.Step(handlers.StateSendConfirmation).End(),
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
