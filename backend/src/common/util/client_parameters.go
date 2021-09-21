package util

// ClientParameters contains parameters needed when creating a client
type ClientParameters struct {
	// Use float64 instead of float32 here to use flag.Float64Var() directly
	QPS   float64
	Burst int
}
