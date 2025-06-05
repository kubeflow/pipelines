package api

type Severity int

const (
	S1 Severity = iota
	S2
	S3
)

var severityName = map[Severity]string{
	S1: "BLOCKER",
	S2: "CRITICAL",
	S3: "TRIVIAL",
}

func (s Severity) String() string {
	return severityName[s]
}
