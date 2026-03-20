from datetime import datetime, timezone, timedelta

STATE_SCHEDULE_NOTIFICATION = "ScheduleNotification"
STATE_WAIT_FOR_SEND_TIME    = "WaitForSendTime"
STATE_SEND_NOTIFICATION     = "SendNotification"
STATE_LOG_DELIVERY          = "LogDelivery"


def schedule_notification(input: dict) -> dict:
    recipient = input["recipient"]
    message = input["message"]
    send_at = (datetime.now(timezone.utc) + timedelta(seconds=2)).isoformat()
    print(f"  ScheduleNotification: to={recipient} msg={message!r} send_at={send_at}")
    return {
        "recipient": recipient,
        "message":   message,
        "send_at":   send_at,
        "scheduled": True,
    }


def send_notification(input: dict) -> dict:
    now = datetime.now(timezone.utc).isoformat()
    print(f"  SendNotification: sending to={input['recipient']} at {now}")
    return {
        "recipient":  input["recipient"],
        "message":    input["message"],
        "sent_at":    now,
        "channel":    "email",
        "message_id": "msg_xyz789",
    }


def log_delivery(input: dict) -> dict:
    print(f"  LogDelivery: msg_id={input['message_id']} channel={input['channel']} recipient={input['recipient']}")
    return {"message_id": input["message_id"], "logged": True}
