package info_structure

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/prashantgupta17/nlpromql/config"
)

// LoadInformationStructure loads all information structures from JSON files.
func LoadInformationStructure() (MetricMap, LabelMap, MetricLabelMap, LabelValueMap, NlpToMetricMap, error) {
	var metricMap MetricMap
	if err := loadMapFromFile(config.MetricMapFile, &metricMap); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}

	var labelMap LabelMap
	if err := loadMapFromFile(config.LabelMapFile, &labelMap); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}

	var metricLabelMapJSON MapForJSON
	if err := loadMapFromFile(config.MetricLabelMapFile, &metricLabelMapJSON); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}
	metricLabelMap := convertJSONToMetricLabelMap(metricLabelMapJSON)

	var labelValueMapJSON MapForJSON
	if err := loadMapFromFile(config.LabelValueMapFile, &labelValueMapJSON); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}
	labelValueMap := convertJSONToLabelValueMap(labelValueMapJSON)

	var nlpToMetricMap NlpToMetricMap
	if err := loadMapFromFile(config.NlpToMetricMapFile, &nlpToMetricMap); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}

	return metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap, nil
}

// loadMapFromFile loads a map from a JSON file.
func loadMapFromFile(filePath string, data interface{}) error {
	fmt.Println("Loading:", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Return nil if file doesn't exist
		}
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(data); err != nil {
		return fmt.Errorf("error decoding JSON: %v", err)
	}

	return nil
}

// convertJSONToMetricLabelMap converts the JSON representation of MetricLabelMap back to the original type.
func convertJSONToMetricLabelMap(data MapForJSON) MetricLabelMap {
	result := make(MetricLabelMap)
	for metric, labelsRaw := range data {
		labels := labelsRaw.(map[string]interface{})
		result[metric] = make(map[string]map[string]struct{})
		for label, valuesRaw := range labels {
			values := valuesRaw.([]interface{})
			result[metric][label] = make(map[string]struct{})
			for _, v := range values {
				value := fmt.Sprintf("%v", v) // Convert interface{} to string
				result[metric][label][value] = struct{}{}
			}
		}
	}
	return result
}

// convertJSONToLabelValueMap converts the JSON representation of LabelValueMap back to the original type.
func convertJSONToLabelValueMap(data MapForJSON) LabelValueMap {
	result := make(LabelValueMap)
	for label, valuesRaw := range data {
		values := valuesRaw.([]interface{})
		result[label] = make(map[string]struct{})
		for _, v := range values {
			value := fmt.Sprintf("%v", v) // Convert interface{} to string
			result[label][value] = struct{}{}
		}
	}
	return result
}
