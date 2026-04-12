package main

type ExecutionPaths struct {
	ExecutionID    string
	IterationCount string
	// DriverOutputsDir is the directory where the driver writes output files
	// for the launcher to read (used in init-container mode).
	DriverOutputsDir string
	// Legacy output paths kept for the standalone driver pod (dummy images).
	CachedDecision string
	Condition      string
	PodSpecPatch   string
}
