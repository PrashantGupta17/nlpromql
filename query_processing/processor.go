package query_processing

import (
	"encoding/json"
	"fmt"

	"github.com/prashantgupta17/nlpromql/info_structure" // Replace with the actual path
	"github.com/prashantgupta17/nlpromql/openai"         // Replace with the actual path
)

// processUserQuery3 processes the user query to extract potential metrics and labels.
func processUserQuery3(client *openai.OpenAIClient, userQuery string) (map[string]interface{}, error) {
	possibleMatches, err := client.ProcessUserQuery(userQuery)
	if err != nil {
		return nil, err
	}
	return possibleMatches, nil
}

// processUserQuery processes the user query using the information structure.
func ProcessUserQuery(client *openai.OpenAIClient, userQuery string, metricMap info_structure.MetricMap, labelMap info_structure.LabelMap,
	metricLabelMap info_structure.MetricLabelMap, labelValueMap info_structure.LabelValueMap,
	nlpToMetricMap info_structure.NlpToMetricMap) (map[string]interface{}, openai.RelevantMetricsMap, openai.RelevantLabelsMap, map[string]interface{}, error) {

	possibleMatches, err := processUserQuery3(client, userQuery)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	fmt.Println("Possible Matches:", possibleMatches)
	relevantMetrics := make(openai.RelevantMetricsMap)
	relevantLabels := make(openai.RelevantLabelsMap)
	relevantHistory := make(map[string]interface{})

	// Process possible metric names (logic similar to your Python code)
	for _, metricToken := range possibleMatches["possible_metric_names"].([]interface{}) {
		metricTokenStr := metricToken.(string)
		if metricNames, exists := metricMap.Map[metricTokenStr]; exists {
			for metricName := range metricNames {
				// Extract the MetricInfo struct from the map
				metricInfo, exists := relevantMetrics[metricName]
				if !exists {
					// If it doesn't exist, initialize it
					metricInfo = openai.RelevantMetricInfo{
						MatchScore: 1, // Start with a score of 1
						Labels:     make(map[string]openai.RelevantLabelInfo),
					}
					relevantMetrics[metricName] = metricInfo
				} else {
					// If it exists, increment the MatchScore
					metricInfo.MatchScore++
					relevantMetrics[metricName] = metricInfo
					continue
				}

				// Process possible label names for this metric
				for _, labelToken := range possibleMatches["possible_label_names"].([]interface{}) {
					labelTokenStr := labelToken.(string)
					if labelNames, exists := labelMap.Map[labelTokenStr]; exists {
						for labelName := range labelNames {
							labelInfo, exists := relevantMetrics[metricName].Labels[labelName]
							if !exists {
								// If it doesn't exist, initialize it
								labelInfo = openai.RelevantLabelInfo{
									MatchScore: 1, // Start with a score of 1
									Values:     make(map[string]interface{}),
								}
							} else {
								// If it exists, increment the MatchScore
								labelInfo.MatchScore++
								relevantMetrics[metricName].Labels[labelName] = labelInfo
								continue
							}
							if labelValues, exists := metricLabelMap[metricName].Labels[labelName]; exists {
								labelValuesSlice := make([]string, 0, len(labelValues.Values))
								for val := range labelValues.Values {
									labelValuesSlice = append(labelValuesSlice, val)
								}
								if len(labelValuesSlice) > 5 {
									labelValuesSlice = labelValuesSlice[:5]
								}
								for _, value := range labelValuesSlice {
									labelInfo.Values[value] = nil
								}
							}
							relevantMetrics[metricName].Labels[labelName] = labelInfo
						}
					}
				}
			}
		}
	}

	// Process possible label names (logic similar to your Python code)
	for _, labelToken := range possibleMatches["possible_label_names"].([]interface{}) {
		labelTokenStr := labelToken.(string)
		for actualLabelName := range labelMap.Map[labelTokenStr] {
			relevantLabelInfo, exists := relevantLabels[actualLabelName]
			if !exists {
				// Initialize the LabelInfo struct
				relevantLabelInfo = openai.RelevantLabelInfo{
					MatchScore: 1, // Start with a score of 1
					Values:     make(map[string]interface{}),
				}
			} else {
				// If it exists, increment the MatchScore
				relevantLabelInfo.MatchScore++
				relevantLabels[actualLabelName] = relevantLabelInfo
				continue
			}

			if labelInfo, exists := labelValueMap[actualLabelName]; exists {
				// Ensure values is a slice of strings before slicing
				valuesSlice := make([]string, 0, len(labelInfo.Values))
				for val := range labelInfo.Values {
					valuesSlice = append(valuesSlice, val)
				}
				if len(valuesSlice) > 5 {
					valuesSlice = valuesSlice[:5]
				}
				for _, value := range valuesSlice {
					relevantLabelInfo.Values[value] = nil
				}
			}
			relevantLabels[actualLabelName] = relevantLabelInfo
		}
	}

	// Retrieve relevant info from nlp_to_metric_map (without score updates)
	for key, value := range nlpToMetricMap {
		keyParts := make([]string, 0)
		if err := json.Unmarshal([]byte(key), &keyParts); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("error unmarshaling nlpToMetricMap key: %v", err)
		}
		if containsAny(possibleMatches["possible_metric_names"].([]interface{}), keyParts[0]) &&
			containsAny(possibleMatches["possible_label_names"].([]interface{}), keyParts[1]) {
			// Convert value (interface{}) to map[string]interface{}
			var valueMap map[string]interface{}
			if err := json.Unmarshal([]byte(value), &valueMap); err != nil {
				return nil, nil, nil, nil, fmt.Errorf("error unmarshaling nlpToMetricMap value: %v", err)
			}
			for k, v := range valueMap {
				relevantHistory[k] = v
			}
		}
	}
	fmt.Println("Relevant Metrics:", relevantMetrics)
	fmt.Println("Relevant Labels:", relevantLabels)
	fmt.Println("Relevant History:", relevantHistory)
	return possibleMatches, relevantMetrics, relevantLabels, relevantHistory, nil
}

// Helper function to check if a slice of interface{} contains any of the elements in a given string
func containsAny(slice []interface{}, str string) bool {
	for _, item := range slice {
		if itemStr, ok := item.(string); ok && itemStr == str {
			return true
		}
	}
	return false
}
