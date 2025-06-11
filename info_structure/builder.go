package info_structure

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prashantgupta17/nlpromql/llm"
)

// NewInfoBuilder creates a new InfoBuilder struct.
func NewInfoBuilder(queryEngine QueryEngine, llmClient llm.LLMClient,
	loaderSaver InfoLoaderSaver) (*InfoStructure, error) {
	if loaderSaver == nil {
		defaultLoaderSaver, err := getDefaultInfoLoaderSaver()
		if err != nil {
			return nil, fmt.Errorf("error getting default info loader saver: %v", err)
		}
		loaderSaver = defaultLoaderSaver
	}
	return &InfoStructure{
		QueryEngine:     queryEngine,
		llmClient:       llmClient,
		InfoLoaderSaver: loaderSaver,
	}, nil
}

func getDefaultInfoLoaderSaver() (InfoLoaderSaver, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current working directory: %v", err)
	}
	dir := filepath.Join(pwd, "info")
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating info directory: %v", err)
	}
	return &InfoStructureManager{
		PathToMetricMap:      filepath.Join(dir, "metric_map.json"),
		PathToLabelMap:       filepath.Join(dir, "label_map.json"),
		PathToMetricLabelMap: filepath.Join(dir, "metric_label_map.json"),
		PathToLabelValueMap:  filepath.Join(dir, "label_value_map.json"),
		PathToNlpToMetricMap: filepath.Join(dir, "nlp_to_metric_map.json"),
	}, nil
}

// BuildInformationStructure builds or updates the information structure from Prometheus data.
func (is *InfoStructure) BuildInformationStructure() error {
	is.buildStatusLock.Lock()
	is.buildStatus = BuildStatus{
		IsRunning:     true,
		StartTime:     time.Now(),
		ProgressStage: "Initializing",
	}
	is.buildStatusLock.Unlock()

	defer func() {
		is.buildStatusLock.Lock()
		is.buildStatus.IsRunning = false
		is.buildStatus.EndTime = time.Now()
		is.buildStatusLock.Unlock()
	}()

	is.updateProgressStage("Loading info structure")
	// Load existing information structure (if it exists)
	metricMap, labelMap, metricLabelMap, labelValueMap,
		nlpToMetricMap, err := is.InfoLoaderSaver.LoadInfoStructure()
	if err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error loading info structure: %v", err)
	}
	is.MetricMap = &metricMap
	is.LabelMap = &labelMap
	is.MetricLabelMap = &metricLabelMap
	is.LabelValueMap = &labelValueMap
	is.NlpToMetricMap = &nlpToMetricMap

	// Fetch all metric names from Prometheus
	is.updateProgressStage("Fetching existing metric names")
	allMetricNames, err := is.QueryEngine.AllMetrics()
	if err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error fetching all metric names: %v", err)
	}

	// Fetch all metric descriptions from Prometheus
	is.updateProgressStage("Fetching existing metric descriptions")
	allMetricDescriptions, err := is.QueryEngine.AllMetadata()
	if err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error fetching all metric descriptions: %v", err)
	}

	// Update metricMap and get new metric synonyms
	is.updateProgressStage("Updating existing metric map")
	err = is.updateMetricMap(allMetricNames, allMetricDescriptions)
	if err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error updating metric map: %v", err)
	}

	// Fetch all label names from Prometheus
	is.updateProgressStage("Fetching existing label names")
	allLabelNames, err := is.QueryEngine.AllLabels()
	if err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error fetching all metric names: %v", err)
	}

	// Update labelMap and get new label synonyms
	is.updateProgressStage("Fetching existing label map")
	err = is.updateLabelMap(allLabelNames)
	if err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error updating label map: %v", err)
	}

	// Batch query Prometheus for metric and label details
	is.updateProgressStage("Updating existing metric label combinations map")
	err = is.updateMetricLabelMapAndLabelValueMap(allMetricNames)
	if err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error updating metric-label and label-value maps: %v", err)
	}

	// Save the updated information structure
	is.updateProgressStage("Saving new info structure")
	if err := is.InfoLoaderSaver.SaveInfoStructure(
		*is.MetricMap, *is.LabelMap, *is.MetricLabelMap, *is.LabelValueMap, *is.NlpToMetricMap); err != nil {
		is.updateErrorStatus(err)
		return fmt.Errorf("error saving information structure: %v", err)
	}

	return nil
}

func (is *InfoStructure) updateProgressStage(stage string) {
	log.Printf("%s\n", stage)
	is.buildStatusLock.Lock()
	is.buildStatus.ProgressStage = stage
	is.buildStatusLock.Unlock()
}

func (is *InfoStructure) updateErrorStatus(err error) {
	is.buildStatusLock.Lock()
	is.buildStatus.Error = err
	is.buildStatus.IsRunning = false
	is.buildStatus.EndTime = time.Now()
	is.buildStatusLock.Unlock()
}

// Status Checking Methods
func (is *InfoStructure) GetBuildStatus() BuildStatus {
	is.buildStatusLock.RLock()
	defer is.buildStatusLock.RUnlock()
	return is.buildStatus
}

func (is *InfoStructure) IsBuilding() bool {
	is.buildStatusLock.RLock()
	defer is.buildStatusLock.RUnlock()
	return is.buildStatus.IsRunning
}

// UpdateMetricMap updates the metricMap with new metric names and their synonyms.
// Exported for testing purposes.
func (is *InfoStructure) UpdateMetricMap(allMetricNames []string,
	allMetricDescriptions map[string]string) error {
	newMetricNames := make([]string, 0) // Using a slice for newMetricNames
	// Determine new metric names that are not already in the MetricMap
	for _, metric := range allMetricNames {
		found := false
		for existingMetric, _ := range is.MetricMap.AllNames {
			if existingMetric == metric {
				found = true
				break
			}
		}
		if !found {
			newMetricNames = append(newMetricNames, metric)
		}
	}

	if len(newMetricNames) == 0 {
		return nil // No new metrics to process
	}

	// Prepare map of new metrics to their descriptions
	metricsToQueryForSynonyms := make(map[string]string)
	for _, metricName := range newMetricNames {
		if desc, exists := allMetricDescriptions[metricName]; exists {
			metricsToQueryForSynonyms[metricName] = desc
		} else {
			metricsToQueryForSynonyms[metricName] = "" // Use empty string if no description
		}
	}
	fmt.Printf("Found %d new metrics to get synonyms for\n", len(metricsToQueryForSynonyms))

	// Batch preparation for GetMetricSynonyms
	const metricBatchSize = 10
	metricBatches := []map[string]string{}
	currentBatch := make(map[string]string)
	countInCurrentBatch := 0

	for metricName, description := range metricsToQueryForSynonyms {
		currentBatch[metricName] = description
		countInCurrentBatch++
		if countInCurrentBatch >= metricBatchSize {
			metricBatches = append(metricBatches, currentBatch)
			currentBatch = make(map[string]string)
			countInCurrentBatch = 0
		}
	}
	if countInCurrentBatch > 0 {
		metricBatches = append(metricBatches, currentBatch)
	}

	if len(metricBatches) > 0 {
		newMetricSynonyms, err := is.llmClient.GetMetricSynonyms(metricBatches)
		if err != nil {
			return fmt.Errorf("error getting metric synonyms: %w", err)
		}
		if is.MetricMap.Map == nil {
			is.MetricMap.Map = make(map[string]map[string]struct{})
		}
		if is.MetricMap.AllNames == nil {
			is.MetricMap.AllNames = make(map[string]struct{})
		}
		// Populate metric_map (only for new metrics)
		for metric, synonyms := range newMetricSynonyms {
			for _, token := range append([]string{strings.ToLower(metric)}, synonyms...) {
				if _, ok := is.MetricMap.Map[token]; !ok {
					is.MetricMap.Map[token] = make(map[string]struct{})
				}
				is.MetricMap.Map[token][metric] = struct{}{}
				is.MetricMap.AllNames[metric] = struct{}{}
			}
		}
	}
	return nil
}

// UpdateLabelMap updates the labelMap with new label names and their synonyms.
// Exported for testing purposes.
func (is *InfoStructure) UpdateLabelMap(allLabelNames []string) error {
	newLabelNames := make([]string, 0) // Using a slice for newLabelNames
	// Determine new label names that are not already in the LabelMap
	for _, label := range allLabelNames {
		found := false
		for existingLabel := range is.LabelMap.AllNames {
			if existingLabel == label {
				found = true
				break
			}
		}
		if !found {
			newLabelNames = append(newLabelNames, label)
		}
	}

	if len(newLabelNames) == 0 {
		return nil // No new labels to process
	}
	fmt.Printf("Found %d new labels to get synonyms for\n", len(newLabelNames))

	// Batch preparation for GetLabelSynonyms
	const labelBatchSize = 10
	labelBatches := [][]string{}
	currentBatch := []string{}

	for _, labelName := range newLabelNames {
		currentBatch = append(currentBatch, labelName)
		if len(currentBatch) >= labelBatchSize {
			labelBatches = append(labelBatches, currentBatch)
			currentBatch = []string{}
		}
	}
	if len(currentBatch) > 0 {
		labelBatches = append(labelBatches, currentBatch)
	}

	if len(labelBatches) > 0 {
		newLabelSynonyms, err := is.llmClient.GetLabelSynonyms(labelBatches)
		if err != nil {
			return fmt.Errorf("error getting label synonyms: %w", err)
		}
		if is.LabelMap.Map == nil {
			is.LabelMap.Map = make(map[string]map[string]struct{})
		}
		if is.LabelMap.AllNames == nil {
			is.LabelMap.AllNames = make(map[string]struct{})
		}
		// Populate label_map (only for new labels)
		for label, synonyms := range newLabelSynonyms {
			for _, token := range append([]string{strings.ToLower(label)}, synonyms...) {
				if is.LabelMap.Map[token] == nil {
					is.LabelMap.Map[token] = make(map[string]struct{})
				}
				is.LabelMap.Map[token][label] = struct{}{}
				is.LabelMap.AllNames[label] = struct{}{}
			}
		}
	}
	return nil
}

// updateMetricLabelMapAndLabelValueMap updates the metricLabelMap and labelValueMap from Prometheus data.
func (is *InfoStructure) updateMetricLabelMapAndLabelValueMap(allMetricNames []string) error {
	metricsToQuery := make([]string, 0) // Use a slice instead of a list
	for _, metric := range allMetricNames {
		if _, exists := (*is.MetricLabelMap)[metric]; !exists {
			metricsToQuery = append(metricsToQuery, metric)
		}
	}

	batchSize := 100
	for i := 0; i < len(metricsToQuery); i += batchSize {
		if (i + batchSize) > len(metricsToQuery) {
			batchSize = len(metricsToQuery) - i
		}
		metricBatch := metricsToQuery[i : i+batchSize]
		metricNameRegex := ""
		if len(metricBatch) > 0 {
			nonEmptyMetrics := make([]string, 0)
			for _, metric := range metricBatch {
				if metric != "" {
					nonEmptyMetrics = append(nonEmptyMetrics, metric)
				}
			}
			metricNameRegex = strings.Join(nonEmptyMetrics, "|")
		}

		query := fmt.Sprintf("{__name__=~\"%s\", __aggregation__!=\"None\"}", metricNameRegex) // Use double quotes around regex
		result, err := is.QueryEngine.CustomQuery(query)
		if err != nil {
			return fmt.Errorf("error executing PromQL query: %v", err)
		}

		for _, item := range result {
			metricName := item.Metric["__name__"]
			if _, exists := (*is.MetricLabelMap)[metricName]; !exists {
				(*is.MetricLabelMap)[metricName] = MetricInfo{
					Labels: make(map[string]LabelInfo),
				}
			}

			for label, value := range item.Metric {
				if label != "__name__" {
					if _, exists := (*is.MetricLabelMap)[metricName].Labels[label]; !exists {
						(*is.MetricLabelMap)[metricName].Labels[label] = LabelInfo{
							Values: make(map[string]struct{}),
						}
					}
					(*is.MetricLabelMap)[metricName].Labels[label].Values[value] = struct{}{}

					if _, exists := (*is.LabelValueMap)[label]; !exists {
						(*is.LabelValueMap)[label] = LabelInfo{
							Values: make(map[string]struct{}),
						}
					}
					(*is.LabelValueMap)[label].Values[value] = struct{}{}
				}
			}
		}
	}
	return nil
}
