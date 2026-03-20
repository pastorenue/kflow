package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/pastorenue/kflow/pkg/kflow"
)

type NotificationHandlers struct{}

func New() *NotificationHandlers { return &NotificationHandlers{} }

func (h *NotificationHandlers) ScheduleNotification(_ context.Context, input kflow.Input) (kflow.Output, error) {
	recipient := input["recipient"]
	message := input["message"]
	sendAt := time.Now().Add(2 * time.Second)
	fmt.Printf("  ScheduleNotification: to=%v msg=%q send_at=%s\n",
		recipient, message, sendAt.Format(time.RFC3339))
	return kflow.Output{
		"recipient": recipient,
		"message":   message,
		"send_at":   sendAt.Format(time.RFC3339),
		"scheduled": true,
	}, nil
}

func (h *NotificationHandlers) SendNotification(_ context.Context, input kflow.Input) (kflow.Output, error) {
	fmt.Printf("  SendNotification: sending to=%v at %s\n",
		input["recipient"], time.Now().Format(time.RFC3339))
	return kflow.Output{
		"recipient":  input["recipient"],
		"message":    input["message"],
		"sent_at":    time.Now().Format(time.RFC3339),
		"channel":    "email",
		"message_id": "msg_xyz789",
	}, nil
}

func (h *NotificationHandlers) LogDelivery(_ context.Context, input kflow.Input) (kflow.Output, error) {
	fmt.Printf("  LogDelivery: msg_id=%v channel=%v recipient=%v\n",
		input["message_id"], input["channel"], input["recipient"])
	return kflow.Output{
		"message_id": input["message_id"],
		"logged":     true,
	}, nil
}
