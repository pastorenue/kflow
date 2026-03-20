package runner

import (
	"context"
	"testing"
	"time"

	kflowv1 "github.com/pastorenue/kflow/internal/gen/kflow/v1"
	"github.com/pastorenue/kflow/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestRunnerServer(t *testing.T) (*RunnerServiceServer, store.Store) {
	t.Helper()
	ms := store.NewMemoryStore()
	return NewRunnerServiceServer(ms, testSecret), ms
}

func makeToken(t *testing.T, execID, state string, attempt int) string {
	t.Helper()
	tok, err := GenerateStateToken(execID, state, attempt, testSecret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return tok
}

func TestRunnerServer_GetInput(t *testing.T) {
	srv, ms := newTestRunnerServer(t)
	ctx := context.Background()

	execID := "exec-get-input"
	if err := ms.CreateExecution(ctx, store.ExecutionRecord{
		ID:        execID,
		Workflow:  "wf",
		Input:     map[string]any{"key": "val"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	tok := makeToken(t, execID, "StateA", 1)
	resp, err := srv.GetInput(ctx, &kflowv1.GetInputRequest{Token: tok})
	if err != nil {
		t.Fatalf("GetInput: %v", err)
	}
	if resp.GetPayload() == nil {
		t.Fatal("expected non-nil payload")
	}
	if resp.GetPayload().GetFields()["key"].GetStringValue() != "val" {
		t.Fatalf("unexpected payload: %v", resp.GetPayload())
	}
}

func TestRunnerServer_CompleteState(t *testing.T) {
	srv, ms := newTestRunnerServer(t)
	ctx := context.Background()

	execID := "exec-complete"
	if err := ms.CreateExecution(ctx, store.ExecutionRecord{
		ID: execID, Workflow: "wf", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ms.WriteAheadState(ctx, store.StateRecord{
		ExecutionID: execID, StateName: "S1", Attempt: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ms.MarkRunning(ctx, execID, "S1"); err != nil {
		t.Fatal(err)
	}

	tok := makeToken(t, execID, "S1", 1)
	output, _ := structpb.NewStruct(map[string]any{"result": "ok"})
	_, err := srv.CompleteState(ctx, &kflowv1.CompleteStateRequest{Token: tok, Output: output})
	if err != nil {
		t.Fatalf("CompleteState: %v", err)
	}

	out, err := ms.GetStateOutput(ctx, execID, "S1")
	if err != nil {
		t.Fatalf("GetStateOutput: %v", err)
	}
	if out["result"] != "ok" {
		t.Fatalf("unexpected output: %v", out)
	}
}

func TestRunnerServer_FailState(t *testing.T) {
	srv, ms := newTestRunnerServer(t)
	ctx := context.Background()

	execID := "exec-fail"
	if err := ms.CreateExecution(ctx, store.ExecutionRecord{
		ID: execID, Workflow: "wf", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ms.WriteAheadState(ctx, store.StateRecord{
		ExecutionID: execID, StateName: "S2", Attempt: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ms.MarkRunning(ctx, execID, "S2"); err != nil {
		t.Fatal(err)
	}

	tok := makeToken(t, execID, "S2", 1)
	_, err := srv.FailState(ctx, &kflowv1.FailStateRequest{Token: tok, ErrorMessage: "boom"})
	if err != nil {
		t.Fatalf("FailState: %v", err)
	}

	states, err := ms.ListStates(ctx, execID)
	if err != nil {
		t.Fatal(err)
	}
	for _, sr := range states {
		if sr.StateName == "S2" && sr.Status == store.StatusFailed {
			return
		}
	}
	t.Fatal("expected S2 to be failed")
}

func TestRunnerServer_InvalidToken(t *testing.T) {
	srv, _ := newTestRunnerServer(t)
	ctx := context.Background()

	_, err := srv.GetInput(ctx, &kflowv1.GetInputRequest{Token: "invalid.token"})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}
