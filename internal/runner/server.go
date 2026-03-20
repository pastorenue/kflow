package runner

import (
	"context"
	"fmt"

	kflowv1 "github.com/pastorenue/kflow/internal/gen/kflow/v1"
	"github.com/pastorenue/kflow/internal/store"
	"github.com/pastorenue/kflow/pkg/kflow"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// RunnerServiceServer handles gRPC callbacks from K8s Job containers.
// It is the sole caller of store.CompleteState and store.FailState for
// K8s-executed states.
type RunnerServiceServer struct {
	store  store.Store
	secret []byte
	kflowv1.UnimplementedRunnerServiceServer
}

// NewRunnerServiceServer creates a RunnerServiceServer.
func NewRunnerServiceServer(st store.Store, secret []byte) *RunnerServiceServer {
	return &RunnerServiceServer{store: st, secret: secret}
}

// GetInput validates the token and returns the state's input as a proto Struct.
func (s *RunnerServiceServer) GetInput(ctx context.Context, req *kflowv1.GetInputRequest) (*kflowv1.GetInputResponse, error) {
	payload, err := s.validateToken(req.GetToken())
	if err != nil {
		return nil, err
	}

	rec, err := s.store.GetExecution(ctx, payload.ExecID)
	if err == store.ErrExecutionNotFound {
		return nil, status.Errorf(codes.NotFound, "execution %q not found", payload.ExecID)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get execution: %v", err)
	}

	// Try to get the state's own input from the store first.
	// Fall back to the execution's top-level input.
	var inputMap kflow.Input
	states, err := s.store.ListStates(ctx, payload.ExecID)
	if err == nil {
		for _, sr := range states {
			if sr.StateName == payload.State {
				if sr.Input != nil {
					inputMap = sr.Input
				}
				break
			}
		}
	}
	if inputMap == nil {
		inputMap = rec.Input
	}

	st, err := structpb.NewStruct(inputMap)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode input: %v", err)
	}
	return &kflowv1.GetInputResponse{Payload: st}, nil
}

// CompleteState validates the token and marks the state as completed with output.
func (s *RunnerServiceServer) CompleteState(ctx context.Context, req *kflowv1.CompleteStateRequest) (*kflowv1.CompleteStateResponse, error) {
	payload, err := s.validateToken(req.GetToken())
	if err != nil {
		return nil, err
	}

	var output kflow.Output
	if req.GetOutput() != nil {
		output = req.GetOutput().AsMap()
	}

	if err := s.store.CompleteState(ctx, payload.ExecID, payload.State, output); err != nil {
		return nil, status.Errorf(codes.Internal, "complete state: %v", err)
	}
	return &kflowv1.CompleteStateResponse{}, nil
}

// FailState validates the token and marks the state as failed.
func (s *RunnerServiceServer) FailState(ctx context.Context, req *kflowv1.FailStateRequest) (*kflowv1.FailStateResponse, error) {
	payload, err := s.validateToken(req.GetToken())
	if err != nil {
		return nil, err
	}

	if err := s.store.FailState(ctx, payload.ExecID, payload.State, req.GetErrorMessage()); err != nil {
		return nil, status.Errorf(codes.Internal, "fail state: %v", err)
	}
	return &kflowv1.FailStateResponse{}, nil
}

func (s *RunnerServiceServer) validateToken(token string) (TokenPayload, error) {
	if token == "" {
		return TokenPayload{}, status.Error(codes.Unauthenticated, "missing state token")
	}
	payload, err := ValidateStateToken(token, s.secret)
	if err != nil {
		return TokenPayload{}, status.Errorf(codes.Unauthenticated, fmt.Sprintf("invalid state token: %v", err))
	}
	return payload, nil
}
