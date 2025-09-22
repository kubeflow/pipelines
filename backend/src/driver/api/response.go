package api

type DriverResponse struct {
	Node Node `json:"node"`
}

type Node struct {
	Phase   string  `json:"phase"`
	Outputs Outputs `json:"outputs"`
	Message string  `json:"message"`
}

type Outputs struct {
	Parameters []Parameter `json:"parameters"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
