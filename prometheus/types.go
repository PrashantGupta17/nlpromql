package prometheus

// Metric represents a Prometheus metric with its labels and value.
type Metric struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

// AllMetricsResult represents the response from the Prometheus /api/v1/label/__name__/values endpoint.
type AllMetricsResult struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

// AllMetricsResult represents the response from the Prometheus /api/v1/labels endpoint.
type AllLabelsResult struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}
