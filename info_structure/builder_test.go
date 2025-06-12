package info_structure_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/prashantgupta17/nlpromql/info_structure"
	"github.com/prashantgupta17/nlpromql/llm"
	"github.com/prashantgupta17/nlpromql/prometheus" // Added for prometheus.Metric type
)

// --- Mocks ---

// MockLLMClient for builder tests
type MockLLMClient_BuilderTest struct {
	GetMetricSynonymsFunc func(metricBatches []map[string]string) (map[string][]string, error)
	GetLabelSynonymsFunc  func(labelBatches [][]string) (map[string][]string, error)

	// Store received batches
	ReceivedMetricBatches []map[string]string
	ReceivedLabelBatches  [][]string
}

func (m *MockLLMClient_BuilderTest) GetMetricSynonyms(metricBatches []map[string]string) (map[string][]string, error) {
	m.ReceivedMetricBatches = metricBatches
	if m.GetMetricSynonymsFunc != nil {
		return m.GetMetricSynonymsFunc(metricBatches)
	}
	return make(map[string][]string), nil // Default happy path response
}

func (m *MockLLMClient_BuilderTest) GetLabelSynonyms(labelBatches [][]string) (map[string][]string, error) {
	m.ReceivedLabelBatches = labelBatches
	if m.GetLabelSynonymsFunc != nil {
		return m.GetLabelSynonymsFunc(labelBatches)
	}
	return make(map[string][]string), nil // Default happy path response
}

// Implement other llm.LLMClient methods if needed by the code paths being tested, otherwise panic or return defaults.
func (m *MockLLMClient_BuilderTest) ProcessUserQuery(userQuery string) (map[string]interface{}, error) {
	panic("ProcessUserQuery not implemented in MockLLMClient_BuilderTest")
}

func (m *MockLLMClient_BuilderTest) GetPromQLFromLLM(userQuery string, relevantMetrics llm.RelevantMetricsMap, relevantLabels llm.RelevantLabelsMap, relevantHistory map[string]interface{}) ([]string, error) {
	panic("GetPromQLFromLLM not implemented in MockLLMClient_BuilderTest")
}

func (m *MockLLMClient_BuilderTest) Reset() {
	m.ReceivedMetricBatches = nil
	m.ReceivedLabelBatches = nil
}

var _ llm.LLMClient = (*MockLLMClient_BuilderTest)(nil)

// MockQueryEngine for builder tests
type MockQueryEngine_BuilderTest struct {
	AllMetricsFunc    func() ([]string, error)
	AllMetadataFunc   func() (map[string]string, error)
	AllLabelsFunc     func() ([]string, error)
	CustomQueryFunc   func(query string) ([]prometheus.Metric, error)
}

func (m *MockQueryEngine_BuilderTest) AllMetrics() ([]string, error) {
	if m.AllMetricsFunc != nil {
		return m.AllMetricsFunc()
	}
	return []string{}, nil
}

func (m *MockQueryEngine_BuilderTest) AllMetadata() (map[string]string, error) {
	if m.AllMetadataFunc != nil {
		return m.AllMetadataFunc()
	}
	return make(map[string]string), nil
}

func (m *MockQueryEngine_BuilderTest) AllLabels() ([]string, error) {
	if m.AllLabelsFunc != nil {
		return m.AllLabelsFunc()
	}
	return []string{}, nil
}

func (m *MockQueryEngine_BuilderTest) CustomQuery(query string) ([]prometheus.Metric, error) {
	if m.CustomQueryFunc != nil {
		return m.CustomQueryFunc(query)
	}
	return []prometheus.Metric{}, nil
}

var _ info_structure.QueryEngine = (*MockQueryEngine_BuilderTest)(nil)

// MockInfoLoaderSaver for builder tests
type MockInfoLoaderSaver_BuilderTest struct {
	LoadInfoStructureFunc func() (info_structure.MetricMap, info_structure.LabelMap, info_structure.MetricLabelMap, info_structure.LabelValueMap, info_structure.NlpToMetricMap, error)
	SaveInfoStructureFunc func(metricMap info_structure.MetricMap, labelMap info_structure.LabelMap, metricLabelMap info_structure.MetricLabelMap, labelValueMap info_structure.LabelValueMap, nlpToMetricMap info_structure.NlpToMetricMap) error
}

func (m *MockInfoLoaderSaver_BuilderTest) LoadInfoStructure() (info_structure.MetricMap, info_structure.LabelMap, info_structure.MetricLabelMap, info_structure.LabelValueMap, info_structure.NlpToMetricMap, error) {
	if m.LoadInfoStructureFunc != nil {
		return m.LoadInfoStructureFunc()
	}
	// Return empty, initialized maps to avoid nil pointer issues in the functions under test
	return info_structure.MetricMap{Map: make(map[string]map[string]struct{}), AllNames: make(map[string]struct{})},
		info_structure.LabelMap{Map: make(map[string]map[string]struct{}), AllNames: make(map[string]struct{})},
		make(info_structure.MetricLabelMap),
		make(info_structure.LabelValueMap),
		make(info_structure.NlpToMetricMap),
		nil
}

func (m *MockInfoLoaderSaver_BuilderTest) SaveInfoStructure(metricMap info_structure.MetricMap, labelMap info_structure.LabelMap, metricLabelMap info_structure.MetricLabelMap, labelValueMap info_structure.LabelValueMap, nlpToMetricMap info_structure.NlpToMetricMap) error {
	if m.SaveInfoStructureFunc != nil {
		return m.SaveInfoStructureFunc(metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap)
	}
	return nil
}

var _ info_structure.InfoLoaderSaver = (*MockInfoLoaderSaver_BuilderTest)(nil)

// --- Tests ---

func TestUpdateMetricMap_Batching(t *testing.T) {
	const metricBatchSize = 10 // Must match the constant in builder.go

	tests := []struct {
		name                   string
		existingMetricNames    map[string]struct{}
		allMetricNamesFromProm []string
		allMetricDescriptions  map[string]string
		expectedBatches        []map[string]string
		expectLLMCall          bool
	}{
		{
			name:                   "no new metrics",
			existingMetricNames:    map[string]struct{}{"metric1": {}},
			allMetricNamesFromProm: []string{"metric1"},
			allMetricDescriptions:  map[string]string{"metric1": "desc1"},
			expectedBatches:        nil,
			expectLLMCall:          false,
		},
		{
			name:                   "new metrics less than batch size",
			existingMetricNames:    map[string]struct{}{},
			allMetricNamesFromProm: []string{"metric1", "metric2"},
			allMetricDescriptions:  map[string]string{"metric1": "desc1", "metric2": "desc2"},
			expectedBatches:        []map[string]string{{"metric1": "desc1", "metric2": "desc2"}},
			expectLLMCall:          true,
		},
		{
			name:                   "new metrics equal to batch size",
			existingMetricNames:    map[string]struct{}{},
			allMetricNamesFromProm: generateMetrics(metricBatchSize, 0),
			allMetricDescriptions:  generateMetricDescs(metricBatchSize, 0),
			expectedBatches:        []map[string]string{generateMetricDescs(metricBatchSize, 0)},
			expectLLMCall:          true,
		},
		{
			name:                   "new metrics more than batch size (not multiple)",
			existingMetricNames:    map[string]struct{}{},
			allMetricNamesFromProm: generateMetrics(metricBatchSize+2, 0),
			allMetricDescriptions:  generateMetricDescs(metricBatchSize+2, 0),
			expectedBatches: []map[string]string{
				generateMetricDescs(metricBatchSize, 0), // First batch full
				generateMetricDescs(2, metricBatchSize), // Second batch with remaining 2
			},
			expectLLMCall: true,
		},
		{
			name:                   "new metrics more than batch size (multiple)",
			existingMetricNames:    map[string]struct{}{},
			allMetricNamesFromProm: generateMetrics(metricBatchSize*2, 0),
			allMetricDescriptions:  generateMetricDescs(metricBatchSize*2, 0),
			expectedBatches: []map[string]string{
				generateMetricDescs(metricBatchSize, 0),
				generateMetricDescs(metricBatchSize, metricBatchSize),
			},
			expectLLMCall: true,
		},
		{
			name:                "some new, some existing metrics",
			existingMetricNames: map[string]struct{}{"metric_existing_0": {}}, // metric_existing_0 exists
			allMetricNamesFromProm: append(
				[]string{"metric_existing_0", "metric_new_1"}, // metric_new_1 is new
				generateMetrics(metricBatchSize-1, 2, "metric_new_")..., // metric_new_2, ..., metric_new_BATCHSIZE are new
			),
			allMetricDescriptions: func() map[string]string {
				desc := map[string]string{
					"metric_existing_0": "desc_e0",
					"metric_new_1":      "desc_n1", // Specific description
				}
				// Add descriptions for metric_new_2 to metric_new_BATCHSIZE
				for i := 2; i <= metricBatchSize; i++ {
					name := fmt.Sprintf("metric_new_%d", i)
					desc[name] = fmt.Sprintf("description for %s", name)
				}
				return desc
			}(),
			expectedBatches: []map[string]string{
				func() map[string]string {
					batch := map[string]string{
						"metric_new_1": "desc_n1",
					}
					for i := 2; i <= metricBatchSize; i++ {
						name := fmt.Sprintf("metric_new_%d", i)
						batch[name] = fmt.Sprintf("description for %s", name)
					}
					return batch
				}(),
			},
			expectLLMCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &MockLLMClient_BuilderTest{}
			mockQueryEngine := &MockQueryEngine_BuilderTest{
				AllMetricsFunc:  func() ([]string, error) { return tt.allMetricNamesFromProm, nil },
				AllMetadataFunc: func() (map[string]string, error) { return tt.allMetricDescriptions, nil },
			}
			mockLoaderSaver := &MockInfoLoaderSaver_BuilderTest{}

			is, err := info_structure.NewInfoBuilder(mockQueryEngine, mockLLM, mockLoaderSaver)
			if err != nil {
				t.Fatalf("NewInfoBuilder returned an unexpected error: %v", err)
			}
			if is == nil {
				t.Fatalf("NewInfoBuilder returned a nil InfoStructure instance")
			}

			// Manually initialize maps as BuildInformationStructure would do
			metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap, loadErr := mockLoaderSaver.LoadInfoStructure()
			if loadErr != nil {
				t.Fatalf("mockLoaderSaver.LoadInfoStructure() returned an error: %v", loadErr)
			}
			is.MetricMap = &metricMap
			is.LabelMap = &labelMap
			is.MetricLabelMap = &metricLabelMap
			is.LabelValueMap = &labelValueMap
			is.NlpToMetricMap = &nlpToMetricMap

			if is.MetricMap == nil {
				t.Fatalf("is.MetricMap is nil after manual initialization")
			}

			// Pre-populate existing metrics
			is.MetricMap.AllNames = tt.existingMetricNames
			// is.MetricMap.Map is initialized by the mockLoaderSaver.LoadInfoStructure

			err = is.UpdateMetricMap(tt.allMetricNamesFromProm, tt.allMetricDescriptions)
			if err != nil {
				t.Fatalf("UpdateMetricMap returned an unexpected error: %v", err)
			}

			if !tt.expectLLMCall {
				if len(mockLLM.ReceivedMetricBatches) != 0 {
					t.Errorf("expected GetMetricSynonyms not to be called, but it was called with %d batches", len(mockLLM.ReceivedMetricBatches))
				}
				return
			}

			// Compare batches in an order-insensitive way for the outer slice
			if len(mockLLM.ReceivedMetricBatches) != len(tt.expectedBatches) {
				t.Errorf("Expected %d batches, got %d.\nExpected: %v\nGot:      %v", len(tt.expectedBatches), len(mockLLM.ReceivedMetricBatches), tt.expectedBatches, mockLLM.ReceivedMetricBatches)
			} else {
				foundMatch := make([]bool, len(tt.expectedBatches))
				for _, receivedBatch := range mockLLM.ReceivedMetricBatches {
					matchFoundForThisReceivedBatch := false
					for i, expectedBatch := range tt.expectedBatches {
						if !foundMatch[i] && reflect.DeepEqual(receivedBatch, expectedBatch) {
							foundMatch[i] = true
							matchFoundForThisReceivedBatch = true
							break
						}
					}
					if !matchFoundForThisReceivedBatch {
						t.Errorf("Received an unexpected batch or a duplicate batch: %v", receivedBatch)
					}
				}
				for i, found := range foundMatch {
					if !found {
						t.Errorf("Expected batch was not found: %v", tt.expectedBatches[i])
					}
				}
			}
		})
	}
}

func TestUpdateLabelMap_Batching(t *testing.T) {
	const labelBatchSize = 10 // Must match the constant in builder.go

	tests := []struct {
		name                  string
		existingLabelNames    map[string]struct{}
		allLabelNamesFromProm []string
		expectedBatches       [][]string
		expectLLMCall         bool
	}{
		{
			name:                  "no new labels",
			existingLabelNames:    map[string]struct{}{"label1": {}},
			allLabelNamesFromProm: []string{"label1"},
			expectedBatches:       nil,
			expectLLMCall:         false,
		},
		{
			name:                  "new labels less than batch size",
			existingLabelNames:    map[string]struct{}{},
			allLabelNamesFromProm: []string{"label1", "label2"},
			expectedBatches:       [][]string{{"label1", "label2"}},
			expectLLMCall:         true,
		},
		{
			name:                  "new labels equal to batch size",
			existingLabelNames:    map[string]struct{}{},
			allLabelNamesFromProm: generateLabels(labelBatchSize, 0),
			expectedBatches:       [][]string{generateLabels(labelBatchSize, 0)},
			expectLLMCall:         true,
		},
		{
			name:                  "new labels more than batch size (not multiple)",
			existingLabelNames:    map[string]struct{}{},
			allLabelNamesFromProm: generateLabels(labelBatchSize+2, 0),
			expectedBatches:       [][]string{generateLabels(labelBatchSize, 0), generateLabels(2, labelBatchSize)},
			expectLLMCall:         true,
		},
		{
			name:                  "new labels more than batch size (multiple)",
			existingLabelNames:    map[string]struct{}{},
			allLabelNamesFromProm: generateLabels(labelBatchSize*2, 0),
			expectedBatches:       [][]string{generateLabels(labelBatchSize, 0), generateLabels(labelBatchSize, labelBatchSize)},
			expectLLMCall:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &MockLLMClient_BuilderTest{}
			mockQueryEngine := &MockQueryEngine_BuilderTest{
				AllLabelsFunc: func() ([]string, error) { return tt.allLabelNamesFromProm, nil },
			}
			mockLoaderSaver := &MockInfoLoaderSaver_BuilderTest{}

			is, err := info_structure.NewInfoBuilder(mockQueryEngine, mockLLM, mockLoaderSaver)
			if err != nil {
				t.Fatalf("NewInfoBuilder returned an unexpected error: %v", err)
			}
			if is == nil {
				t.Fatalf("NewInfoBuilder returned a nil InfoStructure instance")
			}

			// Manually initialize maps as BuildInformationStructure would do
			// No need to call LoadInfoStructure again if already done for the same 'is' instance
			// but for isolated test functions, this is fine. If tests were methods on a suite, setup could be shared.
			metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap, loadErr := mockLoaderSaver.LoadInfoStructure()
			if loadErr != nil {
				t.Fatalf("mockLoaderSaver.LoadInfoStructure() returned an error: %v", loadErr)
			}
			is.MetricMap = &metricMap // Required by UpdateLabelMap if it shared logic or touched MetricMap
			is.LabelMap = &labelMap
			is.MetricLabelMap = &metricLabelMap
			is.LabelValueMap = &labelValueMap
			is.NlpToMetricMap = &nlpToMetricMap

			if is.LabelMap == nil {
				t.Fatalf("is.LabelMap is nil after manual initialization")
			}

			is.LabelMap.AllNames = tt.existingLabelNames
			// is.LabelMap.Map is initialized by the mockLoaderSaver.LoadInfoStructure

			err = is.UpdateLabelMap(tt.allLabelNamesFromProm)
			if err != nil {
				t.Fatalf("UpdateLabelMap returned an unexpected error: %v", err)
			}

			if !tt.expectLLMCall {
				if len(mockLLM.ReceivedLabelBatches) != 0 {
					t.Errorf("expected GetLabelSynonyms not to be called, but it was called with %d batches", len(mockLLM.ReceivedLabelBatches))
				}
				return
			}

			if !reflect.DeepEqual(mockLLM.ReceivedLabelBatches, tt.expectedBatches) {
				t.Errorf("GetLabelSynonyms called with incorrect batches.\nExpected: %v\nGot:      %v", tt.expectedBatches, mockLLM.ReceivedLabelBatches)
			}
		})
	}
}

// --- Test Helpers ---

func generateMetrics(count, offset int, prefixOptions ...string) []string {
	prefix := "metric"
	if len(prefixOptions) > 0 && prefixOptions[0] != "" {
		prefix = prefixOptions[0]
	}
	metrics := make([]string, count)
	for i := 0; i < count; i++ {
		metrics[i] = fmt.Sprintf("%s%d", prefix, i+offset)
	}
	return metrics
}

func generateMetricDescs(count, offset int, prefixOptions ...string) map[string]string {
	prefix := "metric"
	if len(prefixOptions) > 0 {
		prefix = prefixOptions[0]
	}
	descs := make(map[string]string)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%s%d", prefix, i+offset)
		descs[key] = fmt.Sprintf("description for %s%d", prefix, i+offset)
	}
	return descs
}

func generateLabels(count, offset int) []string {
	labels := make([]string, count)
	for i := 0; i < count; i++ {
		labels[i] = fmt.Sprintf("label%d", i+offset)
	}
	return labels
}

func mergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// Expose internal methods for testing - this would ideally not be needed if
// info_structure.BuildInformationStructure() was more easily testable in units,
// or if these were public utility methods.
// The methods UpdateMetricMap and UpdateLabelMap are now directly exported from builder.go for testing.
