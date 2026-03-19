package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/pastorenue/kflow/internal/k8s"
	"github.com/pastorenue/kflow/internal/runner"
	"github.com/pastorenue/kflow/internal/store"
	"github.com/pastorenue/kflow/pkg/kflow"
)

// Telemetry is the interface K8sExecutor uses to record state transitions.
// It is satisfied by telemetry.EventWriter (Phase 6). Nil disables telemetry.
type Telemetry interface {
	RecordStateTransition(ctx context.Context, execID, stateName string, status store.Status)
}

// K8sExecutor drives workflow execution by dispatching each state as a
// Kubernetes Job. It wraps Executor with a K8s-backed Handler.
type K8sExecutor struct {
	Store             store.Store
	K8s               *k8s.Client
	Image             string
	RunnerEndpoint    string // KFLOW_GRPC_ENDPOINT injected into Job containers
	RunnerTokenSecret []byte // HMAC key for state token signing
	Telemetry         Telemetry
}

// Run drives a full workflow execution using K8s Jobs.
func (e *K8sExecutor) Run(ctx context.Context, execID string, g *Graph, input kflow.Input) error {
	ex := &Executor{
		Store:   e.Store,
		Handler: e.buildHandler(ctx, execID),
	}
	return ex.Run(ctx, execID, g, input)
}

// buildHandler returns a HandlerFunc that spawns a K8s Job for each state.
func (e *K8sExecutor) buildHandler(ctx context.Context, execID string) func(context.Context, string, kflow.Input) (kflow.Output, error) {
	return func(ctx context.Context, stateName string, _ kflow.Input) (kflow.Output, error) {
		name := k8s.JobName(execID, stateName)

		tok, err := runner.GenerateStateToken(execID, stateName, 1, e.RunnerTokenSecret)
		if err != nil {
			return nil, fmt.Errorf("k8s_executor: generate token for %q: %w", stateName, err)
		}

		if e.Telemetry != nil {
			e.Telemetry.RecordStateTransition(ctx, execID, stateName, store.StatusRunning)
		}

		_, err = e.K8s.CreateJob(ctx, k8s.JobSpec{
			Name:  name,
			Image: e.Image,
			Args:  []string{"--state=" + stateName},
			Env: []k8s.EnvVar{
				{Name: "KFLOW_EXECUTION_ID", Value: execID},
				{Name: "KFLOW_STATE_TOKEN", Value: tok},
				{Name: "KFLOW_GRPC_ENDPOINT", Value: e.RunnerEndpoint},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("k8s_executor: create job for %q: %w", stateName, err)
		}

		result, err := e.K8s.WaitForJob(ctx, name)
		if err != nil {
			e.deleteJobBestEffort(ctx, name)
			return nil, fmt.Errorf("k8s_executor: wait for job %q: %w", name, err)
		}

		if result.Failed {
			e.deleteJobBestEffort(ctx, name)
			if e.Telemetry != nil {
				e.Telemetry.RecordStateTransition(ctx, execID, stateName, store.StatusFailed)
			}
			return nil, fmt.Errorf("k8s_executor: job for %q failed: %s", stateName, result.Message)
		}

		output, err := e.Store.GetStateOutput(ctx, execID, stateName)
		if err != nil {
			e.deleteJobBestEffort(ctx, name)
			return nil, fmt.Errorf("k8s_executor: get output for %q: %w", stateName, err)
		}

		e.deleteJobBestEffort(ctx, name)
		if e.Telemetry != nil {
			e.Telemetry.RecordStateTransition(ctx, execID, stateName, store.StatusCompleted)
		}
		return output, nil
	}
}

func (e *K8sExecutor) deleteJobBestEffort(ctx context.Context, name string) {
	if err := e.K8s.DeleteJob(ctx, name); err != nil {
		log.Printf("k8s_executor: delete job %q (best-effort): %v", name, err)
	}
}
