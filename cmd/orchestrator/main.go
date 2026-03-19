// Package main is the Control Plane entry point (composition root).
// Flag dispatch selects the execution mode:
//
//	--state=<name>    run a single state handler and exit (K8s Job mode)
//	--service=<name>  run a persistent service deployment (Phase 5)
//	(no flag)         start the Control Plane server
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pastorenue/kflow/internal/config"
	k8sclient "github.com/pastorenue/kflow/internal/k8s"
	"github.com/pastorenue/kflow/internal/store"
)

func main() {
	stateName := flag.String("state", "", "state name to execute (K8s Job mode)")
	serviceName := flag.String("service", "", "service name to run (Phase 5)")
	flag.Parse()

	if *stateName != "" {
		runStateMode(*stateName)
		return
	}

	if *serviceName != "" {
		log.Fatalf("--service mode not yet implemented (Phase 5)")
	}

	runServerMode()
}

// runStateMode is the --state=<name> execution path.
// The binary dials KFLOW_GRPC_ENDPOINT, retrieves the state input via
// RunnerService.GetInput, calls the HandlerFunc, and reports the result.
// RunnerService gRPC protocol is implemented in Phase 13.
func runStateMode(stateName string) {
	execID := requireEnv("KFLOW_EXECUTION_ID")
	stateToken := requireEnv("KFLOW_STATE_TOKEN")
	grpcEndpoint := requireEnv("KFLOW_GRPC_ENDPOINT")

	log.Printf("state mode: execID=%s state=%s endpoint=%s", execID, stateName, grpcEndpoint)
	log.Printf("state mode: token present=%v", stateToken != "")

	// TODO(Phase 13): dial grpcEndpoint, call RunnerService.GetInput(stateToken),
	// look up HandlerFunc for stateName, invoke it, then call
	// RunnerService.CompleteState or RunnerService.FailState.
	log.Fatal("state mode: RunnerService gRPC not yet implemented (Phase 13)")
}

// runServerMode starts the Control Plane server.
func runServerMode() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	ms, err := store.NewMongoStore(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("store: connect: %v", err)
	}
	if err := ms.EnsureIndexes(ctx); err != nil {
		log.Fatalf("store: ensure indexes: %v", err)
	}
	log.Println("store: connected to MongoDB")

	k8s, err := k8sclient.NewClient(cfg.Namespace)
	if err != nil {
		log.Fatalf("k8s: %v", err)
	}
	log.Printf("k8s: client initialised (namespace=%s)", k8s.Namespace())

	// TODO(Phase 5): start HTTP API and gRPC servers.
	log.Println("orchestrator: running (waiting for signal)")
	<-ctx.Done()
	log.Println("orchestrator: shutdown")
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		fmt.Fprintf(os.Stderr, "error: required environment variable %s is not set\n", name)
		os.Exit(1)
	}
	return v
}
