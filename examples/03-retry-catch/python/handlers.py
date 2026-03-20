STATE_FETCH_EXTERNAL_DATA = "FetchExternalData"
STATE_ENRICH_RECORD       = "EnrichRecord"
STATE_PERSIST             = "Persist"
STATE_HANDLE_FETCH_ERROR  = "HandleFetchError"


class PipelineHandlers:
    """Simulates a flaky external service. succeed_after=0 means always fail."""

    def __init__(self, succeed_after: int):
        self._succeed_after = succeed_after
        self._attempt = 0

    def fetch_external_data(self, input: dict) -> dict:
        self._attempt += 1
        print(f"  FetchExternalData: attempt {self._attempt}")
        if self._succeed_after == 0 or self._attempt < self._succeed_after:
            raise RuntimeError(f"upstream timeout (attempt {self._attempt})")
        print("  FetchExternalData: success")
        return {
            "record_id": input["record_id"],
            "raw":       {"value": 42, "source": "api"},
        }

    def enrich_record(self, input: dict) -> dict:
        raw = input["raw"]
        print(f"  EnrichRecord: enriching record_id={input['record_id']}")
        return {
            "record_id": input["record_id"],
            "enriched":  {"value": raw["value"], "label": "verified", "source": raw["source"]},
        }

    def persist(self, input: dict) -> dict:
        print(f"  Persist: saved record_id={input['record_id']}")
        return {"record_id": input["record_id"], "persisted": True}

    def handle_fetch_error(self, input: dict) -> dict:
        print(f"  HandleFetchError: caught error={input.get('_error')}")
        return {"record_id": input["record_id"], "status": "fetch_failed"}
