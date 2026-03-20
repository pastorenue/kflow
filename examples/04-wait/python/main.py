# Example 04: wait state — scheduled notification with a short pause
# ScheduleNotification → Wait(2s) → SendNotification → LogDelivery → end
import sys, os, time
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../sdk/python"))
import kflow

from handlers import (
    STATE_SCHEDULE_NOTIFICATION,
    STATE_WAIT_FOR_SEND_TIME,
    STATE_SEND_NOTIFICATION,
    STATE_LOG_DELIVERY,
    schedule_notification,
    send_notification,
    log_delivery,
)


def main():
    wf = kflow.Workflow("scheduled-notification")
    wf.task(STATE_SCHEDULE_NOTIFICATION)(schedule_notification)
    wf.wait(STATE_WAIT_FOR_SEND_TIME, 2)
    wf.task(STATE_SEND_NOTIFICATION)(send_notification)
    wf.task(STATE_LOG_DELIVERY)(log_delivery)

    wf.flow(
        kflow.step(STATE_SCHEDULE_NOTIFICATION).next(STATE_WAIT_FOR_SEND_TIME),
        kflow.step(STATE_WAIT_FOR_SEND_TIME).next(STATE_SEND_NOTIFICATION),
        kflow.step(STATE_SEND_NOTIFICATION).next(STATE_LOG_DELIVERY),
        kflow.step(STATE_LOG_DELIVERY).end(),
    )

    print("=== 04-wait: scheduled-notification (2s pause) ===")
    start = time.time()
    kflow.run_local(wf, {
        "recipient": "carol@example.com",
        "message":   "Your order ORD-3001 has shipped!",
    })
    print(f"  → workflow COMPLETED in {time.time() - start:.1f}s")


if __name__ == "__main__":
    main()
