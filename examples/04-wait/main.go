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
	wf.Task("ScheduleNotification", h.ScheduleNotification)
	wf.Wait("WaitForSendTime", 2*time.Second)
	wf.Task("SendNotification", h.SendNotification)
	wf.Task("LogDelivery", h.LogDelivery)

	wf.Flow(
		kflow.Step("ScheduleNotification").Next("WaitForSendTime"),
		kflow.Step("WaitForSendTime").Next("SendNotification"),
		kflow.Step("SendNotification").Next("LogDelivery"),
		kflow.Step("LogDelivery").End(),
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
