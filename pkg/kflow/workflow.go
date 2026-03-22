package kflow

import "time"

// Workflow is the Aggregate Root for a state-machine definition.
type Workflow struct {
	name  string
	image string
	tasks map[string]*TaskDef
	steps []*StepBuilder
	// names preserves insertion order including duplicates so Validate() can
	// detect duplicate registrations (the map silently overwrites).
	names []string
}

// New creates an empty Workflow with the given name.
func New(name string) *Workflow {
	return &Workflow{
		name:  name,
		tasks: make(map[string]*TaskDef),
	}
}

// Task registers a task state with an inline HandlerFunc.
func (w *Workflow) Task(name string, fn HandlerFunc) *TaskDef {
	td := &TaskDef{name: name, fn: fn, kind: taskState}
	w.tasks[name] = td
	w.names = append(w.names, name)
	return td
}

// Choice registers a choice state driven by a ChoiceFunc.
func (w *Workflow) Choice(name string, fn ChoiceFunc) *TaskDef {
	td := &TaskDef{name: name, choiceFn: fn, kind: choiceState}
	w.tasks[name] = td
	w.names = append(w.names, name)
	return td
}

// Wait registers a wait state that pauses execution for dur.
func (w *Workflow) Wait(name string, dur time.Duration) *TaskDef {
	td := &TaskDef{name: name, kind: waitState, waitDur: dur}
	w.tasks[name] = td
	w.names = append(w.names, name)
	return td
}

// Parallel registers a parallel fan-out state.
func (w *Workflow) Parallel(name string, fn HandlerFunc) *TaskDef {
	td := &TaskDef{name: name, fn: fn, kind: parallelState}
	w.tasks[name] = td
	w.names = append(w.names, name)
	return td
}

// Flow sets the ordered step sequence for this workflow.
func (w *Workflow) Flow(steps ...*StepBuilder) *Workflow {
	w.steps = steps
	return w
}

// WithImage sets the container image used to execute this workflow's states as K8s Jobs.
// Leave empty for in-process (pass-through) execution.
func (w *Workflow) WithImage(image string) *Workflow {
	w.image = image
	return w
}

// Name returns the workflow name.
func (w *Workflow) Name() string { return w.name }

// Image returns the container image for K8s Job execution (empty = in-process).
func (w *Workflow) Image() string { return w.image }

// Steps returns the ordered step sequence.
func (w *Workflow) Steps() []*StepBuilder { return w.steps }

// Tasks returns the registered task map (keyed by name).
func (w *Workflow) Tasks() map[string]*TaskDef { return w.tasks }

// TaskDef is the Entity representing a single state in the workflow.
type TaskDef struct {
	name          string
	fn            HandlerFunc
	choiceFn      ChoiceFunc
	serviceTarget string
	retry         *RetryPolicy // nil means "not set"
	catch         string
	kind          stateKind
	waitDur       time.Duration
}

// InvokeService configures the task to dispatch to a named Service instead of
// running an inline handler.
func (t *TaskDef) InvokeService(target string) *TaskDef {
	t.serviceTarget = target
	return t
}

// Retry attaches a retry policy to the task.
func (t *TaskDef) Retry(p RetryPolicy) *TaskDef {
	t.retry = &p
	return t
}

// Catch sets the fallback state name if this task fails.
func (t *TaskDef) Catch(state string) *TaskDef {
	t.catch = state
	return t
}

// Exported accessors used by the engine (internal/engine).
func (t *TaskDef) Name() string          { return t.name }
func (t *TaskDef) Fn() HandlerFunc       { return t.fn }
func (t *TaskDef) ChoiceFn() ChoiceFunc  { return t.choiceFn }
func (t *TaskDef) ServiceTarget() string { return t.serviceTarget }
func (t *TaskDef) RetryPolicy() *RetryPolicy { return t.retry }
func (t *TaskDef) CatchState() string    { return t.catch }
func (t *TaskDef) WaitDur() time.Duration { return t.waitDur }

// State-kind boolean helpers avoid exporting the stateKind type.
func (t *TaskDef) IsChoice() bool   { return t.kind == choiceState }
func (t *TaskDef) IsWait() bool     { return t.kind == waitState }
func (t *TaskDef) IsParallel() bool { return t.kind == parallelState }

// StepBuilder is a Value Object that describes one step in the Flow sequence.
type StepBuilder struct {
	name  string
	next  string
	catch string
	retry *RetryPolicy
	isEnd bool
}

// Step creates a new StepBuilder for the named state.
func Step(name string) *StepBuilder {
	return &StepBuilder{name: name}
}

// Next sets the state to transition to on success.
func (s *StepBuilder) Next(state string) *StepBuilder {
	s.next = state
	return s
}

// Catch sets the state to transition to on failure.
func (s *StepBuilder) Catch(state string) *StepBuilder {
	s.catch = state
	return s
}

// Retry attaches a retry policy at the step level.
func (s *StepBuilder) Retry(p RetryPolicy) *StepBuilder {
	s.retry = &p
	return s
}

// End marks this step as a terminal success transition.
func (s *StepBuilder) End() *StepBuilder {
	s.next = Succeed
	s.isEnd = true
	return s
}

// Exported accessors.
func (s *StepBuilder) Name() string          { return s.name }
func (s *StepBuilder) NextState() string     { return s.next }
func (s *StepBuilder) CatchState() string    { return s.catch }
func (s *StepBuilder) RetryPolicy() *RetryPolicy { return s.retry }
func (s *StepBuilder) IsEnd() bool           { return s.isEnd }
