package initialization

import (
	"flag"
	"time"
)

var namespace = flag.String("namespace", "kubeflow", "The namespace ml pipeline deployed to")
var initializeTimeout = flag.Duration("initializeTimeout", 2*time.Minute, "Duration to wait for test initialization")
var runIntegrationTests = flag.Bool("runIntegrationTests", false, "Whether to also run integration tests that call the service")
