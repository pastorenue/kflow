package kflow

// Run registers the workflow with the Control Plane and triggers execution.
// It panics with the validation error if the workflow is invalid.
func Run(wf *Workflow) {
	if err := wf.Validate(); err != nil {
		panic("kflow.Run: invalid workflow: " + err.Error())
	}
	panic("kflow.Run: not implemented")
}

// RunLocal executes the workflow in-process using MemoryStore (no Kubernetes).
// It panics with the validation error if the workflow is invalid.
func RunLocal(wf *Workflow) {
	if err := wf.Validate(); err != nil {
		panic("kflow.RunLocal: invalid workflow: " + err.Error())
	}
	panic("kflow.RunLocal: not implemented")
}

// RunService registers and starts a persistent or on-demand Service.
// It panics with the validation error if the service definition is invalid.
func RunService(svc *ServiceDef) {
	if err := svc.Validate(); err != nil {
		panic("kflow.RunService: invalid service: " + err.Error())
	}
	panic("kflow.RunService: not implemented")
}
