package api

type TestType int

const (
	SMOKE TestType = iota
	CRITICAL_ONLY
	FULL_SUITE
)

var testTypeName = map[TestType]string{
	SMOKE:         "SMOKE",
	CRITICAL_ONLY: "CRITICAL_ONLY",
	FULL_SUITE:    "FULL_SUITE",
}

func (s TestType) String() string {
	return testTypeName[s]
}
