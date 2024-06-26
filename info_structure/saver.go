package info_structure

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/prashantgupta17/nlpromql/config"
)

// MapForJSON represents a map that can be directly serialized to JSON.
type MapForJSON map[string]interface{}

// SaveInfoStructure saves all information structures to JSON files.
func SaveInfoStructure(metricMap MetricMap, labelMap LabelMap, metricLabelMap MetricLabelMap,
	labelValueMap LabelValueMap, nlpToMetricMap NlpToMetricMap) error {

	if err := saveMapToFile(config.MetricMapFile, metricMap); err != nil {
		return err
	}
	if err := saveMapToFile(config.LabelMapFile, labelMap); err != nil {
		return err
	}
	metricLabelMapJSON := convertMetricLabelMapToLists(metricLabelMap)
	if err := saveMapToFile(config.MetricLabelMapFile, metricLabelMapJSON); err != nil {
		return err
	}
	labelValueMapJSON := convertLabelValueMapToLists(labelValueMap)
	if err := saveMapToFile(config.LabelValueMapFile, labelValueMapJSON); err != nil {
		return err
	}
	if err := saveMapToFile(config.NlpToMetricMapFile, nlpToMetricMap); err != nil {
		return err
	}
	return nil
}

// saveMapToFile saves a map to a JSON file.
func saveMapToFile(filePath string, data interface{}) error {
	fmt.Println("Saving:", filePath)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Indent for readability
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error encoding JSON: %v", err)
	}

	return nil
}

// convertMetricLabelMapToLists converts the label values in a MetricLabelMap from sets to lists for saving.
func convertMetricLabelMapToLists(metricLabelMap MetricLabelMap) MapForJSON {
	result := make(MapForJSON)
	for metric, labels := range metricLabelMap {
		result[metric] = make(MapForJSON)
		for label, values := range labels {
			listValues := make([]string, 0, len(values))
			for v := range values {
				listValues = append(listValues, v)
			}
			result[metric].(MapForJSON)[label] = listValues
		}
	}
	return result
}

// convertLabelValueMapToLists converts the values in a LabelValueMap from sets to lists for saving.
func convertLabelValueMapToLists(labelValueMap LabelValueMap) MapForJSON {
	result := make(MapForJSON)
	for label, values := range labelValueMap {
		listValues := make([]string, 0, len(values))
		for v := range values {
			listValues = append(listValues, v)
		}
		result[label] = listValues
	}
	return result
}
