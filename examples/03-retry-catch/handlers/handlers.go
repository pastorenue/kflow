package handlers

import (
	"context"
	"fmt"

	"github.com/pastorenue/kflow/pkg/kflow"
)

// PipelineHandlers simulates a flaky external service.
// succeedAfter == 0 means always fail (triggers catch after retries exhausted).
type PipelineHandlers struct {
	succeedAfter int
	attempt      int
}

func New(succeedAfter int) *PipelineHandlers {
	return &PipelineHandlers{succeedAfter: succeedAfter}
}

func (h *PipelineHandlers) FetchExternalData(_ context.Context, input kflow.Input) (kflow.Output, error) {
	h.attempt++
	fmt.Printf("  FetchExternalData: attempt %d\n", h.attempt)
	if h.succeedAfter == 0 || h.attempt < h.succeedAfter {
		return nil, fmt.Errorf("upstream timeout (attempt %d)", h.attempt)
	}
	fmt.Println("  FetchExternalData: success")
	return kflow.Output{
		"record_id": input["record_id"],
		"raw":       map[string]any{"value": 42, "source": "api"},
	}, nil
}

func (h *PipelineHandlers) EnrichRecord(_ context.Context, input kflow.Input) (kflow.Output, error) {
	raw, _ := input["raw"].(map[string]any)
	fmt.Printf("  EnrichRecord: enriching record_id=%v\n", input["record_id"])
	return kflow.Output{
		"record_id": input["record_id"],
		"enriched":  map[string]any{"value": raw["value"], "label": "verified", "source": raw["source"]},
	}, nil
}

func (h *PipelineHandlers) Persist(_ context.Context, input kflow.Input) (kflow.Output, error) {
	fmt.Printf("  Persist: saved record_id=%v\n", input["record_id"])
	return kflow.Output{"record_id": input["record_id"], "persisted": true}, nil
}

func (h *PipelineHandlers) HandleFetchError(_ context.Context, input kflow.Input) (kflow.Output, error) {
	fmt.Printf("  HandleFetchError: caught error=%v\n", input["_error"])
	return kflow.Output{"record_id": input["record_id"], "status": "fetch_failed"}, nil
}
