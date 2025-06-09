package query_processing

import (
	"encoding/json"
	"fmt"

	"github.com/prashantgupta17/nlpromql/info_structure"
	"github.com/prashantgupta17/nlpromql/llm"
)

// processUserQuery3 helper function to call LLM for initial query processing.
func processUserQuery3(client llm.LLMClient, userQuery string) (map[string]interface{}, error) {
	possibleMatches, err := client.ProcessUserQuery(userQuery)
	if err != nil {
		return nil, err
	}
	return possibleMatches, nil
}

// ProcessUserQuery processes a user's natural language query to extract structured information
// relevant for forming PromQL queries. It uses an LLM to identify potential metrics, labels,
// and values, then cross-references these with known information from Prometheus
// (metricMap, labelMap, etc.) to build contextually relevant maps.
func ProcessUserQuery(client llm.LLMClient, userQuery string, metricMap info_structure.MetricMap, labelMap info_structure.LabelMap,
	metricLabelMap info_structure.MetricLabelMap, labelValueMap info_structure.LabelValueMap,
	nlpToMetricMap info_structure.NlpToMetricMap) (map[string]interface{}, llm.RelevantMetricsMap, llm.RelevantLabelsMap, map[string]interface{}, error) {

	possibleMatches, err := processUserQuery3(client, userQuery)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("error processing user query via LLM: %w", err)
	}
	// fmt.Println("Possible Matches from LLM:", possibleMatches) // Debug print

	relevantMetrics := make(llm.RelevantMetricsMap)
	relevantLabels := make(llm.RelevantLabelsMap)
	relevantHistory := make(map[string]interface{})

	// Define a helper to extract up to 5 string values from a map[string]struct{}
	getSampleValues := func(valueSet map[string]struct{}) []string {
		values := make([]string, 0, len(valueSet))
		count := 0
		for val := range valueSet {
			if count >= 5 {
				break
			}
			values = append(values, val)
			count++
		}
		return values
	}

	// Process possible metric names to populate relevantMetrics.
	// relevantMetrics structure: map[metricName]map[labelName]LabelContextDetail
	if metricTokens, ok := possibleMatches["possible_metric_names"].([]interface{}); ok {
		for _, metricToken := range metricTokens {
			metricTokenStr, isString := metricToken.(string)
			if !isString {
				continue
			}
			// Find actual metric names from the token (synonym)
			if actualMetricNames, exists := metricMap.Map[metricTokenStr]; exists {
				for metricName := range actualMetricNames {
					if _, metricEntryExists := relevantMetrics[metricName]; !metricEntryExists {
						relevantMetrics[metricName] = make(map[string]llm.LabelContextDetail)
					}

					// Now, for this metricName, find its relevant labels and their values
					// Iterate through possible label names identified by the LLM
					if labelTokens, ok := possibleMatches["possible_label_names"].([]interface{}); ok {
						for _, labelToken := range labelTokens {
							labelTokenStr, isStringLabelToken := labelToken.(string)
							if !isStringLabelToken {
								continue
							}
							// Find actual label names from the token
							if actualLabelNames, labelTokenExists := labelMap.Map[labelTokenStr]; labelTokenExists {
								for actualLabelName := range actualLabelNames {
									// Check if this actualLabelName is a valid label for metricName
									if metricInfoFromMap, metricInLabelMapExists := metricLabelMap[metricName]; metricInLabelMapExists {
										if labelDetailForMetric, labelValidForMetric := metricInfoFromMap.Labels[actualLabelName]; labelValidForMetric {
											// We found a valid label for this metric. Populate its context.
											if _, labelContextExists := relevantMetrics[metricName][actualLabelName]; !labelContextExists {
												relevantMetrics[metricName][actualLabelName] = llm.LabelContextDetail{
													MatchScore: 1.0, // Placeholder score
													Values:     getSampleValues(labelDetailForMetric.Values),
												}
											} else {
												// If label context already exists, we could increment score or merge values.
												// For now, simple approach: assume first encountered is fine, or update score.
												temp := relevantMetrics[metricName][actualLabelName]
												temp.MatchScore += 0.5 // Increment score if mentioned again
												relevantMetrics[metricName][actualLabelName] = temp
											}
										}
									}
								}
							}
						}
					}
					// Also consider labels directly from possible_label_values if they apply to this metric
                    if labelValueTokens, ok := possibleMatches["possible_label_values"].([]interface{}); ok {
                        for _, lvToken := range labelValueTokens {
                            lvTokenStr, isStringLVToken := lvToken.(string)
                            if !isStringLVToken {
                                continue
                            }
                            // This part is tricky: lvTokenStr is a value. We need to find which label it belongs to
                            // and if that label is relevant for the current metricName.
                            // This might require iterating through all labels of metricName and checking if lvTokenStr is a value for any of them.
                            if metricInfoFromMap, metricInLabelMapExists := metricLabelMap[metricName]; metricInLabelMapExists {
                                for labelNameForMetric, labelDetailForMetric := range metricInfoFromMap.Labels {
                                    if _, valueExistsInLabel := labelDetailForMetric.Values[lvTokenStr]; valueExistsInLabel {
                                        if _, labelContextExists := relevantMetrics[metricName][labelNameForMetric]; !labelContextExists {
                                             relevantMetrics[metricName][labelNameForMetric] = llm.LabelContextDetail{
                                                MatchScore: 1.0, // Placeholder for value match
                                                Values:     []string{lvTokenStr}, // Specific value matched
                                            }
                                        } else {
                                            // Append value if not present, update score
                                            temp := relevantMetrics[metricName][labelNameForMetric]
                                            valueFound := false
                                            for _, v := range temp.Values { if v == lvTokenStr { valueFound = true; break } }
                                            if !valueFound { temp.Values = append(temp.Values, lvTokenStr) }
                                            temp.MatchScore += 0.2 // Increment score for value match
                                            relevantMetrics[metricName][labelNameForMetric] = temp
                                        }
                                    }
                                }
                            }
                        }
                    }
				}
			}
		}
	}

	// Process possible label names to populate relevantLabels.
	// relevantLabels structure: map[labelName]LabelContextDetail
	if labelTokens, ok := possibleMatches["possible_label_names"].([]interface{}); ok {
		for _, labelToken := range labelTokens {
			labelTokenStr, isString := labelToken.(string)
			if !isString {
				continue
			}
			if actualLabelNames, exists := labelMap.Map[labelTokenStr]; exists {
				for actualLabelName := range actualLabelNames {
					if _, labelEntryExists := relevantLabels[actualLabelName]; !labelEntryExists {
						// Get sample values for this general label from labelValueMap
						sampleValues := []string{}
						if labelInfoFromMap, labelInValueMapExists := labelValueMap[actualLabelName]; labelInValueMapExists {
							sampleValues = getSampleValues(labelInfoFromMap.Values)
						}
						relevantLabels[actualLabelName] = llm.LabelContextDetail{
							MatchScore: 1.0, // Placeholder score
							Values:     sampleValues,
						}
					} else {
						temp := relevantLabels[actualLabelName]
						temp.MatchScore += 0.5
						relevantLabels[actualLabelName] = temp
					}
				}
			}
		}
	}
    // Add label values from "possible_label_values" to relevantLabels
    if labelValueTokens, ok := possibleMatches["possible_label_values"].([]interface{}); ok {
        for _, lvToken := range labelValueTokens {
            lvTokenStr, isStringLVToken := lvToken.(string)
            if !isStringLVToken {
                continue
            }
            // Find which label this value might belong to by checking labelValueMap
            for generalLabelName, generalLabelInfo := range labelValueMap {
                if _, valueExistsInGeneralLabel := generalLabelInfo.Values[lvTokenStr]; valueExistsInGeneralLabel {
                    if entry, exists := relevantLabels[generalLabelName]; !exists {
                        relevantLabels[generalLabelName] = llm.LabelContextDetail{
                            MatchScore: 1.0,
                            Values:     []string{lvTokenStr},
                        }
                    } else {
                        valueFound := false
                        for _, v := range entry.Values { if v == lvTokenStr { valueFound = true; break } }
                        if !valueFound { entry.Values = append(entry.Values, lvTokenStr) }
                        entry.MatchScore += 0.2
                        relevantLabels[generalLabelName] = entry
                    }
                }
            }
        }
    }


	// Retrieve relevant info from nlp_to_metric_map (logic remains similar)
	// This part populates `relevantHistory` which is map[string]interface{} and doesn't need structural change for its value.
	if possibleMetricNames, pmnOK := possibleMatches["possible_metric_names"].([]interface{}); pmnOK {
		if possibleLabelNames, plnOK := possibleMatches["possible_label_names"].([]interface{}); plnOK {
			for key, value := range nlpToMetricMap {
				keyParts := make([]string, 0)
				if err := json.Unmarshal([]byte(key), &keyParts); err != nil {
					return nil, nil, nil, nil, fmt.Errorf("error unmarshaling nlpToMetricMap key: %v", err)
				}
				if len(keyParts) == 2 && containsAny(possibleMetricNames, keyParts[0]) &&
					containsAny(possibleLabelNames, keyParts[1]) {
					var valueMap map[string]interface{}
					if err := json.Unmarshal([]byte(value), &valueMap); err != nil {
						return nil, nil, nil, nil, fmt.Errorf("error unmarshaling nlpToMetricMap value: %v", err)
					}
					for k, v := range valueMap {
						relevantHistory[k] = v
					}
				}
			}
		}
	}

	// Debug prints for the final constructed relevance maps. Can be noisy.
	// fmt.Println("Final Relevant Metrics:", relevantMetrics)
	// fmt.Println("Final Relevant Labels:", relevantLabels)
	// fmt.Println("Final Relevant History:", relevantHistory)
	return possibleMatches, relevantMetrics, relevantLabels, relevantHistory, nil
}

// containsAny checks if a slice of interface{} (expected to be strings) contains a specific string.
func containsAny(slice []interface{}, str string) bool {
	for _, item := range slice {
		if itemStr, ok := item.(string); ok && itemStr == str {
			return true
		}
	}
	return false
}
