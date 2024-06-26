package info_structure

// MetricMap represents a map of metric tokens to metric names.
type MetricMap struct {
	Map      map[string][]string `json:"map"`
	AllNames []string            `json:"all_names"`
}

// LabelMap represents a map of label tokens to label names.
type LabelMap struct {
	Map      map[string][]string `json:"map"`
	AllNames []string            `json:"all_names"`
}

// MetricLabelMap represents a map of metric names to their labels and values (sets).
type MetricLabelMap map[string]map[string]map[string]struct{} // Nested map: metric -> label -> value set

// LabelValueMap represents a map of label names to their values (sets).
type LabelValueMap map[string]map[string]struct{} // Nested map: label -> value set

// NlpToMetricMap represents a map of natural language queries to relevant metric-label pairs.
type NlpToMetricMap map[string]string // Map: natural language query -> metric-label pair
