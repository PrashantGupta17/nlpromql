package info_structure

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadInformationStructure loads all information structures from JSON files.
func (im *InfoStructureManager) LoadInfoStructure() (MetricMap, LabelMap,
	MetricLabelMap, LabelValueMap, NlpToMetricMap, error) {
	var metricMapJSON MetricJsonMap
	if err := loadMapFromFile(im.PathToMetricMap, &metricMapJSON); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}
	metricMap := convertJSONToMetricMap(metricMapJSON)

	var labelMapJSON LabelJsonMap
	if err := loadMapFromFile(im.PathToLabelMap, &labelMapJSON); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}
	labelMap := convertJSONToLabelMap(labelMapJSON)

	var metricLabelMapJSON MapForJSON
	if err := loadMapFromFile(im.PathToMetricLabelMap, &metricLabelMapJSON); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}
	metricLabelMap := convertJSONToMetricLabelMap(metricLabelMapJSON)

	var labelValueMapJSON MapForJSON
	if err := loadMapFromFile(im.PathToLabelValueMap, &labelValueMapJSON); err != nil {
		return MetricMap{}, LabelMap{}, nil, nil, nil, err
	}
	labelValueMap := convertJSONToLabelValueMap(labelValueMapJSON)

	var nlpToMetricMap NlpToMetricMap
	if err := loadMapFromFile(im.PathToNlpToMetricMap, &nlpToMetricMap); err != nil {
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

// convertJSONToLabelMap converts the JSON representation of LabelMap back to the original type.
func convertJSONToLabelMap(data LabelJsonMap) LabelMap {
	result := LabelMap{
		Map:      make(map[string]map[string]struct{}),
		AllNames: make(map[string]struct{}),
	}
	for label, names := range data.Map {
		result.Map[label] = make(map[string]struct{})
		for _, name := range names {
			result.Map[label][name] = struct{}{}
		}
	}

	for _, name := range data.AllNames {
		result.AllNames[name] = struct{}{}
	}
	return result
}

// convertJSONToMetricMap converts the JSON representation of MetricMap back to the original type.
func convertJSONToMetricMap(data MetricJsonMap) MetricMap {
	result := MetricMap{
		Map:      make(map[string]map[string]struct{}),
		AllNames: make(map[string]struct{}),
	}
	for synonym, metrics := range data.Map {
		result.Map[synonym] = make(map[string]struct{})
		for _, name := range metrics {
			result.Map[synonym][name] = struct{}{}
		}
	}

	for _, name := range data.AllNames {
		result.AllNames[name] = struct{}{}
	}
	return result
}

// convertJSONToMetricLabelMap converts the JSON representation of MetricLabelMap back to the original type.
func convertJSONToMetricLabelMap(data MapForJSON) MetricLabelMap {
	result := make(MetricLabelMap)
	for metric, labelsRaw := range data {
		labels := labelsRaw.(map[string]interface{})
		result[metric] = MetricInfo{
			Labels: make(map[string]LabelInfo),
		}

		for label, valuesRaw := range labels {
			labelInfo, exists := result[metric].Labels[label]
			if !exists {
				labelInfo = LabelInfo{
					Values: make(map[string]struct{}),
				}
			}
			values := valuesRaw.([]interface{})
			for _, v := range values {
				value := fmt.Sprintf("%v", v) // Convert interface{} to string
				labelInfo.Values[value] = struct{}{}
			}
			result[metric].Labels[label] = labelInfo
		}
	}
	return result
}

// convertJSONToLabelValueMap converts the JSON representation of LabelValueMap back to the original type.
func convertJSONToLabelValueMap(data MapForJSON) LabelValueMap {
	result := make(LabelValueMap)
	for label, valuesRaw := range data {
		values := valuesRaw.([]interface{})
		labelInfo, exists := result[label]
		if !exists {
			labelInfo = LabelInfo{
				Values: make(map[string]struct{}),
			}
		}
		for _, v := range values {
			value := fmt.Sprintf("%v", v) // Convert interface{} to string
			labelInfo.Values[value] = struct{}{}
		}
		result[label] = labelInfo
	}
	return result
}
