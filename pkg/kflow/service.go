package kflow

import "time"

// ServiceMode controls how a Service is deployed on Kubernetes.
type ServiceMode int

const (
	Deployment ServiceMode = iota // K8s Deployment; minScale >= 1 enforced
	Lambda                        // K8s Job per invocation; scale/port are ignored
)

// ServiceDef describes a persistent or on-demand service.
type ServiceDef struct {
	name        string
	fn          HandlerFunc
	mode        ServiceMode
	port        int
	minScale    int
	maxScale    int
	ingressHost string
	timeout     time.Duration
}

// NewService creates a ServiceDef with defaults: Deployment mode, port 8080, timeout 30 s.
func NewService(name string) *ServiceDef {
	return &ServiceDef{
		name:    name,
		mode:    Deployment,
		port:    8080,
		timeout: 30 * time.Second,
	}
}

// Handler sets the handler function for the service.
func (s *ServiceDef) Handler(fn HandlerFunc) *ServiceDef {
	s.fn = fn
	return s
}

// Mode sets the deployment mode.
func (s *ServiceDef) Mode(m ServiceMode) *ServiceDef {
	s.mode = m
	return s
}

// Port sets the container port.
func (s *ServiceDef) Port(p int) *ServiceDef {
	s.port = p
	return s
}

// Scale sets the minimum and maximum replica counts.
func (s *ServiceDef) Scale(min, max int) *ServiceDef {
	s.minScale = min
	s.maxScale = max
	return s
}

// Expose sets the ingress hostname for the service.
func (s *ServiceDef) Expose(host string) *ServiceDef {
	s.ingressHost = host
	return s
}

// Timeout sets the per-invocation timeout.
func (s *ServiceDef) Timeout(d time.Duration) *ServiceDef {
	s.timeout = d
	return s
}

// Exported accessors — names differ from builder methods to avoid ambiguity.
func (s *ServiceDef) Name() string             { return s.name }
func (s *ServiceDef) Fn() HandlerFunc          { return s.fn }
func (s *ServiceDef) ServiceMode() ServiceMode { return s.mode }
func (s *ServiceDef) ServicePort() int         { return s.port }
func (s *ServiceDef) MinScale() int            { return s.minScale }
func (s *ServiceDef) MaxScale() int            { return s.maxScale }
func (s *ServiceDef) IngressHost() string      { return s.ingressHost }
func (s *ServiceDef) ServiceTimeout() time.Duration { return s.timeout }

// Validate returns ErrScaleMin if the service is in Deployment mode and minScale < 1.
func (s *ServiceDef) Validate() error {
	if s.mode == Deployment && s.minScale < 1 {
		return ErrScaleMin
	}
	return nil
}
