package info_structure

import (
	"fmt"
	"strings"

	"github.com/prashantgupta17/nlpromql/openai"
	"github.com/prashantgupta17/nlpromql/prometheus"
)

// BuildInformationStructure builds or updates the information structure from Prometheus data.
func BuildInformationStructure(promClient *prometheus.PrometheusConnect, openaiClient *openai.OpenAIClient) (MetricMap, LabelMap, MetricLabelMap, LabelValueMap, NlpToMetricMap, error) {
	// Load existing information structure (if it exists)
	metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap, err := LoadInformationStructure()
	if err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}
	fmt.Println("Metric Map:", len(metricMap.AllNames))

	// Fetch all metric names from Prometheus
	allMetricNames, err := promClient.AllMetrics()
	if err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, fmt.Errorf("error fetching all metric names: %v", err)
	}

	// Update metricMap and get new metric synonyms
	err = updateMetricMap(openaiClient, &metricMap, allMetricNames)
	if err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, fmt.Errorf("error updating metric map: %v", err)
	}

	// Fetch all label names from Prometheus
	allLabelNames, err := promClient.AllLabels()
	if err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, fmt.Errorf("error fetching all metric names: %v", err)
	}

	// Update labelMap and get new label synonyms
	err = updateLabelMap(openaiClient, &labelMap, allLabelNames)
	if err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, fmt.Errorf("error updating label map: %v", err)
	}

	// Batch query Prometheus for metric and label details
	err = updateMetricLabelMapAndLabelValueMap(promClient, metricLabelMap, labelValueMap, allMetricNames)
	if err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, fmt.Errorf("error updating metric-label and label-value maps: %v", err)
	}

	// Save the updated information structure
	if err := SaveInfoStructure(metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, fmt.Errorf("error saving information structure: %v", err)
	}

	return metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap, nil
}

// updateMetricMap updates the metricMap with new metric names and their synonyms.
func updateMetricMap(openaiClient *openai.OpenAIClient, metricMap *MetricMap, allMetricNames []string) error {
	newMetricNames := make([]string, 0) // Using a slice for newMetricNames
	for _, metric := range allMetricNames {
		for _, existingMetric := range metricMap.AllNames {
			if existingMetric == metric {
				continue
			}
		}
	}
	fmt.Println("New metrics:")
	fmt.Println(newMetricNames)

	// Get metric synonyms (only for new metrics)
	if len(newMetricNames) > 0 {
		newMetricSynonyms, err := openaiClient.GetMetricSynonyms(newMetricNames)
		if err != nil {
			return fmt.Errorf("error getting metric synonyms: %w", err)
		}

		// Populate metric_map (only for new metrics)
		for metric, synonyms := range newMetricSynonyms {
			for _, token := range append([]string{strings.ToLower(metric)}, synonyms...) {
				metricMap.Map[token] = append(metricMap.Map[token], metric)
				metricMap.AllNames = append(metricMap.AllNames, metric)
			}
		}
	}
	return nil
}

// updateLabelMap updates the labelMap with new label names and their synonyms.
func updateLabelMap(openaiClient *openai.OpenAIClient, labelMap *LabelMap, allLabelNames []string) error {
	newLabelNames := make([]string, 0)    // Using a slice for newLabelNames
	for _, label := range allLabelNames { // Assuming you have getLabels() function in prometheus package
		for _, existingLabel := range labelMap.AllNames {
			if existingLabel == label {
				continue
			}
		}
	}
	fmt.Println("New labels:")
	fmt.Println(newLabelNames)

	// Get label synonyms (only for new labels)
	if len(newLabelNames) > 0 {
		newLabelSynonyms, err := openaiClient.GetLabelSynonyms(newLabelNames)
		if err != nil {
			return fmt.Errorf("error getting label synonyms: %w", err)
		}

		// Populate label_map (only for new labels)
		for label, synonyms := range newLabelSynonyms {
			for _, token := range append([]string{strings.ToLower(label)}, synonyms...) {
				labelMap.Map[token] = append(labelMap.Map[token], label)
				labelMap.AllNames = append(labelMap.AllNames, label)
			}
		}
	}
	return nil
}

// updateMetricLabelMapAndLabelValueMap updates the metricLabelMap and labelValueMap from Prometheus data.
func updateMetricLabelMapAndLabelValueMap(promClient *prometheus.PrometheusConnect, metricLabelMap MetricLabelMap, labelValueMap LabelValueMap, allMetricNames []string) error {
	metricsToQuery := make([]string, 0) // Use a slice instead of a list
	for _, metric := range allMetricNames {
		if _, exists := metricLabelMap[metric]; !exists {
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
		result, err := promClient.CustomQuery(query)
		if err != nil {
			return fmt.Errorf("error executing PromQL query: %v", err)
		}

		for _, item := range result {
			metricName := item.Metric["__name__"]
			if _, exists := metricLabelMap[metricName]; !exists {
				metricLabelMap[metricName] = make(map[string]map[string]struct{})
			}

			for label, value := range item.Metric {
				if label != "__name__" {
					if _, exists := metricLabelMap[metricName][label]; !exists {
						metricLabelMap[metricName][label] = make(map[string]struct{})
					}
					metricLabelMap[metricName][label][value] = struct{}{}

					if _, exists := labelValueMap[label]; !exists {
						labelValueMap[label] = make(map[string]struct{})
					}
					labelValueMap[label][value] = struct{}{}
				}
			}
		}
	}
	return nil
}
