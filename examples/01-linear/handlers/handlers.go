package handlers

import (
	"context"
	"fmt"

	"github.com/pastorenue/kflow/pkg/kflow"
)

const (
	StateValidateOrder    = "ValidateOrder"
	StateCalculateTax     = "CalculateTax"
	StateChargePayment    = "ChargePayment"
	StateSendConfirmation = "SendConfirmation"
)

type OrderHandlers struct{}

func New() *OrderHandlers { return &OrderHandlers{} }

func (h *OrderHandlers) ValidateOrder(_ context.Context, input kflow.Input) (kflow.Output, error) {
	orderID := input["order_id"]
	total, _ := input["total"].(float64)
	if total <= 0 {
		return nil, fmt.Errorf("invalid order total: %v", total)
	}
	fmt.Printf("  ValidateOrder: order=%v total=$%.2f ✓\n", orderID, total)
	return kflow.Output{"order_id": orderID, "total": total, "validated": true}, nil
}

func (h *OrderHandlers) CalculateTax(_ context.Context, input kflow.Input) (kflow.Output, error) {
	total, _ := input["total"].(float64)
	tax := total * 0.08
	fmt.Printf("  CalculateTax: subtotal=$%.2f tax=$%.2f\n", total, tax)
	return kflow.Output{
		"order_id":    input["order_id"],
		"total":       total,
		"tax":         tax,
		"grand_total": total + tax,
	}, nil
}

func (h *OrderHandlers) ChargePayment(_ context.Context, input kflow.Input) (kflow.Output, error) {
	grand, _ := input["grand_total"].(float64)
	fmt.Printf("  ChargePayment: charging $%.2f\n", grand)
	return kflow.Output{
		"order_id":       input["order_id"],
		"grand_total":    grand,
		"payment_status": "captured",
		"txn_id":         "txn_abc123",
	}, nil
}

func (h *OrderHandlers) SendConfirmation(_ context.Context, input kflow.Input) (kflow.Output, error) {
	fmt.Printf("  SendConfirmation: order=%v txn=%v → email sent\n",
		input["order_id"], input["txn_id"])
	return kflow.Output{"order_id": input["order_id"], "confirmed": true}, nil
}
