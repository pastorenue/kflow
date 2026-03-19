package kflow

import "context"

// HandlerFunc is the signature every task and service handler must implement.
type HandlerFunc func(ctx context.Context, input Input) (Output, error)

// ChoiceFunc returns the name of the next state based on the current input.
type ChoiceFunc func(ctx context.Context, input Input) (string, error)

// RetryPolicy configures automatic retry behaviour for a task.
type RetryPolicy struct {
	MaxAttempts    int // 0 = executor default; negative = invalid
	BackoffSeconds int // 0 = no delay between retries
}

// stateKind classifies a TaskDef; used internally by Validate() and the Phase 2 engine.
type stateKind int

const (
	taskState stateKind = iota
	choiceState
	waitState
	parallelState
)

// Succeed and Fail are sentinel state names.  They are never registered as
// TaskDefs; the engine handles them as terminal transitions.
const (
	Succeed = "__succeed__"
	Fail    = "__fail__"
)

// isSentinel reports whether name is one of the two terminal sentinels.
func isSentinel(name string) bool { return name == Succeed || name == Fail }

