package info_structure

import (
	"github.com/prashantgupta17/nlpromql/openai"
	"github.com/prashantgupta17/nlpromql/prometheus"
)

// MapForJSON represents a map that can be directly serialized to JSON.
type MapForJSON map[string]interface{}

// InfoStructure represents the structure for storing metric and label maps.
type InfoStructure struct {
	MetricMap       *MetricMap
	LabelMap        *LabelMap
	MetricLabelMap  *MetricLabelMap
	LabelValueMap   *LabelValueMap
	NlpToMetricMap  *NlpToMetricMap
	QueryEngine     QueryEngine
	OpenAIClient    *openai.OpenAIClient
	InfoLoaderSaver InfoLoaderSaver
}

// MetricMap represents a map of metric tokens to metric names.
type MetricMap struct {
	Map      map[string]map[string]struct{} `json:"map"`
	AllNames map[string]struct{}            `json:"all_names"`
}

// LabelMap represents a map of label tokens to label names.
type LabelMap struct {
	Map      map[string]map[string]struct{} `json:"map"`
	AllNames map[string]struct{}            `json:"all_names"`
}

// MetricMap represents a map of metric tokens to metric names.
type MetricJsonMap struct {
	Map      map[string][]string `json:"map"`
	AllNames []string            `json:"all_names"`
}

// LabelMap represents a map of label tokens to label names.
type LabelJsonMap struct {
	Map      map[string][]string `json:"map"`
	AllNames []string            `json:"all_names"`
}

// MetricInfo holds information about a metric, including its labels.
type MetricInfo struct {
	Labels map[string]LabelInfo `json:"labels"`
}

// LabelInfo holds information about a label, including its values.
type LabelInfo struct {
	Values map[string]struct{} `json:"values"`
}

// MetricLabelMap represents a map of metric names to their labels and values (sets).
type MetricLabelMap map[string]MetricInfo // Nested map: metric -> label -> value set

// LabelValueMap represents a map of label names to their values (sets).
type LabelValueMap map[string]LabelInfo // Nested map: label -> value set

// NlpToMetricMap represents a map of natural language queries to relevant metric-label pairs.
type NlpToMetricMap map[string]string // Map: natural language query -> metric-label pair

// QueryInterface defines the operations for querying metrics and labels.
type QueryEngine interface {
	// allMetrics returns a list of all metric names.
	AllMetrics() ([]string, error)

	// allLabels returns a list of all label names.
	AllLabels() ([]string, error)

	// instantQuery performs a query at a single point in time and returns the result.
	CustomQuery(query string) ([]prometheus.Metric, error)

	// allMetadata returns all metadata for the Prometheus instance.
	AllMetadata() (map[string]string, error)
}

// InfoStructureManager represents the manager for InfoStructure and its maps.
type InfoStructureManager struct {
	PathToMetricMap      string
	PathToLabelMap       string
	PathToMetricLabelMap string
	PathToLabelValueMap  string
	PathToNlpToMetricMap string
}

// InfoLoaderSaver defines the operations for loading and saving the InfoStructure maps.
type InfoLoaderSaver interface {
	// LoadInfoStructure loads all the maps in the InfoStructureManager.
	LoadInfoStructure() (MetricMap, LabelMap, MetricLabelMap, LabelValueMap, NlpToMetricMap, error)

	// SaveInfoStructure saves all the maps in the InfoStructureManager.
	SaveInfoStructure(metricMap MetricMap, labelMap LabelMap, metricLabelMap MetricLabelMap, labelValueMap LabelValueMap, nlpToMetricMap NlpToMetricMap) error
}
