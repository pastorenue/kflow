# Example 03: retry policy + catch state — flaky external API call
# FetchExternalData (retry ×3) → EnrichRecord → Persist → end
#                ↓ catch
# HandleFetchError → end
import sys, os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "../../../sdk/python"))
import kflow

from handlers import (
    STATE_FETCH_EXTERNAL_DATA,
    STATE_ENRICH_RECORD,
    STATE_PERSIST,
    STATE_HANDLE_FETCH_ERROR,
    PipelineHandlers,
)


def build_workflow(h: PipelineHandlers):
    wf = kflow.Workflow("data-pipeline")
    wf.task(STATE_FETCH_EXTERNAL_DATA)(h.fetch_external_data)
    wf.task(STATE_ENRICH_RECORD)(h.enrich_record)
    wf.task(STATE_PERSIST)(h.persist)
    wf.task(STATE_HANDLE_FETCH_ERROR)(h.handle_fetch_error)

    wf.flow(
        kflow.step(STATE_FETCH_EXTERNAL_DATA)
            .next(STATE_ENRICH_RECORD)
            .catch(STATE_HANDLE_FETCH_ERROR)
            .retry(3, 0),
        kflow.step(STATE_ENRICH_RECORD).next(STATE_PERSIST),
        kflow.step(STATE_PERSIST).end(),
        kflow.step(STATE_HANDLE_FETCH_ERROR).end(),
    )
    return wf


def main():
    print("=== 03-retry-catch: data-pipeline ===")
    kflow.run_local(build_workflow(PipelineHandlers(succeed_after=2)), {"record_id": "REC-9900"})
    print("  → workflow COMPLETED")

    print("\n--- triggering catch (all retries exhausted) ---")
    kflow.run_local(build_workflow(PipelineHandlers(succeed_after=0)), {"record_id": "REC-9901"})
    print("  → workflow COMPLETED (via catch)")


if __name__ == "__main__":
    main()
