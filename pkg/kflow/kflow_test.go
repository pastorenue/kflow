package kflow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pastorenue/kflow/pkg/kflow"
)

// noopHandler is a valid HandlerFunc for test fixtures.
var noopHandler kflow.HandlerFunc = func(_ context.Context, _ kflow.Input) (kflow.Output, error) {
	return kflow.Output{}, nil
}

// --- Validation error cases ---

func TestValidate_NoEntryPoint(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", noopHandler)
	// No Flow() call
	if err := wf.Validate(); !errors.Is(err, kflow.ErrNoEntryPoint) {
		t.Fatalf("want ErrNoEntryPoint, got %v", err)
	}
}

func TestValidate_DuplicateName(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", noopHandler)
	wf.Task("A", noopHandler) // duplicate
	wf.Flow(kflow.Step("A").End())
	if err := wf.Validate(); !errors.Is(err, kflow.ErrDuplicateName) {
		t.Fatalf("want ErrDuplicateName, got %v", err)
	}
}

func TestValidate_AmbiguousHandler(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", noopHandler).InvokeService("my-svc")
	wf.Flow(kflow.Step("A").End())
	if err := wf.Validate(); !errors.Is(err, kflow.ErrAmbiguousHandler) {
		t.Fatalf("want ErrAmbiguousHandler, got %v", err)
	}
}

func TestValidate_MissingHandler(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", nil) // no handler, no InvokeService
	wf.Flow(kflow.Step("A").End())
	if err := wf.Validate(); !errors.Is(err, kflow.ErrMissingHandler) {
		t.Fatalf("want ErrMissingHandler, got %v", err)
	}
}

func TestValidate_UnknownNextState(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", noopHandler)
	wf.Flow(kflow.Step("A").Next("DoesNotExist"))
	if err := wf.Validate(); !errors.Is(err, kflow.ErrUnknownState) {
		t.Fatalf("want ErrUnknownState, got %v", err)
	}
}

func TestValidate_UnknownCatchState(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", noopHandler)
	wf.Flow(kflow.Step("A").End().Catch("DoesNotExist"))
	if err := wf.Validate(); !errors.Is(err, kflow.ErrUnknownState) {
		t.Fatalf("want ErrUnknownState, got %v", err)
	}
}

// --- ServiceDef validation ---

func TestServiceDef_ScaleMin(t *testing.T) {
	svc := kflow.NewService("my-svc").
		Handler(noopHandler).
		Mode(kflow.Deployment).
		Scale(0, 5) // minScale = 0 → invalid for Deployment
	if err := svc.Validate(); !errors.Is(err, kflow.ErrScaleMin) {
		t.Fatalf("want ErrScaleMin, got %v", err)
	}
}

func TestServiceDef_LambdaIgnoresScale(t *testing.T) {
	svc := kflow.NewService("my-svc").
		Handler(noopHandler).
		Mode(kflow.Lambda).
		Scale(0, 0) // Lambda does not enforce minScale
	if err := svc.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestServiceDef_ValidScale(t *testing.T) {
	svc := kflow.NewService("my-svc").
		Handler(noopHandler).
		Scale(1, 3)
	if err := svc.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

// --- Happy paths ---

func TestValidate_ValidWorkflow(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", noopHandler)
	wf.Task("B", noopHandler)
	wf.Flow(
		kflow.Step("A").Next("B"),
		kflow.Step("B").End(),
	)
	if err := wf.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestValidate_InvokeServiceOnly(t *testing.T) {
	wf := kflow.New("wf")
	wf.Task("A", nil).InvokeService("payment-svc")
	wf.Flow(kflow.Step("A").End())
	if err := wf.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestValidate_ChoiceWaitParallelExempt(t *testing.T) {
	noopChoice := func(_ context.Context, _ kflow.Input) (string, error) {
		return kflow.Succeed, nil
	}
	wf := kflow.New("wf")
	wf.Choice("C", noopChoice)
	wf.Wait("W", 5*time.Second)
	wf.Parallel("P", noopHandler)
	wf.Flow(
		kflow.Step("C").Next("W"),
		kflow.Step("W").Next("P"),
		kflow.Step("P").End(),
	)
	// None of these should trigger MissingHandler or AmbiguousHandler
	if err := wf.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

// --- StepBuilder helpers ---

func TestStepBuilder_End(t *testing.T) {
	s := kflow.Step("final").End()
	if s.NextState() != kflow.Succeed {
		t.Fatalf("want NextState==%q, got %q", kflow.Succeed, s.NextState())
	}
	if !s.IsEnd() {
		t.Fatal("want IsEnd()==true")
	}
}
