package info_structure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/prashantgupta17/nlpromql/openai"
)

// NewInfoBuilder creates a new InfoBuilder struct.
func NewInfoBuilder(queryEngine QueryEngine, openaiClient *openai.OpenAIClient,
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
		OpenAIClient:    openaiClient,
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
	// Load existing information structure (if it exists)
	metricMap, labelMap, metricLabelMap, labelValueMap,
		nlpToMetricMap, err := is.InfoLoaderSaver.LoadInfoStructure()
	if err != nil {
		return err
	}
	is.MetricMap = &metricMap
	is.LabelMap = &labelMap
	is.MetricLabelMap = &metricLabelMap
	is.LabelValueMap = &labelValueMap
	is.NlpToMetricMap = &nlpToMetricMap

	// Fetch all metric names from Prometheus
	allMetricNames, err := is.QueryEngine.AllMetrics()
	if err != nil {
		return fmt.Errorf("error fetching all metric names: %v", err)
	}

	// Fetch all metric descriptions from Prometheus
	allMetricDescriptions, err := is.QueryEngine.AllMetadata()
	if err != nil {
		return fmt.Errorf("error fetching all metric descriptions: %v", err)
	}

	// Update metricMap and get new metric synonyms
	err = is.updateMetricMap(allMetricNames, allMetricDescriptions)
	if err != nil {
		return fmt.Errorf("error updating metric map: %v", err)
	}

	// Fetch all label names from Prometheus
	allLabelNames, err := is.QueryEngine.AllLabels()
	if err != nil {
		return fmt.Errorf("error fetching all metric names: %v", err)
	}

	// Update labelMap and get new label synonyms
	err = is.updateLabelMap(allLabelNames)
	if err != nil {
		return fmt.Errorf("error updating label map: %v", err)
	}

	// Batch query Prometheus for metric and label details
	err = is.updateMetricLabelMapAndLabelValueMap(allMetricNames)
	if err != nil {
		return fmt.Errorf("error updating metric-label and label-value maps: %v", err)
	}

	// Save the updated information structure
	if err := is.InfoLoaderSaver.SaveInfoStructure(
		*is.MetricMap, *is.LabelMap, *is.MetricLabelMap, *is.LabelValueMap, *is.NlpToMetricMap); err != nil {
		return fmt.Errorf("error saving information structure: %v", err)
	}

	return nil
}

// updateMetricMap updates the metricMap with new metric names and their synonyms.
func (is *InfoStructure) updateMetricMap(allMetricNames []string,
	allMetricDescriptions map[string]string) error {
	newMetricNames := make([]string, 0) // Using a slice for newMetricNames
	for _, metric := range allMetricNames {
		found := false
		fmt.Println("Metric:", metric)
		for existingMetric, _ := range is.MetricMap.AllNames {
			if existingMetric == metric {
				fmt.Println("Metric:", metric)
				found = true
				break
			}
		}
		if !found {
			newMetricNames = append(newMetricNames, metric)
		}
	}
	fmt.Println("New metrics:")
	fmt.Println(newMetricNames)
	newMetricMap := make(map[string]string)
	for _, metric := range newMetricNames {
		if desc, exists := allMetricDescriptions[metric]; exists {
			newMetricMap[metric] = desc
		} else {
			newMetricMap[metric] = ""
		}
	}

	// Get metric synonyms (only for new metrics)
	if len(newMetricNames) > 0 {
		newMetricSynonyms, err := is.OpenAIClient.GetMetricSynonyms(newMetricMap)
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

// updateLabelMap updates the labelMap with new label names and their synonyms.
func (is *InfoStructure) updateLabelMap(allLabelNames []string) error {
	newLabelNames := make([]string, 0)    // Using a slice for newLabelNames
	for _, label := range allLabelNames { // Assuming you have getLabels() function in prometheus package
		found := false
		fmt.Println("Label:", label)
		for existingLabel, _ := range is.LabelMap.AllNames {
			if existingLabel == label {
				fmt.Println("Label:", label)
				found = true
				break
			}
		}
		if !found {
			newLabelNames = append(newLabelNames, label)
		}
	}
	fmt.Println("New labels:")
	fmt.Println(newLabelNames)

	// Get label synonyms (only for new labels)
	if len(newLabelNames) > 0 {
		newLabelSynonyms, err := is.OpenAIClient.GetLabelSynonyms(newLabelNames)
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
	fmt.Println("New dict:")
	fmt.Println(len(metricsToQuery))

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
