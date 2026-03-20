// Example 04: wait state — scheduled notification with a short pause
// ScheduleNotification → Wait(2s) → SendNotification → LogDelivery → end
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/pastorenue/kflow/examples/04-wait/handlers"
	"github.com/pastorenue/kflow/internal/local"
	"github.com/pastorenue/kflow/pkg/kflow"
)

func main() {
	h := handlers.New()

	wf := kflow.New("scheduled-notification")
	wf.Task(handlers.StateScheduleNotification, h.ScheduleNotification)
	wf.Wait(handlers.StateWaitForSendTime, 2*time.Second)
	wf.Task(handlers.StateSendNotification, h.SendNotification)
	wf.Task(handlers.StateLogDelivery, h.LogDelivery)

	wf.Flow(
		kflow.Step(handlers.StateScheduleNotification).Next(handlers.StateWaitForSendTime),
		kflow.Step(handlers.StateWaitForSendTime).Next(handlers.StateSendNotification),
		kflow.Step(handlers.StateSendNotification).Next(handlers.StateLogDelivery),
		kflow.Step(handlers.StateLogDelivery).End(),
	)

	fmt.Println("=== 04-wait: scheduled-notification (2s pause) ===")
	start := time.Now()
	if err := local.RunLocal(wf, kflow.Input{
		"recipient": "carol@example.com",
		"message":   "Your order ORD-3001 has shipped!",
	}); err != nil {
		log.Fatalf("workflow failed: %v", err)
	}
	fmt.Printf("  → workflow COMPLETED in %.1fs\n", time.Since(start).Seconds())
}
