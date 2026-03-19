package kflow

import "errors"

var (
	ErrDuplicateName    = errors.New("kflow: duplicate state or service name")
	ErrMissingHandler   = errors.New("kflow: task has no handler and no InvokeService target")
	ErrAmbiguousHandler = errors.New("kflow: task has both inline handler and InvokeService target")
	ErrUnknownState     = errors.New("kflow: reference to unknown state name")
	ErrNoEntryPoint     = errors.New("kflow: workflow has no entry point (call Flow first)")
	ErrScaleMin         = errors.New("kflow: Deployment-mode service must have min scale >= 1")
)

// Validate checks the workflow definition for correctness.  Rules are applied
// in order; the first violation encountered is returned.
func (w *Workflow) Validate() error {
	// Rule 1: entry point
	if len(w.steps) == 0 {
		return ErrNoEntryPoint
	}

	// Rule 2: duplicate names (use w.names slice, not the map)
	seen := make(map[string]struct{}, len(w.names))
	for _, name := range w.names {
		if _, exists := seen[name]; exists {
			return ErrDuplicateName
		}
		seen[name] = struct{}{}
	}

	// Rule 3: handler consistency (task states only)
	for _, td := range w.tasks {
		if td.kind != taskState {
			continue
		}
		if td.fn != nil && td.serviceTarget != "" {
			return ErrAmbiguousHandler
		}
		if td.fn == nil && td.serviceTarget == "" {
			return ErrMissingHandler
		}
	}

	// Rule 4: unknown state references in the step sequence
	for _, step := range w.steps {
		if next := step.next; next != "" && !isSentinel(next) {
			if _, ok := w.tasks[next]; !ok {
				return ErrUnknownState
			}
		}
		if catch := step.catch; catch != "" && !isSentinel(catch) {
			if _, ok := w.tasks[catch]; !ok {
				return ErrUnknownState
			}
		}
	}

	return nil
}
