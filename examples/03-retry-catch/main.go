// Example 03: retry policy + catch state — flaky external API call
// FetchExternalData (retry ×3) → EnrichRecord → Persist → end
//
//	↓ catch
//
// HandleFetchError → end
package main

import (
	"fmt"
	"log"

	"github.com/pastorenue/kflow/examples/03-retry-catch/handlers"
	"github.com/pastorenue/kflow/internal/local"
	"github.com/pastorenue/kflow/pkg/kflow"
)

func buildWorkflow(h *handlers.PipelineHandlers) *kflow.Workflow {
	wf := kflow.New("data-pipeline")

	wf.Task("FetchExternalData", h.FetchExternalData).
		Retry(kflow.RetryPolicy{MaxAttempts: 3, BackoffSeconds: 0}).
		Catch("HandleFetchError")

	wf.Task("EnrichRecord", h.EnrichRecord)
	wf.Task("Persist", h.Persist)
	wf.Task("HandleFetchError", h.HandleFetchError)

	wf.Flow(
		kflow.Step("FetchExternalData").Next("EnrichRecord").Catch("HandleFetchError"),
		kflow.Step("EnrichRecord").Next("Persist"),
		kflow.Step("Persist").End(),
		kflow.Step("HandleFetchError").End(),
	)
	return wf
}

func main() {
	fmt.Println("=== 03-retry-catch: data-pipeline ===")
	if err := local.RunLocal(buildWorkflow(handlers.New(2)), kflow.Input{"record_id": "REC-9900"}); err != nil {
		log.Fatalf("workflow failed: %v", err)
	}
	fmt.Println("  → workflow COMPLETED")

	fmt.Println("\n--- triggering catch (all retries exhausted) ---")
	if err := local.RunLocal(buildWorkflow(handlers.New(0)), kflow.Input{"record_id": "REC-9901"}); err != nil {
		log.Fatalf("workflow failed: %v", err)
	}
	fmt.Println("  → workflow COMPLETED (via catch)")
}
